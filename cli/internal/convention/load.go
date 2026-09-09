package convention

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/schema"
)

const RelativePath = ".harness/git-conventions.yaml"

var (
	legacyBranchPattern = regexp.MustCompile(`^(feature|fix|bugfix|chore|docs|refactor|test|release)/([A-Za-z0-9]+-)?[a-z0-9]+(?:-[a-z0-9]+)*$`)
	placeholderPattern  = regexp.MustCompile(`\{([a-z_]+)\}`)
	typePattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

// Load reads the committed convention file. A missing file preserves the
// original Stackcord convention instead of creating new project state.
func Load(root string) (Config, bool, error) {
	path := filepath.Join(root, filepath.FromSlash(RelativePath))
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, fmt.Errorf("inspect Git conventions: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Config{}, false, fmt.Errorf("Git conventions must be a regular non-symlink file")
	}
	raw, err := schema.LoadYAML[map[string]any](path)
	if err != nil {
		return Config{}, false, err
	}
	if issues := schema.Validate("git-conventions", raw); len(issues) > 0 {
		return Config{}, false, fmt.Errorf("validate Git conventions: %s", issues[0].Message)
	}
	config, err := schema.LoadYAML[Config](path)
	if err != nil {
		return Config{}, false, err
	}
	if err := validateConfig(config); err != nil {
		return Config{}, false, fmt.Errorf("validate Git conventions: %w", err)
	}
	return config, true, nil
}

// ValidateSafeBranch checks Git reference invariants that repository
// conventions are never allowed to weaken.
func ValidateSafeBranch(branch string) error {
	if branch == "" || len(branch) > 255 || branch == "@" || strings.HasPrefix(branch, "-") || strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".") {
		return fmt.Errorf("branch is not a safe Git reference")
	}
	if strings.Contains(branch, "..") || strings.Contains(branch, "@{") || strings.Contains(branch, "//") || strings.ContainsAny(branch, "\\ ~^:?*[") {
		return fmt.Errorf("branch is not a safe Git reference")
	}
	for _, character := range branch {
		if character < 0x20 || character == 0x7f {
			return fmt.Errorf("branch is not a safe Git reference")
		}
	}
	for _, component := range strings.Split(branch, "/") {
		if component == "" || component == "." || component == ".." || strings.HasPrefix(component, ".") || strings.HasSuffix(component, ".") || strings.HasSuffix(strings.ToLower(component), ".lock") {
			return fmt.Errorf("branch is not a safe Git reference")
		}
	}
	return nil
}

// ValidateBranchIdentity validates recoverable collaboration state without
// assuming access to a repository's convention file.
func ValidateBranchIdentity(branch string) error {
	if err := ValidateSafeBranch(branch); err != nil {
		return err
	}
	if containsAIMarker(branch) {
		return fmt.Errorf("branch must not contain AI, agent, model, or tool attribution")
	}
	return nil
}

// ValidateBranch applies immutable Git safety and the repository convention.
func ValidateBranch(root, branch string) error {
	if err := ValidateBranchIdentity(branch); err != nil {
		return err
	}
	config, configured, err := Load(root)
	if err != nil {
		return err
	}
	if !configured || config.Branch == nil {
		if !legacyBranchPattern.MatchString(branch) {
			return fmt.Errorf("branch does not match the default Stackcord convention")
		}
		return nil
	}
	pattern, err := compileBranchFormat(*config.Branch)
	if err != nil {
		return err
	}
	if !pattern.MatchString(branch) {
		return fmt.Errorf("branch does not match the configured Git convention %q", config.Branch.Format)
	}
	return nil
}

