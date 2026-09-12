package governance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"go.yaml.in/yaml/v3"
)

type AuthorityAction string

const (
	AuthorityAdd    AuthorityAction = "add"
	AuthorityRemove AuthorityAction = "remove"
)

var authoritySubjectPattern = regexp.MustCompile(`^(?:user|team):[A-Za-z0-9][A-Za-z0-9._/-]*$`)

type AuthorityChangeRequest struct {
	Root                      string
	Action                    AuthorityAction
	Subjects                  []string
	ExpectedPolicyFingerprint string
}

type AuthorityChange struct {
	Action            AuthorityAction `json:"action"`
	Subjects          []string        `json:"subjects"`
	PolicyFingerprint string          `json:"policy_fingerprint"`
	Before            []string        `json:"before"`
	After             []string        `json:"after"`
}

// PlanAuthorityChange prepares one exact governance-policy proposal without granting approval.
func PlanAuthorityChange(ctx context.Context, request AuthorityChangeRequest) (AuthorityChange, operation.Plan, error) {
	located, err := workspace.FindRoot(ctx, request.Root)
	if err != nil {
		return AuthorityChange{}, operation.Plan{}, err
	}
	request.Root = located.Path
	duplicate := ""
	seen := make(map[string]bool, len(request.Subjects))
	subjects := make([]string, 0, len(request.Subjects))
	for _, subject := range request.Subjects {
		subject = strings.TrimSpace(subject)
		if seen[subject] && duplicate == "" {
			duplicate = subject
		}
		seen[subject] = true
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)
	request.Subjects = subjects
	plan := operation.Plan{ID: authorityOperationID(request.Action, request.Subjects, ""), Root: request.Root}
	change := AuthorityChange{Action: request.Action, Subjects: append([]string(nil), request.Subjects...), Before: []string{}, After: []string{}}

	policy, err := LoadPolicy(request.Root)
	if err != nil {
		return AuthorityChange{}, operation.Plan{}, err
	}
	path := filepath.Join(request.Root, ".harness", "governance.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return AuthorityChange{}, operation.Plan{}, err
	}
	change.PolicyFingerprint = policyDigest(raw)
	plan.ID = authorityOperationID(request.Action, request.Subjects, change.PolicyFingerprint)
	change.Before = append([]string(nil), policy.ProductAuthorities...)
	change.After = append([]string(nil), change.Before...)

	block := func(code, message string, refs ...string) (AuthorityChange, operation.Plan, error) {
		plan.Files = nil
		plan.Blockers = append(plan.Blockers, domain.Item{Code: code, Message: message, Refs: refs})
		return change, plan, nil
	}
	if request.ExpectedPolicyFingerprint != "" && request.ExpectedPolicyFingerprint != change.PolicyFingerprint {
		return block("governance.policy-stale", "Governance policy changed after the authority plan was reviewed.", request.ExpectedPolicyFingerprint, change.PolicyFingerprint)
	}
	if !policy.Enabled {
		return block("governance.disabled", "Product governance must be enabled before authorities can be changed.")
	}
	if len(request.Subjects) == 0 {
		return block("governance.authority-required", "At least one product authority subject is required.")
	}
	if duplicate != "" {
		return block("governance.authority-duplicate", "Each product authority may appear only once in a change.", duplicate)
	}
	for _, subject := range request.Subjects {
		if !authoritySubjectPattern.MatchString(subject) {
			return block("governance.authority-invalid", "Product authority must use a normalized user: or team: subject.", subject)
		}
	}

	current := make(map[string]bool, len(change.Before))
	for _, subject := range change.Before {
		current[subject] = true
	}
	switch request.Action {
	case AuthorityAdd:
		for _, subject := range request.Subjects {
			if current[subject] {
				return block("governance.authority-exists", "Product authority is already configured.", subject)
			}
		}
		change.After = append(change.After, request.Subjects...)
	case AuthorityRemove:
		for _, subject := range request.Subjects {
			if !current[subject] {
				return block("governance.authority-missing", "Product authority is not configured.", subject)
			}
		}
		if len(change.Before) == len(request.Subjects) {
			return block("governance.authority-lockout", "The final product authority cannot be removed while governance is enabled.", request.Subjects...)
		}
		removed := make(map[string]bool, len(request.Subjects))
		for _, subject := range request.Subjects {
			removed[subject] = true
		}
		change.After = change.After[:0]
		for _, subject := range change.Before {
			if !removed[subject] {
				change.After = append(change.After, subject)
			}
		}
		if policy.Approval.Minimum > len(change.After) {
			return block("governance.approval-lockout", "Removing these authorities would make the configured approval minimum impossible.", request.Subjects...)
		}
	default:
		return block("governance.authority-action-invalid", "Authority action must be add or remove.", string(request.Action))
	}
	sort.Strings(change.After)

	content, err := replaceAuthorities(raw, change.After)
	if err != nil {
		return AuthorityChange{}, operation.Plan{}, err
	}
	proposed, err := schema.DecodeYAML[Policy](content)
	if err != nil {
		return AuthorityChange{}, operation.Plan{}, fmt.Errorf("decode proposed governance policy: %w", err)
	}
	if issues := schema.Validate("governance", proposed); len(issues) > 0 {
		return AuthorityChange{}, operation.Plan{}, fmt.Errorf("validate proposed governance policy: %s", issues[0].Message)
	}
	info, err := os.Stat(path)
	if err != nil {
		return AuthorityChange{}, operation.Plan{}, err
	}
	plan.Files = []operation.FileChange{{Path: filepath.ToSlash(filepath.Join(".harness", "governance.yaml")), Content: content, Mode: info.Mode().Perm()}}
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	if err != nil {
		return AuthorityChange{}, operation.Plan{}, err
	}
	currentRaw, err := os.ReadFile(path)
	if err != nil {
		return AuthorityChange{}, operation.Plan{}, err
	}
	if policyDigest(currentRaw) != change.PolicyFingerprint {
		return block("governance.policy-stale", "Governance policy changed while the authority plan was being created.", change.PolicyFingerprint, policyDigest(currentRaw))
	}
	return change, plan, nil
}

func replaceAuthorities(raw []byte, authorities []string) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("governance policy must be one YAML mapping")
	}
	mapping := document.Content[0]
	for index := 0; index < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value != "product_authorities" {
			continue
		}
		sequence := mapping.Content[index+1]
		style := sequence.Style
		existing := make(map[string]*yaml.Node, len(sequence.Content))
		for _, item := range sequence.Content {
			if item.Kind == yaml.ScalarNode {
				existing[item.Value] = item
			}
		}
		sequence.Kind = yaml.SequenceNode
		sequence.Tag = "!!seq"
		sequence.Style = style
		sequence.Content = nil
		for _, subject := range authorities {
			item := existing[subject]
			if item == nil {
				item = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: subject}
			}
			sequence.Content = append(sequence.Content, item)
		}
		return yaml.Marshal(&document)
	}
	return nil, fmt.Errorf("governance policy has no product_authorities field")
}

func authorityOperationID(action AuthorityAction, subjects []string, policyFingerprint string) string {
	sum := sha256.Sum256([]byte(string(action) + "\x00" + strings.Join(subjects, "\x00") + "\x00" + policyFingerprint))
	return "governance-authority-" + string(action) + "-" + hex.EncodeToString(sum[:6])
}

func policyDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
