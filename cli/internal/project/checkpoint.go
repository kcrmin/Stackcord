package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/schema"
	"go.yaml.in/yaml/v3"
)

// DiscoveryFact is one normalized, stable piece of product meaning.
type DiscoveryFact struct {
	ID      string `json:"id" yaml:"id"`
	Summary string `json:"summary" yaml:"summary"`
}

// DiscoveryScenario records observable success and failure behavior.
type DiscoveryScenario struct {
	ID      string `json:"id" yaml:"id"`
	Actor   string `json:"actor" yaml:"actor"`
	Trigger string `json:"trigger" yaml:"trigger"`
	Outcome string `json:"outcome" yaml:"outcome"`
	Failure string `json:"failure" yaml:"failure"`
}

// UICoverage links one role and journey to required user-visible states.
type UICoverage struct {
	ID        string   `json:"id" yaml:"id"`
	RoleID    string   `json:"role_id" yaml:"role_id"`
	JourneyID string   `json:"journey_id" yaml:"journey_id"`
	States    []string `json:"states" yaml:"states"`
}

// DiscoveryDecision keeps a normalized choice and rationale, never raw conversation.
type DiscoveryDecision struct {
	ID        string `json:"id" yaml:"id"`
	Choice    string `json:"choice" yaml:"choice"`
	Rationale string `json:"rationale" yaml:"rationale"`
}

// DiscoveryCheckpoint is the complete replaceable snapshot saved after a material answer.
type DiscoveryCheckpoint struct {
	SchemaVersion   int                 `json:"schema_version" yaml:"schema_version"`
	Summary         string              `json:"summary" yaml:"summary"`
	CurrentFocus    string              `json:"current_focus" yaml:"current_focus"`
	Roles           []DiscoveryFact     `json:"roles" yaml:"roles"`
	Journeys        []DiscoveryFact     `json:"journeys" yaml:"journeys"`
	Capabilities    []DiscoveryFact     `json:"capabilities" yaml:"capabilities"`
	Policies        []DiscoveryFact     `json:"policies" yaml:"policies"`
	Scenarios       []DiscoveryScenario `json:"scenarios" yaml:"scenarios"`
	Quality         []DiscoveryFact     `json:"quality" yaml:"quality"`
	UICoverage      []UICoverage        `json:"ui_coverage" yaml:"ui_coverage"`
	TechnologyNeeds []DiscoveryFact     `json:"technology_needs" yaml:"technology_needs"`
	Decisions       []DiscoveryDecision `json:"decisions" yaml:"decisions"`
	Assumptions     []DiscoveryFact     `json:"assumptions" yaml:"assumptions"`
	OpenQuestions   []DiscoveryFact     `json:"open_questions" yaml:"open_questions"`
}

// CheckpointRequest identifies one long-running discovery and its normalized current snapshot.
type CheckpointRequest struct {
	Parent, DraftID, Locale string
	Checkpoint              DiscoveryCheckpoint
}

