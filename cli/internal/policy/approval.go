package policy

import "time"

// Action is a stable category for a tool operation.
type Action string

const (
	ReadStatus         Action = "read_status"
	WriteRequestedCode Action = "write_requested_code"
	AddSubmodule       Action = "add_submodule"
	PushBranch         Action = "push_branch"
	ForcePush          Action = "force_push"
	PublishProduction  Action = "publish_production"
	SendSecretExternal Action = "send_secret_external"
)

// Consent is a time-bounded authorization for one objective, repository, action, and target.
type Consent struct {
	Objective     string
	Repository    string
	Action        Action
	Target        string
	ExpiresAt     time.Time
	Approved      bool
	ExactDReceipt bool
}

// Scope is the operation being evaluated now.
type Scope struct {
	Objective  string
	Repository string
	Target     string
	Now        time.Time
}

// Decision explains the class and whether current consent satisfies it.
type Decision struct {
	Class         string
	Required      bool
	AlwaysConfirm bool
	Reason        string
}

// Classify applies the non-bypassable A-D approval policy.
func Classify(action Action, consent Consent, scopes ...Scope) Decision {
	decision := Decision{Class: classFor(action)}
	decision.AlwaysConfirm = decision.Class == "D"
	decision.Required = decision.Class != "A"
	if len(scopes) == 0 {
		return decision
	}
	scope := scopes[0]
	if scope.Now.IsZero() {
		scope.Now = time.Now().UTC()
	}
	matches := consent.Approved && consent.Action == action && consent.Objective == scope.Objective && consent.Repository == scope.Repository && consent.Target == scope.Target && consent.ExpiresAt.After(scope.Now)
	if matches && decision.Class != "D" {
		decision.Required = false
	}
	if matches && decision.Class == "D" && consent.ExactDReceipt {
		decision.Required = false
	}
	if decision.Required {
		decision.Reason = "current consent does not exactly authorize this objective, repository, action, target, and time window"
	}
	return decision
}

func classFor(action Action) string {
	switch action {
	case ReadStatus:
		return "A"
	case WriteRequestedCode:
		return "B"
	case AddSubmodule, PushBranch:
		return "C"
	case ForcePush, PublishProduction, SendSecretExternal:
		return "D"
	default:
		return "D"
	}
}
