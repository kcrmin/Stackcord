package context

// SourceRef links a generated or derived document to an authored source fingerprint.
type SourceRef struct {
	Source      string `json:"source" yaml:"source"`
	Fingerprint string `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
}

// IndexEntry is the deterministic, machine-readable view of one authored document.
type IndexEntry struct {
	ID          string      `json:"id"`
	Path        string      `json:"path"`
	Kind        string      `json:"kind"`
	Status      string      `json:"status"`
	Revision    int         `json:"revision"`
	Fingerprint string      `json:"fingerprint"`
	Refs        []string    `json:"refs"`
	Sources     []SourceRef `json:"sources,omitempty"`
}

// Snapshot is rebuilt from canonical files and actual state, never conversation memory.
type Snapshot struct {
	SchemaVersion int                   `json:"schema_version"`
	Index         map[string]IndexEntry `json:"index"`
	Impact        map[string][]string   `json:"impact"`
	Stale         []string              `json:"stale"`
	Unknown       []string              `json:"unknown"`
}

type documentMetadata struct {
	SchemaVersion int         `yaml:"schema_version"`
	ID            string      `yaml:"id"`
	Kind          string      `yaml:"kind"`
	Status        string      `yaml:"status"`
	Revision      int         `yaml:"revision"`
	Refs          []string    `yaml:"refs"`
	Sources       []SourceRef `yaml:"sources"`
}
