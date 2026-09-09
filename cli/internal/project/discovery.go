package project

import (
	"fmt"
	"sort"
	"strings"
)

// DiscoveryPlan is conversation coordination, not an approval of product policy.
// Estimates include known questions and anticipated follow-ups.
type DiscoveryPlan struct {
	ScopeReady bool                `json:"scope_ready" yaml:"scope_ready"`
	Sections   []DiscoverySection  `json:"sections" yaml:"sections"`
	Questions  []DiscoveryQuestion `json:"questions" yaml:"questions"`
}

type DiscoverySection struct {
	ID                 string `json:"id" yaml:"id"`
	Title              string `json:"title" yaml:"title"`
	Status             string `json:"status" yaml:"status"`
	EstimatedRemaining int    `json:"estimated_remaining" yaml:"estimated_remaining"`
}

type DiscoveryOption struct {
	ID    string `json:"id" yaml:"id"`
	Label string `json:"label" yaml:"label"`
}

// QuestionID points to canonical OpenQuestions; recommendations never answer it.
type DiscoveryQuestion struct {
	QuestionID             string            `json:"question_id" yaml:"question_id"`
	SectionID              string            `json:"section_id" yaml:"section_id"`
	Options                []DiscoveryOption `json:"options,omitempty" yaml:"options,omitempty"`
	RecommendedOptionID    string            `json:"recommended_option_id,omitempty" yaml:"recommended_option_id,omitempty"`
	RequiresExplicitAnswer bool              `json:"requires_explicit_answer" yaml:"requires_explicit_answer"`
	Blocking               bool              `json:"blocking" yaml:"blocking"`
	DependsOn              []string          `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
}

// DiscoveryReport can be consumed by any host without changing saved decisions.
type DiscoveryReport struct {
	Prerequisites              []DiscoveryQuestion
	ProgressKnown              bool
	CurrentSection             *DiscoverySection
	CompletedSections          []DiscoverySection
	RemainingSections          []DiscoverySection
	RemainingMin, RemainingMax int
	Batch, ExplicitQuestions   []DiscoveryQuestion
	Decisions                  []DiscoveryDecision
	BlockingQuestions          []string
	HarnessReady               bool
}

func validateDiscovery(c DiscoveryCheckpoint) error {
	open := map[string]bool{}
	for _, q := range c.OpenQuestions {
		open[q.ID] = true
	}
	accepted := map[string]bool{}
	for _, d := range c.Decisions {
		if d.QuestionID == "" {
			continue
		}
		if !projectIDPattern.MatchString(d.QuestionID) || open[d.QuestionID] || accepted[d.QuestionID] {
			return fmt.Errorf("decision %s has an invalid, duplicate or still-open question reference", d.ID)
		}
		accepted[d.QuestionID] = true
	}
	p := c.Discovery
	if p == nil {
		return nil
	}
	if len(p.Sections) == 0 {
		return fmt.Errorf("discovery needs at least one section")
	}
	sections := map[string]DiscoverySection{}
	active := 0
	for _, s := range p.Sections {
		if !projectIDPattern.MatchString(s.ID) || strings.TrimSpace(s.Title) == "" {
			return fmt.Errorf("discovery section needs a stable ID and title")
		}
		if _, exists := sections[s.ID]; exists {
			return fmt.Errorf("duplicate discovery section %s", s.ID)
		}
		if s.Status != "planned" && s.Status != "active" && s.Status != "complete" {
			return fmt.Errorf("invalid discovery section status %s", s.Status)
		}
		if s.EstimatedRemaining < 0 || s.EstimatedRemaining > 10000 {
			return fmt.Errorf("invalid remaining question estimate for %s", s.ID)
		}
		if s.Status == "complete" && s.EstimatedRemaining != 0 {
			return fmt.Errorf("completed section %s has a remaining estimate", s.ID)
		}
		if s.Status == "active" {
			active++
		}
		sections[s.ID] = s
	}
	if active > 1 {
		return fmt.Errorf("only one discovery section can be active")
	}
	questions := map[string]DiscoveryQuestion{}
	counts := map[string]int{}
	for _, q := range p.Questions {
		if !open[q.QuestionID] {
			return fmt.Errorf("question plan %s must reference an open question", q.QuestionID)
		}
		if _, exists := questions[q.QuestionID]; exists {
			return fmt.Errorf("duplicate question plan %s", q.QuestionID)
		}
		s, exists := sections[q.SectionID]
		if !exists || s.Status == "complete" {
			return fmt.Errorf("question %s needs an unfinished section", q.QuestionID)
		}
		if len(q.Options) == 1 || len(q.Options) > 3 {
			return fmt.Errorf("question %s needs either no options or two to three choices", q.QuestionID)
		}
		options := map[string]bool{}
		for _, o := range q.Options {
			if strings.TrimSpace(o.ID) == "" || strings.TrimSpace(o.Label) == "" || options[o.ID] {
				return fmt.Errorf("question %s has invalid or duplicate options", q.QuestionID)
			}
			options[o.ID] = true
		}
		if q.RecommendedOptionID != "" && !options[q.RecommendedOptionID] {
			return fmt.Errorf("question %s recommends an unknown option", q.QuestionID)
		}
		deps := map[string]bool{}
		for _, id := range q.DependsOn {
			if id == q.QuestionID || deps[id] || (!open[id] && !accepted[id]) {
				return fmt.Errorf("question %s has an invalid dependency %s", q.QuestionID, id)
			}
			deps[id] = true
		}
		questions[q.QuestionID] = q
		counts[q.SectionID]++
	}
	if len(questions) != len(open) {
		return fmt.Errorf("every open question needs exactly one question plan")
	}
	for _, s := range p.Sections {
		if s.EstimatedRemaining < counts[s.ID] {
			return fmt.Errorf("section %s estimate is below its known open question count", s.ID)
		}
	}
	visiting, done := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("discovery question dependency cycle at %s", id)
		}
		if done[id] || accepted[id] {
			return nil
		}
		visiting[id] = true
		for _, dependency := range questions[id].DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[id] = false
		done[id] = true
		return nil
	}
	for _, q := range p.Questions {
		if err := visit(q.QuestionID); err != nil {
			return err
		}
	}
	return nil
}

// SummarizeDiscovery derives pending actions. It never mutates or accepts answers.
func SummarizeDiscovery(c DiscoveryCheckpoint) (DiscoveryReport, error) {
	if err := validateCheckpoint(c); err != nil {
		return DiscoveryReport{}, err
	}
	r := DiscoveryReport{Decisions: append([]DiscoveryDecision{}, c.Decisions...), Batch: []DiscoveryQuestion{}, ExplicitQuestions: []DiscoveryQuestion{}, BlockingQuestions: []string{}, CompletedSections: []DiscoverySection{}, RemainingSections: []DiscoverySection{}}
	sort.Slice(r.Decisions, func(i, j int) bool { return r.Decisions[i].ID < r.Decisions[j].ID })
	p := c.Discovery
	if p == nil {
		return r, nil
	}
	r.ProgressKnown = true
	r.RemainingMin = len(c.OpenQuestions)
	for _, s := range p.Sections {
		r.RemainingMax += s.EstimatedRemaining
		if s.Status == "complete" {
			r.CompletedSections = append(r.CompletedSections, s)
		} else {
			r.RemainingSections = append(r.RemainingSections, s)
		}
		if s.Status == "active" {
			section := s
			r.CurrentSection = &section
		}
	}
	if r.CurrentSection == nil && len(r.RemainingSections) > 0 {
		section := r.RemainingSections[0]
		r.CurrentSection = &section
	}
	open := map[string]bool{}
	for _, q := range c.OpenQuestions {
		open[q.ID] = true
	}
	questions := map[string]DiscoveryQuestion{}
	for _, q := range p.Questions {
		questions[q.QuestionID] = q
	}
	prerequisites := map[string]bool{}
	visitedPrerequisites := map[string]bool{}
	var collectPrerequisites func(string)
	collectPrerequisites = func(id string) {
		if !open[id] || visitedPrerequisites[id] {
			return
		}
		visitedPrerequisites[id] = true
		q := questions[id]
		waiting := false
		for _, dep := range q.DependsOn {
			if open[dep] {
				waiting = true
				collectPrerequisites(dep)
			}
		}
		if !waiting && r.CurrentSection != nil && q.SectionID != r.CurrentSection.ID {
			prerequisites[id] = true
		}
	}
	for _, q := range p.Questions {
		if q.Blocking {
			r.BlockingQuestions = append(r.BlockingQuestions, q.QuestionID)
		}
		if r.CurrentSection == nil || q.SectionID != r.CurrentSection.ID {
			continue
		}
		waiting := false
		for _, dep := range q.DependsOn {
			if open[dep] {
				waiting = true
			}
		}
		if waiting {
			for _, dep := range q.DependsOn {
				collectPrerequisites(dep)
			}
			continue
		}
		if q.RequiresExplicitAnswer || len(q.Options) == 0 {
			r.ExplicitQuestions = append(r.ExplicitQuestions, q)
		} else {
			r.Batch = append(r.Batch, q)
		}
	}
	for _, q := range p.Questions {
		if prerequisites[q.QuestionID] {
			r.Prerequisites = append(r.Prerequisites, q)
		}
	}
	r.HarnessReady = p.ScopeReady && len(r.BlockingQuestions) == 0
	return r, nil
}
