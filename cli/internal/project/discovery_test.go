package project_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/project"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"github.com/stretchr/testify/require"
)

func discoveryFixture() project.DiscoveryCheckpoint {
	c := project.ExampleDiscoveryCheckpoint()
	c.Decisions[0].QuestionID = "question.scope"
	c.OpenQuestions = []project.DiscoveryFact{
		{ID: "question.locale", Summary: "Default language?"},
		{ID: "question.theme", Summary: "Default theme?"},
		{ID: "question.security", Summary: "Disable policy approval?"},
		{ID: "question.retention", Summary: "Retention after security choice?"},
	}
	c.Discovery = &project.DiscoveryPlan{ScopeReady: true, Sections: []project.DiscoverySection{
		{ID: "section.scope", Title: "Scope", Status: "complete"},
		{ID: "section.preferences", Title: "Preferences", Status: "active", EstimatedRemaining: 3},
		{ID: "section.policy", Title: "Policy", Status: "planned", EstimatedRemaining: 3},
	}, Questions: []project.DiscoveryQuestion{
		{QuestionID: "question.locale", SectionID: "section.preferences", Options: []project.DiscoveryOption{{ID: "en", Label: "English"}, {ID: "ko", Label: "Korean"}}, RecommendedOptionID: "ko", DependsOn: []string{"question.scope"}},
		{QuestionID: "question.theme", SectionID: "section.preferences", Options: []project.DiscoveryOption{{ID: "system", Label: "System"}, {ID: "light", Label: "Light"}}, RecommendedOptionID: "system"},
		{QuestionID: "question.security", SectionID: "section.policy", RequiresExplicitAnswer: true, Blocking: true},
		{QuestionID: "question.retention", SectionID: "section.policy", DependsOn: []string{"question.security"}},
	}}
	return c
}

func TestDiscoveryBatchesOnlyIndependentRoutineQuestions(t *testing.T) {
	c := discoveryFixture()
	r, err := project.SummarizeDiscovery(c)
	require.NoError(t, err)
	require.True(t, r.ProgressKnown)
	require.Equal(t, "section.preferences", r.CurrentSection.ID)
	require.Equal(t, 4, r.RemainingMin)
	require.Equal(t, 6, r.RemainingMax)
	require.Len(t, r.Batch, 2)
	require.Equal(t, "question.locale", r.Batch[0].QuestionID)
	require.Len(t, r.Decisions, 1, "recommended answers must not become decisions")
	require.False(t, r.HarnessReady)
	require.Equal(t, []string{"question.security"}, r.BlockingQuestions)
	c.Discovery.Sections[1].Status = "planned"
	c.Discovery.Sections[2].Status = "active"
	r, err = project.SummarizeDiscovery(c)
	require.NoError(t, err)
	require.Empty(t, r.Batch)
	require.Len(t, r.ExplicitQuestions, 1)
	require.Equal(t, "question.security", r.ExplicitQuestions[0].QuestionID)
}

func TestDiscoveryRejectsContradictoryOrUnsafePlans(t *testing.T) {
	cases := map[string]func(*project.DiscoveryCheckpoint){
		"settled question reopened": func(c *project.DiscoveryCheckpoint) { c.Decisions[0].QuestionID = "question.locale" },
		"unknown section":           func(c *project.DiscoveryCheckpoint) { c.Discovery.Questions[0].SectionID = "section.missing" },
		"missing question plan":     func(c *project.DiscoveryCheckpoint) { c.Discovery.Questions = c.Discovery.Questions[1:] },
		"duplicate question plan":   func(c *project.DiscoveryCheckpoint) { c.Discovery.Questions[1] = c.Discovery.Questions[0] },
		"unknown option":            func(c *project.DiscoveryCheckpoint) { c.Discovery.Questions[0].RecommendedOptionID = "missing" },
		"unknown dependency": func(c *project.DiscoveryCheckpoint) {
			c.Discovery.Questions[0].DependsOn = []string{"question.missing"}
		},
		"dependency cycle": func(c *project.DiscoveryCheckpoint) {
			c.Discovery.Questions[2].DependsOn = []string{"question.retention"}
		},
		"self dependency":              func(c *project.DiscoveryCheckpoint) { c.Discovery.Questions[0].DependsOn = []string{"question.locale"} },
		"underestimated count":         func(c *project.DiscoveryCheckpoint) { c.Discovery.Sections[1].EstimatedRemaining = 1 },
		"complete with open questions": func(c *project.DiscoveryCheckpoint) { c.Discovery.Sections[1].Status = "complete" },
		"complete with estimate":       func(c *project.DiscoveryCheckpoint) { c.Discovery.Sections[0].EstimatedRemaining = 1 },
		"multiple active sections":     func(c *project.DiscoveryCheckpoint) { c.Discovery.Sections[2].Status = "active" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			c := discoveryFixture()
			change(&c)
			_, err := project.SummarizeDiscovery(c)
			require.Error(t, err)
		})
	}
}

