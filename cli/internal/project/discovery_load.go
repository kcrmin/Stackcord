package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"go.yaml.in/yaml/v3"
)

// Only coordination and references are stored here. Product text stays in specs/.
type discoveryState struct {
	SchemaVersion     int               `yaml:"schema_version"`
	Plan              *DiscoveryPlan    `yaml:"plan"`
	DecisionQuestions map[string]string `yaml:"decision_questions"`
}

func discoveryStateYAML(c DiscoveryCheckpoint) string {
	s := discoveryState{SchemaVersion: 1, Plan: c.Discovery, DecisionQuestions: map[string]string{}}
	for _, d := range c.Decisions {
		if d.QuestionID != "" {
			s.DecisionQuestions[d.ID] = d.QuestionID
		}
	}
	data, _ := yaml.Marshal(s)
	return string(data)
}

// ReadDiscovery reads either a saved draft or initialized canonical product files.
func ReadDiscovery(root string, draft bool) (DiscoveryCheckpoint, string, error) {
	if draft {
		data, err := readDiscoveryFile(root, "checkpoint.yaml")
		if err != nil {
			return DiscoveryCheckpoint{}, "", err
		}
		c, err := decodeDiscoveryCheckpoint(data)
		if err != nil {
			return c, "", err
		}
		return c, discoveryLocale(root, "manifest.yaml"), validateCheckpoint(c)
	}
	c := DiscoveryCheckpoint{SchemaVersion: 1, Summary: "Discovery", Roles: []DiscoveryFact{}, Journeys: []DiscoveryFact{}, Capabilities: []DiscoveryFact{}, Policies: []DiscoveryFact{}, Scenarios: []DiscoveryScenario{}, Quality: []DiscoveryFact{}, UICoverage: []UICoverage{}, TechnologyNeeds: []DiscoveryFact{}, Decisions: []DiscoveryDecision{}, Assumptions: []DiscoveryFact{}, OpenQuestions: []DiscoveryFact{}}
	state := discoveryState{DecisionQuestions: map[string]string{}}
	data, err := readDiscoveryFile(root, ".harness/discovery.yaml")
	if err == nil {
		raw, decodeErr := schema.DecodeYAML[map[string]any](data)
		if decodeErr != nil {
			return c, "", decodeErr
		}
		if issues := schema.Validate("discovery-state", raw); len(issues) > 0 {
			return c, "", fmt.Errorf("invalid discovery state: %s", issues[0].Message)
		}
		state, err = schema.DecodeYAML[discoveryState](data)
		if err != nil {
			return c, "", err
		}
		if state.SchemaVersion != 1 {
			return c, "", fmt.Errorf("unsupported discovery state version")
		}
		c.Discovery = state.Plan
	} else if !os.IsNotExist(err) {
		return c, "", err
	}
	found := map[string]bool{}
	for _, kind := range []string{"open-questions", "decisions"} {
		dir := filepath.Join("specs", "product", kind)
		if err := discoveryPathSafe(root, dir); err != nil && !os.IsNotExist(err) {
			return c, "", err
		}
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return c, "", err
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			rel := filepath.Join(dir, entry.Name())
			data, err := readDiscoveryFile(root, rel)
			if err != nil {
				return c, "", err
			}
			id, status, body, err := readDiscoveryDocument(data)
			if err != nil {
				return c, "", fmt.Errorf("%s: %w", rel, err)
			}
			if entry.Name() != id+".md" {
				return c, "", fmt.Errorf("discovery document identity differs from filename: %s", rel)
			}
			if kind == "open-questions" {
				if status == "unknown" || status == "proposed" || status == "draft" {
					c.OpenQuestions = append(c.OpenQuestions, DiscoveryFact{ID: id, Summary: body})
				}
			} else if status == "approved" {
				choice, rationale, ok := strings.Cut(strings.TrimPrefix(body, "Choice: "), "\n\nRationale: ")
				if !strings.HasPrefix(body, "Choice: ") || !ok {
					return c, "", fmt.Errorf("decision %s needs Choice and Rationale fields", id)
				}
				c.Decisions = append(c.Decisions, DiscoveryDecision{ID: id, QuestionID: state.DecisionQuestions[id], Choice: choice, Rationale: rationale})
				found[id] = true
			}
		}
	}
	for id := range state.DecisionQuestions {
		if !found[id] {
			return c, "", fmt.Errorf("discovery references missing or unapproved decision %s", id)
		}
	}
	return c, discoveryLocale(root, ".harness/manifest.yaml"), validateCheckpoint(c)
}

// LoadDiscoveryCheckpoint validates required fields before Go can default them.
func LoadDiscoveryCheckpoint(path string) (DiscoveryCheckpoint, error) {
	data, err := readDiscoveryFile(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return DiscoveryCheckpoint{}, err
	}
	return decodeDiscoveryCheckpoint(data)
}

func decodeDiscoveryCheckpoint(data []byte) (DiscoveryCheckpoint, error) {
	raw, err := schema.DecodeYAML[map[string]any](data)
	if err != nil {
		return DiscoveryCheckpoint{}, err
	}
	if issues := schema.Validate("discovery", raw); len(issues) > 0 {
		return DiscoveryCheckpoint{}, fmt.Errorf("invalid discovery checkpoint: %s", issues[0].Message)
	}
	return schema.DecodeYAML[DiscoveryCheckpoint](data)
}

func discoveryLocale(root, path string) string {
	data, err := readDiscoveryFile(root, path)
	if err == nil {
		v, err := schema.DecodeYAML[map[string]any](data)
		if err == nil && v["locale"] == "ko" {
			return "ko"
		}
	}
	return "en"
}

func readDiscoveryDocument(data []byte) (string, string, string, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", "", "", fmt.Errorf("canonical discovery document needs frontmatter")
	}
	header, body, ok := strings.Cut(text[4:], "\n---\n")
	if !ok {
		return "", "", "", fmt.Errorf("unclosed discovery frontmatter")
	}
	meta, err := schema.DecodeYAML[map[string]any]([]byte(header))
	if err != nil {
		return "", "", "", err
	}
	if issues := schema.Validate("spec", meta); len(issues) > 0 {
		return "", "", "", fmt.Errorf("invalid discovery document: %s", issues[0].Message)
	}
	body = strings.TrimSpace(body)
	if strings.HasPrefix(body, "# ") {
		_, body, _ = strings.Cut(body, "\n")
		body = strings.TrimSpace(body)
	}
	return meta["id"].(string), meta["status"].(string), body, nil
}

func discoveryPathSafe(root, rel string) error {
	path := root
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("discovery source cannot be a symlink: %s", rel)
		}
	}
	return nil
}

func readDiscoveryFile(root, rel string) ([]byte, error) {
	if err := discoveryPathSafe(root, rel); err != nil {
		return nil, err
	}
	path := filepath.Join(root, rel)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1024*1024 {
		return nil, fmt.Errorf("discovery source must be a regular file of at most 1 MiB: %s", rel)
	}
	return os.ReadFile(path)
}
