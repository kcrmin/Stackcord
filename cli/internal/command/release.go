package command

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/release"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"github.com/spf13/cobra"
)

const maxUserValidationEvidenceBytes = 64 << 20

func newReleaseCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "release", Short: "Prepare and verify one exact production candidate"}
	parent.AddCommand(newReleasePrepare(version, jsonOutput), newReleaseValidate(version, jsonOutput), newReleaseVerify(version, jsonOutput))
	return parent
}

func newReleasePrepare(version string, jsonOutput *bool) *cobra.Command {
	var root, inputPath, outputPath, releaseVersion, profileValue, strictEvidencePath string
	var workIDs []string
	var apply bool
	command := &cobra.Command{Use: "prepare", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		root = located.Path
		var input release.Input
		if inputPath != "" {
			var err error
			input, err = readJSON[release.Input](inputPath)
			if err != nil {
				return err
			}
		} else {
			if releaseVersion == "" {
				return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "release.prepare", "release.version-required", "A product release version is required."))
			}
			profile := release.Profile(profileValue)
			var strictEvidence *release.StrictEvidence
			if strictEvidencePath != "" {
				loaded, err := readJSON[release.StrictEvidence](strictEvidencePath)
				if err != nil {
					return err
				}
				strictEvidence = &loaded
			}
			collected, issues := release.CollectInput(cmd.Context(), root, release.CollectOptions{Version: releaseVersion, Profile: profile, StrictEvidence: strictEvidence, StackcordVersion: version, WorkIDs: workIDs})
			if len(issues) > 0 {
				result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "release.prepare", OperationID: "release-collect-read-only", Status: domain.StatusBlocked, ExitCode: domain.ExitVerification, Summary: "Current service state cannot form a release candidate.", Blockers: issues}
				return writeResult(cmd, *jsonOutput, result)
			}
			input = collected
		}
		candidate, result := release.CreateCandidate(input)
		result.ToolVersion = version
		if result.Status != domain.StatusPassed {
			return writeResult(cmd, *jsonOutput, result)
		}
		if issues := schema.Validate("release-candidate", candidate); len(issues) > 0 {
			return fmt.Errorf("validate release candidate: %s", issues[0].Message)
		}
		data, err := json.MarshalIndent(candidate, "", "  ")
		if err != nil {
			return err
		}
		plan := operation.Plan{ID: result.OperationID, Root: root, Files: []operation.FileChange{{Path: outputPath, Content: append(data, '\n'), Mode: 0o644}}}
		plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
		if err != nil {
			return err
		}
		if apply {
			applied := operation.Apply(cmd.Context(), plan)
			applied.ToolVersion, applied.Command = version, "release.prepare"
			applied.Evidence = append(applied.Evidence, result.Evidence...)
			if applied.Status == domain.StatusPassed {
				applied.Summary = "Exact technical release candidate is recorded; user validation must reference this digest."
				applied.NextActions = []domain.Item{{Code: "release.user-validate", Message: "Run the exact candidate in its target environment, save the result, and confirm that evidence against this digest."}}
			}
			return writeResult(cmd, *jsonOutput, applied)
		}
		planned := planResult(version, "release.prepare.plan", plan, "Exact release candidate write is planned; no file changed.")
		planned.Evidence = result.Evidence
		return writeResult(cmd, *jsonOutput, planned)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root")
	command.Flags().StringVar(&inputPath, "input", "", "legacy release input JSON; normal use collects actual state")
	command.Flags().StringVar(&releaseVersion, "release-version", "", "product release version")
	command.Flags().StringVar(&profileValue, "profile", "", "core or strict-release; defaults to committed profile")
	command.Flags().StringVar(&strictEvidencePath, "strict-evidence", "", "strict release evidence JSON when the strict profile is selected")
	command.Flags().StringSliceVar(&workIDs, "work", nil, "work stable ID included in this release")
	command.Flags().StringVar(&outputPath, "output", ".harness/local/release/candidate.json", "local candidate path relative to root")
	command.Flags().BoolVar(&apply, "apply", false, "write the candidate atomically")
	return command
}

