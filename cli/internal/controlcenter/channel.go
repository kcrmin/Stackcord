package controlcenter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/channel"
)

func (b *Backend) channelSnapshot(ctx context.Context, state map[string]any) {
	s, err := channel.Open(b.Root)
	if err != nil {
		state["channel"] = map[string]any{"configured": false, "error": err.Error()}
		return
	}
	current, err := s.State(ctx, false)
	if err != nil {
		state["channel"] = map[string]any{"configured": false, "error": err.Error()}
		return
	}
	state["channel"] = current
	state["channel_revision"] = current.Revision
}

// Channel actions never configure or invoke local executables through HTTP.
// All displayed message bodies are untrusted; signed results are not approvals.
func (b *Backend) channelAction(ctx context.Context, kind string, payload json.RawMessage) (any, error) {
	var p struct {
		ExpectedRevision string   `json:"expected_revision"`
		Apply            bool     `json:"apply"`
		Channel          string   `json:"channel"`
		Remote           string   `json:"remote"`
		Peer             string   `json:"peer"`
		PublicKey        string   `json:"public_key"`
		To               string   `json:"to"`
		Kind             string   `json:"kind"`
		Title            string   `json:"title"`
		Body             string   `json:"body"`
		Dependencies     []string `json:"dependencies"`
		Scope            []string `json:"scope"`
		RequestID        string   `json:"request_id"`
		Status           string   `json:"status"`
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("invalid channel action payload")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("invalid trailing payload")
	}
	s, err := channel.Open(b.Root)
	if err != nil {
		return nil, err
	}
	current, err := s.State(ctx, false)
	if err != nil {
		return nil, err
	}
	if kind == "channel.refresh" {
		current, err = s.State(ctx, true)
		if err != nil {
			return nil, err
		}
		return map[string]any{"state": current, "summary": "Channel refreshed from the shared remote.", "message": "Channel refreshed from the shared remote."}, nil
	}
	if p.ExpectedRevision == "" || p.ExpectedRevision != current.Revision {
		return nil, fmt.Errorf("channel state changed; refresh and preview again")
	}
	s.ExpectedRevision = p.ExpectedRevision
	var changes any
	var summary string
	switch kind {
	case "channel.setup":
		if strings.TrimSpace(p.Channel) == "" || strings.TrimSpace(p.Peer) == "" || strings.TrimSpace(p.Remote) == "" {
			return nil, fmt.Errorf("channel, remote and peer are required")
		}
		changes = map[string]string{"channel": p.Channel, "remote": p.Remote, "peer": p.Peer}
		summary = "Create this computer's local signing identity. Share only the public key with your collaborators. No worker is started."
	case "channel.trust":
		if p.Peer == "" || p.PublicKey == "" {
			return nil, fmt.Errorf("peer and public key are required")
		}
		changes = map[string]string{"peer": p.Peer, "public_key": p.PublicKey}
		summary = "Trust this verified peer key for future messages. A configured runner may automatically process allowed requests from this peer."
	case "channel.send":
		if p.To == "" || p.Kind == "" || strings.TrimSpace(p.Title) == "" {
			return nil, fmt.Errorf("recipient, kind and title are required")
		}
		changes = channel.RequestInput{To: p.To, Kind: p.Kind, Title: p.Title, Body: p.Body, Dependencies: p.Dependencies, Scope: p.Scope}
		summary = "Publish this signed request to the shared channel. Registered workers may process it automatically."
	case "channel.respond":
		if p.RequestID == "" || (p.Status != "success" && p.Status != "failed") {
			return nil, fmt.Errorf("request and success/failed status are required")
		}
		changes = channel.ResultInput{RequestID: p.RequestID, Status: p.Status, Body: p.Body}
		summary = "Publish a result for a request addressed to this peer. This does not approve policy or create release evidence."
	case "channel.retry":
		if p.RequestID == "" {
			return nil, fmt.Errorf("request ID is required")
		}
		changes = map[string]string{"request_id": p.RequestID}
		summary = "Permit retry after inspecting partial effects of the interrupted execution."
	default:
		return nil, fmt.Errorf("unsupported channel action")
	}
	if !p.Apply {
		return map[string]any{"preview": changes, "changes": changes, "summary": summary, "message": summary, "requiresConfirmation": true}, nil
	}
	var result any
	switch kind {
	case "channel.setup":
		result, err = s.Setup(channel.Config{Channel: p.Channel, Remote: p.Remote, Peer: p.Peer})
	case "channel.trust":
		result, err = s.Trust(p.Peer, p.PublicKey)
	case "channel.send":
		result, err = s.Send(ctx, changes.(channel.RequestInput))
	case "channel.respond":
		result, err = s.Respond(ctx, changes.(channel.ResultInput))
	case "channel.retry":
		result, err = s.Retry(ctx, p.RequestID)
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"result": result, "summary": "Channel operation completed.", "message": "Channel operation completed."}, nil
}
