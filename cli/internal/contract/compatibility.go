package contract

import "sort"

// Field is a normalized request, response, event, or data field obligation.
type Field struct {
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
}

// Definition includes structural and behavioral obligations.
type Definition struct {
	SchemaVersion  int               `json:"schema_version,omitempty" yaml:"schema_version,omitempty"`
	ID             string            `json:"id" yaml:"id"`
	Kind           Kind              `json:"kind,omitempty" yaml:"kind,omitempty"`
	Status         Status            `json:"status,omitempty" yaml:"status,omitempty"`
	Revision       int               `json:"revision,omitempty" yaml:"revision,omitempty"`
	Refs           []string          `json:"refs,omitempty" yaml:"refs,omitempty"`
	Purpose        string            `json:"purpose,omitempty" yaml:"purpose,omitempty"`
	NonGoals       []string          `json:"non_goals,omitempty" yaml:"non_goals,omitempty"`
	Rules          []string          `json:"rules,omitempty" yaml:"rules,omitempty"`
	Eligibility    []string          `json:"eligibility,omitempty" yaml:"eligibility,omitempty"`
	Invariants     []string          `json:"invariants,omitempty" yaml:"invariants,omitempty"`
	Outcomes       map[string]string `json:"outcomes,omitempty" yaml:"outcomes,omitempty"`
	Rejections     map[string]string `json:"rejections,omitempty" yaml:"rejections,omitempty"`
	Failures       map[string]string `json:"failures,omitempty" yaml:"failures,omitempty"`
	Fields         map[string]Field  `json:"fields" yaml:"fields"`
	Errors         map[string]string `json:"errors" yaml:"errors"`
	Retry          string            `json:"retry" yaml:"retry"`
	Idempotency    string            `json:"idempotency" yaml:"idempotency"`
	Timeout        string            `json:"timeout" yaml:"timeout"`
	PartialFailure string            `json:"partial_failure" yaml:"partial_failure"`
	Compensation   string            `json:"compensation" yaml:"compensation"`
	DataOwner      string            `json:"data_owner,omitempty" yaml:"data_owner,omitempty"`
	Classification string            `json:"classification,omitempty" yaml:"classification,omitempty"`
	Retention      string            `json:"retention,omitempty" yaml:"retention,omitempty"`
	Deletion       string            `json:"deletion,omitempty" yaml:"deletion,omitempty"`
	Migration      string            `json:"migration,omitempty" yaml:"migration,omitempty"`
}

// Report explains compatibility rather than reducing it to syntax only.
type Report struct {
	Breaking    bool     `json:"breaking"`
	Coordinated bool     `json:"coordinated"`
	Reasons     []string `json:"reasons"`
}

// Compare evaluates structural, failure, retry, and idempotency semantics.
func Compare(old, next Definition) Report {
	report := Report{Reasons: []string{}}
	if effectiveKind(old.Kind) != effectiveKind(next.Kind) {
		report.Breaking = true
		report.Reasons = append(report.Reasons, "changed contract kind")
	}
	for label, changed := range map[string]bool{
		"purpose":             old.Purpose != next.Purpose,
		"business rules":      !sameContractStrings(old.Rules, next.Rules),
		"eligibility":         !sameContractStrings(old.Eligibility, next.Eligibility),
		"invariants":          !sameContractStrings(old.Invariants, next.Invariants),
		"observable outcomes": !sameContractMap(old.Outcomes, next.Outcomes),
		"rejection behavior":  !sameContractMap(old.Rejections, next.Rejections),
		"failure behavior":    !sameContractMap(old.Failures, next.Failures),
		"data lifecycle":      old.DataOwner != next.DataOwner || old.Classification != next.Classification || old.Retention != next.Retention || old.Deletion != next.Deletion || old.Migration != next.Migration,
	} {
		if changed {
			report.Breaking = true
			report.Reasons = append(report.Reasons, "changed "+label)
		}
	}
	for name, previous := range old.Fields {
		current, exists := next.Fields[name]
		if !exists {
			report.Breaking = true
			report.Reasons = append(report.Reasons, "removed field: "+name)
			continue
		}
		if previous.Type != current.Type {
			report.Breaking = true
			report.Reasons = append(report.Reasons, "changed field type: "+name)
		}
		if !previous.Required && current.Required {
			report.Breaking = true
			report.Reasons = append(report.Reasons, "field became required: "+name)
		}
	}
	for name, current := range next.Fields {
		if _, existed := old.Fields[name]; !existed && current.Required {
			report.Breaking = true
			report.Reasons = append(report.Reasons, "new required field: "+name)
		}
	}
	for code, meaning := range old.Errors {
		if next.Errors[code] != meaning {
			report.Breaking = true
			report.Reasons = append(report.Reasons, "changed error semantic: "+code)
		}
	}
	if old.Retry != next.Retry {
		report.Breaking = true
		report.Reasons = append(report.Reasons, "changed retry obligation")
	}
	if old.Idempotency != next.Idempotency {
		report.Breaking = true
		report.Reasons = append(report.Reasons, "changed idempotency obligation")
	}
	if old.Timeout != next.Timeout {
		report.Breaking = true
		report.Reasons = append(report.Reasons, "changed timeout obligation")
	}
	if old.PartialFailure != next.PartialFailure {
		report.Breaking = true
		report.Reasons = append(report.Reasons, "changed partial-failure obligation")
	}
	if old.Compensation != next.Compensation {
		report.Breaking = true
		report.Reasons = append(report.Reasons, "changed compensation obligation")
	}
	if report.Breaking && old.ID != next.ID {
		report.Coordinated = true
	}
	sort.Strings(report.Reasons)
	return report
}

func effectiveKind(kind Kind) Kind {
	if kind == "" {
		return Interface
	}
	return kind
}

func sameContractStrings(left, right []string) bool {
	left, right = append([]string(nil), left...), append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameContractMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
