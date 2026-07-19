package contract_test

import (
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/contract"
	"github.com/stretchr/testify/require"
)

func TestCompareCompatibilityRules(t *testing.T) {
	base := contract.Definition{ID: "contract.identity.recovery.v1", Fields: map[string]contract.Field{"id": {Type: "string", Required: true}}, Errors: map[string]string{"RATE_LIMITED": "retry later"}, Retry: "safe", Idempotency: "required", Timeout: "5s", PartialFailure: "reject", Compensation: "not-required"}

	additive := cloneContract(base)
	additive.Fields["display_name"] = contract.Field{Type: "string", Required: false}
	require.False(t, contract.Compare(base, additive).Breaking)

	for name, mutate := range map[string]func(*contract.Definition){
		"required field": func(d *contract.Definition) {
			d.Fields["display_name"] = contract.Field{Type: "string", Required: true}
		},
		"removed field":   func(d *contract.Definition) { delete(d.Fields, "id") },
		"type changed":    func(d *contract.Definition) { d.Fields["id"] = contract.Field{Type: "integer", Required: true} },
		"error semantic":  func(d *contract.Definition) { d.Errors["RATE_LIMITED"] = "never retry" },
		"retry":           func(d *contract.Definition) { d.Retry = "unsafe" },
		"idempotency":     func(d *contract.Definition) { d.Idempotency = "none" },
		"timeout":         func(d *contract.Definition) { d.Timeout = "10s" },
		"partial failure": func(d *contract.Definition) { d.PartialFailure = "accept" },
		"compensation":    func(d *contract.Definition) { d.Compensation = "required" },
	} {
		t.Run(name, func(t *testing.T) {
			next := cloneContract(base)
			mutate(&next)
			require.True(t, contract.Compare(base, next).Breaking)
		})
	}

	versioned := cloneContract(base)
	versioned.ID = "contract.identity.recovery.v2"
	versioned.Fields["id"] = contract.Field{Type: "integer", Required: true}
	require.True(t, contract.Compare(base, versioned).Coordinated)
}

func TestCheckRequiresCompleteBehaviorObligations(t *testing.T) {
	definition := contract.Definition{ID: "contract.identity.recovery.v1", Retry: "safe", Idempotency: "required"}
	issues := contract.Check(definition)
	require.NotEmpty(t, issues)
	require.Equal(t, "contract.behavior-required", issues[0].Code)
}

func cloneContract(value contract.Definition) contract.Definition {
	copy := value
	copy.Fields = map[string]contract.Field{}
	for key, field := range value.Fields {
		copy.Fields[key] = field
	}
	copy.Errors = map[string]string{}
	for key, meaning := range value.Errors {
		copy.Errors[key] = meaning
	}
	return copy
}
