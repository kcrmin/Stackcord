package governance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/schema"
)

const defaultObservationPath = ".harness/local/governance/approval.yaml"

// LoadPolicy strictly loads committed governance. A missing file means governance is not enabled.
func LoadPolicy(root string) (Policy, error) {
	path := filepath.Join(root, ".harness", "governance.yaml")
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return Policy{SchemaVersion: 1, Enabled: false, ProductAuthorities: []string{}, ProtectedKinds: defaultProtectedKinds(), Approval: ApprovalPolicy{Minimum: 1, AuthoritySelfApproval: true}}, nil
	}
	if err := regularFile(path, "governance policy"); err != nil {
		return Policy{}, err
	}
	raw, err := schema.LoadYAML[map[string]any](path)
	if err != nil {
		return Policy{}, err
	}
	if issues := schema.Validate("governance", raw); len(issues) > 0 {
		return Policy{}, fmt.Errorf("validate governance policy: %s", issues[0].Message)
	}
	policy, err := schema.LoadYAML[Policy](path)
	if err != nil {
		return Policy{}, err
	}
	policy.ProductAuthorities = uniqueSorted(policy.ProductAuthorities)
	policy.ProtectedKinds = uniqueSorted(policy.ProtectedKinds)
	if policy.Enabled && policy.Approval.Minimum > len(policy.ProductAuthorities) {
		return Policy{}, fmt.Errorf("governance approval minimum exceeds configured product authorities")
	}
	return policy, nil
}

func loadObservation(root, path string) (Observation, error) {
	if path == "" {
		path = filepath.Join(root, filepath.FromSlash(defaultObservationPath))
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(root, filepath.FromSlash(path))
	}
	if err := validateObservationLocation(root, path); err != nil {
		return Observation{}, err
	}
	if err := regularFile(path, "governance observation"); err != nil {
		return Observation{}, err
	}
	raw, err := schema.LoadYAML[map[string]any](path)
	if err != nil {
		return Observation{}, err
	}
	if issues := schema.Validate("governance-observation", raw); len(issues) > 0 {
		return Observation{}, fmt.Errorf("validate governance observation: %s", issues[0].Message)
	}
	return schema.LoadYAML[Observation](path)
}

func validateObservationLocation(root, path string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return err
	}
	resolved := filepath.Join(parent, filepath.Base(path))
	temporaryRoot, tempErr := filepath.EvalSymlinks(os.TempDir())
	if tempErr != nil {
		temporaryRoot = os.TempDir()
	}
	localRoot := filepath.Join(root, ".harness", "local", "governance")
	if !pathWithin(localRoot, resolved) && !pathWithin(temporaryRoot, resolved) {
		return fmt.Errorf("governance observations must stay under .harness/local/governance or an explicit temporary path")
	}
	return nil
}

func regularFile(path, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular non-symlink file", label)
	}
	return nil
}

func pathWithin(parent, child string) bool {
	parent, _ = filepath.Abs(parent)
	child, _ = filepath.Abs(child)
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func defaultProtectedKinds() []string {
	return []string{"business", "contract", "policy", "product"}
}