func TestDiscoveryRoundTripAndLegacyReadiness(t *testing.T) {
	c := discoveryFixture()
	parent := t.TempDir()
	p, err := project.PlanCheckpoint(project.CheckpointRequest{Parent: parent, DraftID: "01JPROGRESS", Locale: "ko", Checkpoint: c})
	require.NoError(t, err)
	require.Equal(t, domain.StatusPassed, operation.Apply(context.Background(), p).Status)
	saved, err := schema.LoadYAML[project.DiscoveryCheckpoint](filepath.Join(parent, ".harness-drafts", "01JPROGRESS", "checkpoint.yaml"))
	require.NoError(t, err)
	require.Equal(t, c, saved)
	saved.Discovery = nil
	r, err := project.SummarizeDiscovery(saved)
	require.NoError(t, err)
	require.False(t, r.ProgressKnown)
	require.False(t, r.HarnessReady)
	require.Len(t, r.Decisions, 1)
	c.OpenQuestions = []project.DiscoveryFact{}
	c.Discovery.Questions = []project.DiscoveryQuestion{}
	for i := range c.Discovery.Sections {
		c.Discovery.Sections[i].Status = "complete"
		c.Discovery.Sections[i].EstimatedRemaining = 0
	}
	r, err = project.SummarizeDiscovery(c)
	require.NoError(t, err)
	require.True(t, r.HarnessReady)
	c.Discovery.ScopeReady = false
	r, err = project.SummarizeDiscovery(c)
	require.NoError(t, err)
	require.False(t, r.HarnessReady)
}

func TestDiscoverySurfacesCrossSectionPrerequisites(t *testing.T) {
	c := discoveryFixture()
	c.Discovery.Questions[0].DependsOn = []string{"question.security"}
	c.Discovery.Questions[1].DependsOn = []string{"question.security"}
	r, err := project.SummarizeDiscovery(c)
	require.NoError(t, err)
	require.Empty(t, r.Batch)
	require.Len(t, r.Prerequisites, 1)
	require.Equal(t, "question.security", r.Prerequisites[0].QuestionID)
}

func TestDiscoverySharedDependencyGraph(t *testing.T) {
	c := discoveryFixture()
	c.OpenQuestions = []project.DiscoveryFact{}
	c.Discovery.Questions = []project.DiscoveryQuestion{}
	c.Discovery.Sections[1].EstimatedRemaining = 1
	c.Discovery.Sections[2].EstimatedRemaining = 44
	for i := 0; i < 45; i++ {
		id := fmt.Sprintf("question.item-%d", i)
		c.OpenQuestions = append(c.OpenQuestions, project.DiscoveryFact{ID: id, Summary: id})
		q := project.DiscoveryQuestion{QuestionID: id, SectionID: "section.policy"}
		if i > 0 {
			q.DependsOn = append(q.DependsOn, fmt.Sprintf("question.item-%d", i-1))
		}
		if i > 1 {
			q.DependsOn = append(q.DependsOn, fmt.Sprintf("question.item-%d", i-2))
		}
		if i == 44 {
			q.SectionID = "section.preferences"
		}
		c.Discovery.Questions = append(c.Discovery.Questions, q)
	}
	r, err := project.SummarizeDiscovery(c)
	require.NoError(t, err)
	require.Len(t, r.Prerequisites, 1)
	require.Equal(t, "question.item-0", r.Prerequisites[0].QuestionID)
}
