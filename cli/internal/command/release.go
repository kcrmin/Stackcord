package command

import (
	"encoding/json"
	"os"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/release"
	"fullstack-orchestrator/cli/internal/schema"
	"github.com/spf13/cobra"
)

func newReleaseCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "release", Short: "Prepare and verify one exact production candidate"}
	parent.AddCommand(newReleasePrepare(version, jsonOutput), newReleaseVerify(version, jsonOutput))
	return parent
}

func newReleasePrepare(version string, jsonOutput *bool) *cobra.Command {
	var root, inputPath, outputPath string
	var apply bool
	command := &cobra.Command{Use: "prepare", RunE: func(cmd *cobra.Command, _ []string) error {
		input, err := readJSON[release.Input](inputPath)
		if err != nil {
			return err
		}
		candidate, result := release.CreateCandidate(input)
		result.ToolVersion = version
		if result.Status != domain.StatusPassed {
			return writeResult(cmd, *jsonOutput, result)
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
			return writeResult(cmd, *jsonOutput, applied)
		}
		planned := planResult(version, "release.prepare.plan", plan, "Release candidate write is planned; no file changed.")
		planned.Evidence = result.Evidence
		return writeResult(cmd, *jsonOutput, planned)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root")
	command.Flags().StringVar(&inputPath, "input", "", "release input JSON")
	command.Flags().StringVar(&outputPath, "output", ".harness/state/release-candidate.json", "candidate path relative to root")
	command.Flags().BoolVar(&apply, "apply", false, "write the candidate atomically")
	_ = command.MarkFlagRequired("input")
	return command
}

func newReleaseVerify(version string, jsonOutput *bool) *cobra.Command {
	var candidatePath, inputPath, validationPath string
	command := &cobra.Command{Use: "verify", RunE: func(cmd *cobra.Command, _ []string) error {
		candidate, err := readJSON[release.Candidate](candidatePath)
		if err != nil {
			return err
		}
		input, err := readJSON[release.Input](inputPath)
		if err != nil {
			return err
		}
		validation, err := readJSON[release.UserValidation](validationPath)
		if err != nil {
			return err
		}
		result := release.VerifyCandidate(candidate, input, validation)
		result.ToolVersion = version
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&candidatePath, "candidate", "", "candidate JSON")
	command.Flags().StringVar(&inputPath, "input", "", "current release input JSON")
	command.Flags().StringVar(&validationPath, "validation", "", "user validation JSON bound to the candidate digest")
	_ = command.MarkFlagRequired("candidate")
	_ = command.MarkFlagRequired("input")
	_ = command.MarkFlagRequired("validation")
	return command
}

func readJSON[T any](path string) (T, error) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	return schema.DecodeJSON[T](data)
}
