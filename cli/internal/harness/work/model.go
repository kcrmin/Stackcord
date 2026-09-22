package work

// Readiness states whether a definition is still being discovered or is executable.
type Readiness string

const (
	Draft Readiness = "draft"
	Ready Readiness = "ready"
)

// AcceptanceScenario records observable success and failure behavior, not implementation steps.
type AcceptanceScenario struct {
	ID      string `json:"id" yaml:"id"`
	Given   string `json:"given" yaml:"given"`
	When    string `json:"when" yaml:"when"`
	Then    string `json:"then" yaml:"then"`
	Failure string `json:"failure" yaml:"failure"`
}

// Scope reserves both file paths and service meaning before implementation starts.
type Scope struct {
	Repositories     []string `json:"repositories" yaml:"repositories"`
	Paths            []string `json:"paths" yaml:"paths"`
	PolicyIDs        []string `json:"policy_ids" yaml:"policy_ids"`
	ScenarioIDs      []string `json:"scenario_ids" yaml:"scenario_ids"`
	ContractIDs      []string `json:"contract_ids" yaml:"contract_ids"`
	DBEntities       []string `json:"db_entities" yaml:"db_entities"`
	MigrationSlots   []string `json:"migration_slots" yaml:"migration_slots"`
	UIFlows          []string `json:"ui_flows" yaml:"ui_flows"`
	DependencyMajors []string `json:"dependency_majors" yaml:"dependency_majors"`
	RootPointers     []string `json:"root_pointers" yaml:"root_pointers"`
}

// EvidenceRequirements says what must be proven; actual evidence remains commit-bound state.
type EvidenceRequirements struct {
	Kinds               []string `json:"kinds" yaml:"kinds"`
	IntegrationRequired bool     `json:"integration_required" yaml:"integration_required"`
	UserValidation      bool     `json:"user_validation" yaml:"user_validation"`
	MigrationRequired   bool     `json:"migration_required" yaml:"migration_required"`
	RollbackRequired    bool     `json:"rollback_required" yaml:"rollback_required"`
}

// Definition is canonical work intent. Live owner and status belong to the selected provider.
type Definition struct {
	SchemaVersion    int                  `json:"schema_version" yaml:"schema_version"`
	ID               string               `json:"id" yaml:"id"`
	ParentID         string               `json:"parent_id,omitempty" yaml:"parent_id,omitempty"`
	Readiness        Readiness            `json:"readiness" yaml:"readiness"`
	Title            string               `json:"title" yaml:"title"`
	Outcome          string               `json:"outcome" yaml:"outcome"`
	Acceptance       []AcceptanceScenario `json:"acceptance" yaml:"acceptance"`
	Refs             []string             `json:"refs" yaml:"refs"`
	Workspaces       []string             `json:"workspaces" yaml:"workspaces"`
	Scope            Scope                `json:"scope" yaml:"scope"`
	Dependencies     []string             `json:"dependencies" yaml:"dependencies"`
	MergeOrder       []string             `json:"merge_order" yaml:"merge_order"`
	FirstFailingTest string               `json:"first_failing_test" yaml:"first_failing_test"`
	Evidence         EvidenceRequirements `json:"evidence" yaml:"evidence"`
	UIBaselines      map[string]string    `json:"ui_baselines,omitempty" yaml:"ui_baselines,omitempty"`
	Fingerprint      string               `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
}
