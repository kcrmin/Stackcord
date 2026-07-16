package contract

import "sort"

// Field is a normalized request, response, event, or data field obligation.
type Field struct {
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
}

// Definition includes structural and behavioral obligations.
type Definition struct {
	ID             string            `json:"id" yaml:"id"`
	Fields         map[string]Field  `json:"fields" yaml:"fields"`
	Errors         map[string]string `json:"errors" yaml:"errors"`
	Retry          string            `json:"retry" yaml:"retry"`
	Idempotency    string            `json:"idempotency" yaml:"idempotency"`
	Timeout        string            `json:"timeout" yaml:"timeout"`
	PartialFailure string            `json:"partial_failure" yaml:"partial_failure"`
	Compensation   string            `json:"compensation" yaml:"compensation"`
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