func newReleaseValidate(version string, jsonOutput *bool) *cobra.Command {
	var root, candidatePath, evidencePath, outputPath string
	var confirmed, apply bool
	command := &cobra.Command{Use: "validate", Short: "Bind explicit user validation evidence to one exact candidate", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		root = located.Path
		if !confirmed {
			return writeResult(cmd, *jsonOutput, lifecycleBlocked(version, "release.validate", "release.user-confirmation-required", "User validation must explicitly confirm the exact candidate."))
		}
		candidate, err := readProjectJSON[release.Candidate](root, candidatePath)
		if err != nil {
			return err
		}
		if blockers := release.ValidateCandidate(candidate); len(blockers) > 0 {
			result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "release.validate", OperationID: "release-validate-read-only", Status: domain.StatusBlocked, ExitCode: domain.ExitVerification, Summary: "User validation cannot bind to an invalid or tampered candidate.", Blockers: blockers}
			return writeResult(cmd, *jsonOutput, result)
		}
		data, err := readValidationEvidence(evidencePath)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		validation := release.UserValidation{SchemaVersion: 1, CandidateDigest: candidate.Digest, Confirmed: true, EvidenceDigest: "sha256:" + hex.EncodeToString(digest[:]), VerifiedAt: time.Now().UTC().Format(time.RFC3339)}
		if issues := schema.Validate("release-validation", validation); len(issues) > 0 {
			return fmt.Errorf("validate user release evidence: %s", issues[0].Message)
		}
		encoded, err := json.MarshalIndent(validation, "", "  ")
		if err != nil {
			return err
		}
		plan := operation.Plan{ID: "release-validation-" + candidate.Digest[len("sha256:"):len("sha256:")+12], Root: root, Files: []operation.FileChange{{Path: outputPath, Content: append(encoded, '\n'), Mode: 0o600}}}
		plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
		if err != nil {
			return err
		}
		if apply {
			result := operation.Apply(cmd.Context(), plan)
			result.ToolVersion, result.Command = version, "release.validate"
			if result.Status == domain.StatusPassed {
				result.Summary = "User validation evidence is bound to the exact candidate digest."
				result.Evidence = append(result.Evidence, domain.Item{Code: "release.user-validation", Message: validation.EvidenceDigest, Refs: []string{validation.CandidateDigest}})
			}
			return writeResult(cmd, *jsonOutput, result)
		}
		result := planResult(version, "release.validate.plan", plan, "User validation record is planned for the exact candidate; no file changed.")
		result.Evidence = []domain.Item{{Code: "release.user-validation", Message: validation.EvidenceDigest, Refs: []string{validation.CandidateDigest}}}
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root")
	command.Flags().StringVar(&candidatePath, "candidate", ".harness/local/release/candidate.json", "candidate path relative to root")
	command.Flags().StringVar(&evidencePath, "evidence", "", "user test result file to hash without copying its content")
	command.Flags().StringVar(&outputPath, "output", ".harness/local/release/user-validation.json", "local validation path relative to root")
	command.Flags().BoolVar(&confirmed, "confirm", false, "explicitly confirm the candidate after real target validation")
	command.Flags().BoolVar(&apply, "apply", false, "write the local validation record atomically")
	_ = command.MarkFlagRequired("evidence")
	return command
}

func newReleaseVerify(version string, jsonOutput *bool) *cobra.Command {
	var root, candidatePath, inputPath, validationPath string
	command := &cobra.Command{Use: "verify", RunE: func(cmd *cobra.Command, _ []string) error {
		located, err := workspace.FindRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		root = located.Path
		candidate, err := readProjectJSON[release.Candidate](root, candidatePath)
		if err != nil {
			return err
		}
		var input release.Input
		if inputPath != "" {
			input, err = readJSON[release.Input](inputPath)
			if err != nil {
				return err
			}
		} else {
			var issues []domain.Item
			workIDs := make([]string, 0, len(candidate.Input.ProviderRevisions))
			for workID := range candidate.Input.ProviderRevisions {
				workIDs = append(workIDs, workID)
			}
			input, issues = release.CollectInput(cmd.Context(), root, release.CollectOptions{Version: candidate.Input.Version, Profile: candidate.Input.Profile, StrictEvidence: candidate.Input.StrictEvidence, StackcordVersion: version, WorkIDs: workIDs})
			if len(issues) > 0 {
				result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "release.verify", OperationID: "release-collect-read-only", Status: domain.StatusBlocked, ExitCode: domain.ExitVerification, Summary: "Current service state no longer verifies as the candidate.", Blockers: issues}
				return writeResult(cmd, *jsonOutput, result)
			}
		}
		validation, err := readProjectJSON[release.UserValidation](root, validationPath)
		if err != nil {
			return err
		}
		result := release.VerifyCandidate(candidate, input, validation)
		result.ToolVersion = version
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root")
	command.Flags().StringVar(&candidatePath, "candidate", ".harness/local/release/candidate.json", "candidate path relative to root")
	command.Flags().StringVar(&inputPath, "input", "", "legacy current release input JSON; normal use recollects actual state")
	command.Flags().StringVar(&validationPath, "validation", ".harness/local/release/user-validation.json", "user validation path relative to root")
	return command
}

func readProjectJSON[T any](root, path string) (T, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, filepath.FromSlash(path))
	}
	return readJSON[T](path)
}

func readValidationEvidence(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > maxUserValidationEvidenceBytes {
		return nil, fmt.Errorf("user validation evidence must be a non-empty regular non-symlink file no larger than %d bytes", maxUserValidationEvidenceBytes)
	}
	return os.ReadFile(path)
}

func readJSON[T any](path string) (T, error) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	return schema.DecodeJSON[T](data)
}
