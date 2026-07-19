package project

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"go.yaml.in/yaml/v3"
)

// DraftRequest stores normalized discovery results before a repository name is approved.
type DraftRequest struct {
	Parent        string
	DraftID       string
	Locale        string
	Summary       string
	Decisions     []string
	OpenQuestions []string
}

// CreateDraft creates a durable normalized discovery checkpoint, never raw conversation.
func CreateDraft(request DraftRequest) (operation.Plan, error) {
	if request.Parent == "" || request.DraftID == "" || (request.Locale != "en" && request.Locale != "ko") {
		return operation.Plan{}, fmt.Errorf("parent, draft ID, and locale en|ko are required")
	}
	if !draftIDPattern.MatchString(request.DraftID) {
		return operation.Plan{}, fmt.Errorf("draft ID must contain only letters, digits, underscores, or hyphens")
	}
	root := filepath.Join(request.Parent, ".harness-drafts", request.DraftID)
	now := time.Now().UTC().Format(time.RFC3339)
	manifest, _ := yaml.Marshal(map[string]any{"schema_version": 1, "id": "draft." + request.DraftID, "created_at": now, "locale": request.Locale, "parent": request.Parent})
	state, _ := yaml.Marshal(map[string]any{"schema_version": 1, "stage": "service_discovery", "last_saved_at": now, "next": "resolve_open_question"})
	decisions, _ := yaml.Marshal(map[string]any{"schema_version": 1, "decisions": request.Decisions})
	questions, _ := yaml.Marshal(map[string]any{"schema_version": 1, "open_questions": request.OpenQuestions})
	plan := operation.Plan{ID: "draft-" + request.DraftID, Root: root, Files: []operation.FileChange{
		{Path: "manifest.yaml", Content: manifest, Mode: 0o600},
		{Path: "state.yaml", Content: state, Mode: 0o600},
		{Path: "specs/product/summary.md", Content: []byte("# Normalized product summary\n\n" + strings.TrimSpace(request.Summary) + "\n"), Mode: 0o600},
		{Path: "specs/product/decisions.yaml", Content: decisions, Mode: 0o600},
		{Path: "specs/product/open-questions.yaml", Content: questions, Mode: 0o600},
	}}
	fingerprint, err := operation.StateFingerprint(plan)
	plan.InitialStateFingerprint = fingerprint
	return plan, err
}
