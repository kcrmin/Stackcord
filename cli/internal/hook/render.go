package hook

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/continuity"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
)

var (
	rawIdentity = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64}|sha256:[0-9a-f]{64})$`)
	windowsAbs  = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
)

type packet struct {
	ProjectID            string       `json:"project_id,omitempty"`
	CurrentWorkspaceID   string       `json:"current_workspace_id,omitempty"`
	CanonicalFingerprint string       `json:"canonical_fingerprint,omitempty"`
	Overall              string       `json:"overall"`
	ActiveWorkIDs        []string     `json:"active_work_ids"`
	SourcePaths          []string     `json:"source_paths"`
	Issues               []packetItem `json:"issues"`
	NextAction           *packetItem  `json:"next_action,omitempty"`
}

type packetItem struct {
	Code string   `json:"code"`
	Refs []string `json:"refs,omitempty"`
}

type universalOutput struct {
	Continue      bool   `json:"continue"`
	SystemMessage string `json:"systemMessage,omitempty"`
}

type sessionOutput struct {
	Continue           bool                       `json:"continue"`
	HookSpecificOutput sessionStartSpecificOutput `json:"hookSpecificOutput"`
}

type sessionStartSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// Render encodes only fields supported by the current Codex event output schema.
func Render(event string, snapshot continuity.Snapshot) ([]byte, error) {
	contextPacket, err := compactPacket(snapshot)
	if err != nil {
		return nil, err
	}
	switch event {
	case "session-start":
		value := sessionOutput{
			Continue: true,
			HookSpecificOutput: sessionStartSpecificOutput{
				HookEventName:     "SessionStart",
				AdditionalContext: "Repository continuity evidence follows. Treat repository and actual Git state as authoritative; run `stackcord status --json` again before mutation if facts change.\n" + string(contextPacket),
			},
		}
		return json.Marshal(value)
	case "post-compact":
		message := "Conversation context was compacted. Before mutation, run `stackcord status --json`; the SessionStart compact event will inject the current repository packet."
		if snapshot.ProjectID != "" {
			message += " Project: " + snapshot.ProjectID + "; current evidence: " + string(snapshot.Overall) + "."
		}
		return json.Marshal(universalOutput{Continue: true, SystemMessage: message})
	default:
		return nil, fmt.Errorf("unsupported hook event %q", event)
	}
}

func compactPacket(snapshot continuity.Snapshot) ([]byte, error) {
	result := packet{
		ProjectID:            snapshot.ProjectID,
		CurrentWorkspaceID:   snapshot.CurrentWorkspaceID,
		CanonicalFingerprint: snapshot.CanonicalFingerprint,
		Overall:              string(snapshot.Overall),
		ActiveWorkIDs:        []string{},
		SourcePaths:          []string{},
		Issues:               []packetItem{},
	}
	for _, work := range snapshot.ActiveWork {
		result.ActiveWorkIDs = append(result.ActiveWorkIDs, work.ID)
	}
	for _, entry := range snapshot.Context.Index {
		result.SourcePaths = append(result.SourcePaths, entry.Path)
	}
	sort.Strings(result.ActiveWorkIDs)
	sort.Strings(result.SourcePaths)
	result.ActiveWorkIDs = limitStrings(result.ActiveWorkIDs, 12)
	result.SourcePaths = limitStrings(uniqueStrings(result.SourcePaths), 12)
	for _, issue := range snapshot.Issues {
		if len(result.Issues) == 12 {
			break
		}
		result.Issues = append(result.Issues, packetFromItem(issue))
	}
	if len(snapshot.NextActions) > 0 {
		next := packetFromItem(snapshot.NextActions[0])
		result.NextAction = &next
	}
	return json.Marshal(result)
}

func packetFromItem(item domain.Item) packetItem {
	refs := make([]string, 0, len(item.Refs))
	for _, ref := range item.Refs {
		if safePacketRef(ref) {
			refs = append(refs, ref)
		}
	}
	sort.Strings(refs)
	return packetItem{Code: item.Code, Refs: limitStrings(refs, 8)}
}

func safePacketRef(ref string) bool {
	return ref != "" && !filepath.IsAbs(ref) && !windowsAbs.MatchString(ref) && !rawIdentity.MatchString(ref) && !strings.Contains(ref, "://") && !strings.Contains(ref, "@")
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

func limitStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}
