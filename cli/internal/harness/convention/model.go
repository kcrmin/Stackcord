// Package convention loads repository-owned Git collaboration conventions.
// It keeps immutable Git reference safety separate from configurable team style.
package convention

type Config struct {
	SchemaVersion int                `json:"schema_version" yaml:"schema_version"`
	Branch        *BranchConvention  `json:"branch,omitempty" yaml:"branch,omitempty"`
	Commit        *CommitConvention  `json:"commit,omitempty" yaml:"commit,omitempty"`
	PullRequest   *ContentConvention `json:"pull_request,omitempty" yaml:"pull_request,omitempty"`
	Issue         *ContentConvention `json:"issue,omitempty" yaml:"issue,omitempty"`
}

type BranchConvention struct {
	Format string   `json:"format" yaml:"format"`
	Types  []string `json:"types" yaml:"types"`
}

type CommitConvention struct {
	TitleFormat string   `json:"title_format" yaml:"title_format"`
	Types       []string `json:"types" yaml:"types"`
	Scopes      []string `json:"scopes" yaml:"scopes"`
	MaxLength   int      `json:"max_length" yaml:"max_length"`
}

type ContentConvention struct {
	TitleFormat      string   `json:"title_format" yaml:"title_format"`
	RequiredSections []string `json:"required_sections" yaml:"required_sections"`
}
