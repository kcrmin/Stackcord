package release

import "fullstack-orchestrator/cli/internal/domain"

// Warning is publishable only when ownership and rationale are explicit.
type Warning struct {
	Code      string `json:"code"`
	Owner     string `json:"owner"`
	Rationale string `json:"rationale"`
}

// Gates are production-critical outcomes, not check-box intentions.
type Gates struct {
	RequiredChecksStable      bool      `json:"required_checks_stable"`
	CriticalChecksAutomated   bool      `json:"critical_checks_automated"`
	ArtifactsSigned           bool      `json:"artifacts_signed"`
	MigrationRollbackVerified bool      `json:"migration_rollback_verified"`
	HooksTrustedReadOnly      bool      `json:"hooks_trusted_read_only"`
	MacOSJourneyVerified      bool      `json:"macos_journey_verified"`
	WindowsJourneyVerified    bool      `json:"windows_journey_verified"`
	PluginlessContinuation    bool      `json:"pluginless_continuation"`
	UserValidationMatches     bool      `json:"user_validation_matches"`
	Warnings                  []Warning `json:"warnings"`
}

// Verify aggregates production blockers and owned warnings.
func Verify(gates Gates) domain.Result {
	result := domain.Result{SchemaVersion: "1.0", ToolVersion: "dev", Command: "verify.release", OperationID: "release-gates", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: "All production release gates passed."}
	checks := []struct {
		ok            bool
		code, message string
	}{
		{gates.RequiredChecksStable, "release.required-checks-unstable", "Required checks are failing or flaky."},
		{gates.CriticalChecksAutomated, "release.critical-check-manual", "A production-critical verification is manual-only."},
		{gates.ArtifactsSigned, "release.artifact-unsigned", "Every published artifact must be signed."},
		{gates.MigrationRollbackVerified, "release.rollback-unverified", "Migration rollback has not been verified."},
		{gates.HooksTrustedReadOnly, "release.hook-unsafe", "Plugin Hooks are not proven trusted and read-only."},
		{gates.MacOSJourneyVerified, "release.macos-unverified", "The macOS user journey has not passed."},
		{gates.WindowsJourneyVerified, "release.windows-unverified", "The Windows user journey has not passed."},
		{gates.PluginlessContinuation, "release.pluginless-unverified", "Clone continuation without the Plugin has not passed."},
		{gates.UserValidationMatches, "release.user-validation-mismatch", "User validation does not reference the same RC digest."},
	}
	for _, check := range checks {
		if !check.ok {
			result.Blockers = append(result.Blockers, domain.Item{Code: check.code, Message: check.message})
		}
	}
	for _, warning := range gates.Warnings {
		if warning.Owner == "" || warning.Rationale == "" {
			result.Blockers = append(result.Blockers, domain.Item{Code: "release.warning-unowned", Message: "Every release warning needs an owner and rationale.", Refs: []string{warning.Code}})
			continue
		}
		result.Warnings = append(result.Warnings, domain.Item{Code: warning.Code, Message: warning.Rationale, Refs: []string{warning.Owner}})
	}
	if len(result.Blockers) > 0 {
		result.Status, result.ExitCode, result.Summary = domain.StatusBlocked, domain.ExitVerification, "Production release gates are blocked."
	}
	return result
}