func validateConfig(config Config) error {
	if config.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1")
	}
	if config.Branch != nil {
		if _, err := compileBranchFormat(*config.Branch); err != nil {
			return err
		}
	}
	if config.Commit != nil {
		if err := validateTemplate(config.Commit.TitleFormat, []string{"type", "scope", "subject"}, []string{"type", "subject"}); err != nil {
			return fmt.Errorf("commit title_format: %w", err)
		}
		if err := validateNames("commit type", config.Commit.Types, true); err != nil {
			return err
		}
		if err := validateNames("commit scope", config.Commit.Scopes, false); err != nil {
			return err
		}
	}
	if config.PullRequest != nil {
		if err := validateTemplate(config.PullRequest.TitleFormat, []string{"type", "scope", "issue", "title"}, []string{"title"}); err != nil {
			return fmt.Errorf("pull_request title_format: %w", err)
		}
		if err := validateSections("pull_request", config.PullRequest.RequiredSections); err != nil {
			return err
		}
	}
	if config.Issue != nil {
		if err := validateTemplate(config.Issue.TitleFormat, []string{"type", "issue", "title"}, []string{"title"}); err != nil {
			return fmt.Errorf("issue title_format: %w", err)
		}
		if err := validateSections("issue", config.Issue.RequiredSections); err != nil {
			return err
		}
	}
	return nil
}

func compileBranchFormat(branch BranchConvention) (*regexp.Regexp, error) {
	if err := validateNames("branch type", branch.Types, true); err != nil {
		return nil, err
	}
	if err := validateTemplate(branch.Format, []string{"type", "issue", "description"}, []string{"type", "description"}); err != nil {
		return nil, fmt.Errorf("branch format: %w", err)
	}
	replacements := map[string]string{
		"type":        "(?:" + quotedAlternatives(branch.Types) + ")",
		"issue":       `[A-Za-z0-9][A-Za-z0-9._-]*`,
		"description": `[a-z0-9]+(?:-[a-z0-9]+)*`,
	}
	var expression strings.Builder
	expression.WriteString("^")
	last := 0
	for _, match := range placeholderPattern.FindAllStringSubmatchIndex(branch.Format, -1) {
		expression.WriteString(regexp.QuoteMeta(branch.Format[last:match[0]]))
		expression.WriteString(replacements[branch.Format[match[2]:match[3]]])
		last = match[1]
	}
	expression.WriteString(regexp.QuoteMeta(branch.Format[last:]))
	expression.WriteString("$")
	return regexp.Compile(expression.String())
}

func validateTemplate(value string, allowed, required []string) error {
	if strings.TrimSpace(value) != value || value == "" || strings.ContainsAny(value, "\r\n") || len(value) > 200 {
		return fmt.Errorf("must be one non-empty line")
	}
	allowedSet := map[string]bool{}
	for _, name := range allowed {
		allowedSet[name] = true
	}
	counts := map[string]int{}
	for _, match := range placeholderPattern.FindAllStringSubmatch(value, -1) {
		name := match[1]
		if !allowedSet[name] {
			return fmt.Errorf("unsupported placeholder {%s}", name)
		}
		counts[name]++
		if counts[name] > 1 {
			return fmt.Errorf("placeholder {%s} must not repeat", name)
		}
	}
	remainder := placeholderPattern.ReplaceAllString(value, "")
	if strings.ContainsAny(remainder, "{}") {
		return fmt.Errorf("contains malformed placeholder")
	}
	for _, name := range required {
		if counts[name] != 1 {
			return fmt.Errorf("must contain {%s}", name)
		}
	}
	return nil
}

func validateNames(label string, values []string, required bool) error {
	if required && len(values) == 0 {
		return fmt.Errorf("at least one %s is required", label)
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !typePattern.MatchString(value) || seen[value] {
			return fmt.Errorf("%s values must be unique portable identifiers", label)
		}
		seen[value] = true
	}
	return nil
}

func validateSections(label string, values []string) error {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n") || seen[value] {
			return fmt.Errorf("%s required_sections must be unique non-empty lines", label)
		}
		seen[value] = true
	}
	return nil
}

func quotedAlternatives(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, regexp.QuoteMeta(value))
	}
	return strings.Join(quoted, "|")
}

func containsAIMarker(branch string) bool {
	lower := strings.ToLower(branch)
	for _, token := range strings.FieldsFunc(lower, func(character rune) bool {
		return character == '-' || character == '_' || character == '.' || character == '/'
	}) {
		if token == "ai" || token == "agent" || token == "codex" || token == "gpt" {
			return true
		}
	}
	return strings.Contains(lower, "generated-by") || strings.Contains(lower, "model-generated") || strings.Contains(lower, "generated-model")
}

// ValidateConfig validates an in-memory proposal before any canonical file is written.
func ValidateConfig(config Config) error { return validateConfig(config) }
