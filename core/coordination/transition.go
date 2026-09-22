package coordination

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

func validateMutation(m Mutation, raw []byte) error {
	invalid := func(why string) error { return fmt.Errorf("%w: %s", ErrInvalid, why) }
	if !identifier.MatchString(m.ID) || !identifier.MatchString(m.Project) || !identifier.MatchString(m.Task) {
		return invalid("IDs must be 1-128 letters, digits, dots, underscores or hyphens")
	}
	if len(m.Reason) > 1024 {
		return invalid("reason exceeds 1024 bytes")
	}
	if m.Definition != nil {
		b, _ := json.Marshal(m.Definition)
		if len(b) > 32<<10 {
			return invalid("definition exceeds 32 KiB; reference large inputs")
		}
	}
	if m.Checkpoint != nil {
		b, _ := json.Marshal(m.Checkpoint)
		if len(b) > 24<<10 {
			return invalid("checkpoint exceeds 24 KiB; reference large artifacts")
		}
	}
	if len(raw) > MaxMutationBytes {
		return invalid("mutation exceeds 64 KiB")
	}
	if m.Usage != nil && (m.Usage.InputTokens < 0 || m.Usage.OutputTokens < 0) {
		return invalid("usage cannot be negative")
	}
	if m.LeaseSeconds < 0 || m.LeaseSeconds > 3600 {
		return invalid("lease must be at most 3600 seconds")
	}
	if m.Checkpoint != nil && (len(m.Checkpoint.Artifacts) > 32 || len(m.Checkpoint.Blockers) > 32) {
		return invalid("too many checkpoint entries")
	}
	switch m.Kind {
	case "create":
		if m.Definition == nil || strings.TrimSpace(m.Definition.Goal) == "" {
			return invalid("goal required")
		}
		if len(m.Definition.Dependencies) > 32 || len(m.Definition.Inputs) > 32 || len(m.Definition.Constraints) > 64 {
			return invalid("too many definition entries")
		}
		seen := map[string]bool{}
		for _, id := range m.Definition.Dependencies {
			if !identifier.MatchString(id) || id == m.Task || seen[id] {
				return invalid("invalid or duplicate dependency")
			}
			seen[id] = true
		}
		if m.Owner != "" || m.Session != "" || m.Epoch != 0 || m.LeaseSeconds != 0 || m.ExpectedRevision != 0 || m.Reason != "" || m.Usage != nil || m.Checkpoint != nil {
			return invalid("create only accepts a definition")
		}
	case "claim":
		if !identifier.MatchString(m.Owner) || !identifier.MatchString(m.Session) || m.LeaseSeconds < 1 {
			return invalid("owner, session and 1-3600 second lease required")
		}
		if m.Definition != nil || m.Checkpoint != nil || m.Usage != nil || m.Reason != "" || m.Epoch != 0 || m.ExpectedRevision != 0 {
			return invalid("unexpected claim fields")
		}
	case "requeue":
		if m.ExpectedRevision == 0 {
			return invalid("expected_revision required")
		}
		if m.Owner != "" || m.Session != "" || m.Epoch != 0 || m.Definition != nil || m.Checkpoint != nil || m.Usage != nil || m.LeaseSeconds != 0 || m.Reason != "" {
			return invalid("unexpected requeue fields")
		}
	case "checkpoint", "pause", "complete", "fail":
		if !identifier.MatchString(m.Owner) || !identifier.MatchString(m.Session) || m.Epoch == 0 {
			return invalid("owner, session and epoch required")
		}
		if m.Definition != nil || m.ExpectedRevision != 0 {
			return invalid("definition is immutable")
		}
		if m.Kind == "checkpoint" && m.LeaseSeconds < 1 {
			return invalid("checkpoint requires a renewed lease")
		}
		if m.Kind != "checkpoint" && m.LeaseSeconds != 0 {
			return invalid("terminal updates cannot renew lease")
		}
		if (m.Kind == "pause" || m.Kind == "fail") && strings.TrimSpace(m.Reason) == "" {
			return invalid("reason required")
		}
		if (m.Kind == "checkpoint" || m.Kind == "complete") && m.Reason != "" {
			return invalid("unexpected reason")
		}
	default:
		return invalid("unknown mutation kind")
	}
	return nil
}
func transition(p *bolt.Bucket, t *Task, m Mutation, now time.Time) error {
	switch m.Kind {
	case "claim":
		if t.State == "paused" && t.Reason == "approval_required" {
			return fmt.Errorf("%w: explicit requeue required", ErrBlocked)
		}
		if t.State != "ready" && t.State != "paused" && !(t.State == "running" && !now.Before(t.LeaseUntil)) {
			return ErrConflict
		}
		for _, id := range t.Definition.Dependencies {
			dep, e := readTask(p, id)
			if e != nil {
				return e
			}
			if dep.State != "completed" {
				return fmt.Errorf("%w: dependency %s is %s", ErrBlocked, id, dep.State)
			}
		}
		t.Owner = m.Owner
		t.Session = m.Session
		t.Epoch++
		t.LeaseUntil = now.Add(time.Duration(m.LeaseSeconds) * time.Second)
		t.State = "running"
		t.Reason = ""
	case "requeue":
		if t.Revision != m.ExpectedRevision || (t.State != "paused" && t.State != "failed") {
			return ErrConflict
		}
		t.State = "ready"
		t.Reason = ""
		t.LeaseUntil = time.Time{}
	default:
		if t.State != "running" || t.Owner != m.Owner || t.Session != m.Session || t.Epoch != m.Epoch || !now.Before(t.LeaseUntil) {
			return ErrConflict
		}
		switch m.Kind {
		case "checkpoint":
			t.LeaseUntil = now.Add(time.Duration(m.LeaseSeconds) * time.Second)
		case "pause":
			t.State = "paused"
			t.Reason = m.Reason
			t.LeaseUntil = time.Time{}
		case "complete":
			t.State = "completed"
			t.LeaseUntil = time.Time{}
		case "fail":
			t.State = "failed"
			t.Reason = m.Reason
			t.LeaseUntil = time.Time{}
		}
	}
	return nil
}