type checkpointState struct {
	SchemaVersion int       `yaml:"schema_version"`
	Revision      int       `yaml:"revision"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	CurrentFocus  string    `yaml:"current_focus"`
}

// PlanCheckpoint validates and plans the next atomic revision of one discovery draft.
func PlanCheckpoint(request CheckpointRequest) (operation.Plan, error) {
	if request.Parent == "" || !draftIDPattern.MatchString(request.DraftID) || (request.Locale != "en" && request.Locale != "ko") {
		return operation.Plan{}, fmt.Errorf("parent, safe draft ID, and locale en|ko are required")
	}
	if err := validateCheckpoint(request.Checkpoint); err != nil {
		return operation.Plan{}, err
	}
	root := filepath.Join(request.Parent, ".harness-drafts", request.DraftID)
	revision := 1
	statePath := filepath.Join(root, "state.yaml")
	if state, err := schema.LoadYAML[checkpointState](statePath); err == nil {
		if state.SchemaVersion != 1 || state.Revision < 1 {
			return operation.Plan{}, fmt.Errorf("existing discovery state is invalid")
		}
		revision = state.Revision + 1
	} else if !errors.Is(err, os.ErrNotExist) {
		return operation.Plan{}, err
	}
	now := time.Now().UTC()
	files, err := checkpointFiles(request, revision, now)
	if err != nil {
		return operation.Plan{}, err
	}
	plan := operation.Plan{ID: fmt.Sprintf("checkpoint-%s-r%d", request.DraftID, revision), Root: root, Files: files}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}

func validateCheckpoint(checkpoint DiscoveryCheckpoint) error {
	if issues := schema.Validate("discovery", checkpoint); len(issues) > 0 {
		return fmt.Errorf("validate normalized discovery: %s", issues[0].Message)
	}
	if checkpoint.SchemaVersion != 1 || strings.TrimSpace(checkpoint.Summary) == "" {
		return fmt.Errorf("discovery schema version 1 and summary are required")
	}
	seen := map[string]bool{}
	validateID := func(id, summary string) error {
		if !projectIDPattern.MatchString(id) {
			return fmt.Errorf("stable ID is invalid: %q", id)
		}
		if seen[id] {
			return fmt.Errorf("stable ID is duplicated: %s", id)
		}
		if strings.TrimSpace(summary) == "" {
			return fmt.Errorf("normalized summary is required for %s", id)
		}
		seen[id] = true
		return nil
	}
	for _, group := range [][]DiscoveryFact{checkpoint.Roles, checkpoint.Journeys, checkpoint.Capabilities, checkpoint.Policies, checkpoint.Quality, checkpoint.TechnologyNeeds, checkpoint.Assumptions, checkpoint.OpenQuestions} {
		for _, fact := range group {
			if err := validateID(fact.ID, fact.Summary); err != nil {
				return err
			}
		}
	}
	for _, scenario := range checkpoint.Scenarios {
		if err := validateID(scenario.ID, scenario.Outcome); err != nil {
			return err
		}
		if scenario.Actor == "" || scenario.Trigger == "" || scenario.Failure == "" {
			return fmt.Errorf("scenario %s needs actor, trigger, outcome, and failure", scenario.ID)
		}
	}
	for _, coverage := range checkpoint.UICoverage {
		if err := validateID(coverage.ID, strings.Join(coverage.States, ",")); err != nil {
			return err
		}
		if coverage.RoleID == "" || coverage.JourneyID == "" || len(coverage.States) == 0 {
			return fmt.Errorf("UI coverage %s needs role, journey, and states", coverage.ID)
		}
	}
	for _, decision := range checkpoint.Decisions {
		if err := validateID(decision.ID, decision.Choice); err != nil {
			return err
		}
		if decision.Rationale == "" {
			return fmt.Errorf("decision %s needs a rationale", decision.ID)
		}
	}
	return nil
}

func checkpointFiles(request CheckpointRequest, revision int, now time.Time) ([]operation.FileChange, error) {
	encode := func(value any) ([]byte, error) { return yaml.Marshal(value) }
	manifest, err := encode(map[string]any{"schema_version": 1, "id": "draft." + request.DraftID, "locale": request.Locale})
	if err != nil {
		return nil, err
	}
	state, err := encode(checkpointState{SchemaVersion: 1, Revision: revision, UpdatedAt: now, CurrentFocus: request.Checkpoint.CurrentFocus})
	if err != nil {
		return nil, err
	}
	checkpoint, err := encode(request.Checkpoint)
	if err != nil {
		return nil, err
	}
	wrap := func(key string, value any) ([]byte, error) {
		return encode(map[string]any{"schema_version": 1, key: value})
	}
	values := map[string]any{
		"roles": request.Checkpoint.Roles, "journeys": request.Checkpoint.Journeys, "capabilities": request.Checkpoint.Capabilities,
		"policies": request.Checkpoint.Policies, "scenarios": request.Checkpoint.Scenarios, "quality": request.Checkpoint.Quality,
		"ui-coverage": request.Checkpoint.UICoverage, "technology-needs": request.Checkpoint.TechnologyNeeds,
		"decisions": request.Checkpoint.Decisions, "assumptions": request.Checkpoint.Assumptions, "open-questions": request.Checkpoint.OpenQuestions,
	}
	files := []operation.FileChange{
		{Path: "manifest.yaml", Content: manifest, Mode: 0o600},
		{Path: "state.yaml", Content: state, Mode: 0o600},
		{Path: "checkpoint.yaml", Content: checkpoint, Mode: 0o600},
		{Path: "specs/product/summary.md", Content: []byte("# Normalized product summary\n\n" + strings.TrimSpace(request.Checkpoint.Summary) + "\n"), Mode: 0o600},
	}
	for name, value := range values {
		data, err := wrap(strings.ReplaceAll(name, "-", "_"), value)
		if err != nil {
			return nil, err
		}
		files = append(files, operation.FileChange{Path: "specs/product/" + name + ".yaml", Content: data, Mode: 0o600})
	}
	return files, nil
}
