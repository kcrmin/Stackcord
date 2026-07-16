package command

import (
	"encoding/json"
	"os"

	"fullstack-orchestrator/cli/internal/domain"
	"fullstack-orchestrator/cli/internal/operation"
	"fullstack-orchestrator/cli/internal/policy"
	"fullstack-orchestrator/cli/internal/release"
	"fullstack-orchestrator/cli/internal/schema"
	"github.com/spf13/cobra"
)

func newVerifyCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "verify", Short: "Verify production readiness gates"}
	var gatesPath string
	command := &cobra.Command{Use: "release", RunE: func(cmd *cobra.Command, _ []string) error {
		gates, err := readJSON[release.Gates](gatesPath)
		if err != nil {
			return err
		}
		result := release.Verify(gates)
		result.ToolVersion = version
		return writeResult(cmd, *jsonOutput, result)
	}}
	command.Flags().StringVar(&gatesPath, "gates", "", "production gates JSON")
	_ = command.MarkFlagRequired("gates")
	parent.AddCommand(command)
	return parent
}

func newRCCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "rc", Short: "Create and verify immutable release candidates"}
	parent.AddCommand(newRCCreate("rc.create", version, jsonOutput))
	var candidatePath, inputPath string
	verify := &cobra.Command{Use: "verify", RunE: func(cmd *cobra.Command, _ []string) error {
		candidate, err := readJSON[release.Candidate](candidatePath)
		if err != nil {
			return err
		}
		input, err := readJSON[release.Input](inputPath)
		if err != nil {
			return err
		}
		result := release.VerifyCandidate(candidate, input)
		result.ToolVersion = version
		return writeResult(cmd, *jsonOutput, result)
	}}
	verify.Flags().StringVar(&candidatePath, "candidate", "", "candidate JSON")
	verify.Flags().StringVar(&inputPath, "input", "", "current release input JSON")
	_ = verify.MarkFlagRequired("candidate")
	_ = verify.MarkFlagRequired("input")
	parent.AddCommand(verify)
	return parent
}

func newReleaseCommand(version string, jsonOutput *bool) *cobra.Command {
	parent := &cobra.Command{Use: "release", Short: "Prepare or plan an exactly approved production release"}
	parent.AddCommand(newRCCreate("release.prepare", version, jsonOutput))
	var candidatePath, consentPath string
	publish := &cobra.Command{Use: "publish", RunE: func(cmd *cobra.Command, _ []string) error {
		candidate, err := readJSON[release.Candidate](candidatePath)
		if err != nil {
			return err
		}
		consent := policy.Consent{}
		if consentPath != "" {
			consent, err = readJSON[policy.Consent](consentPath)
			if err != nil {
				return err
			}
		}
		plan, result := release.PlanPublish(candidate, consent)
		result.ToolVersion = version
		if result.Status == domain.StatusPassed {
			planned := planResult(version, "release.publish", plan, result.Summary)
			planned.Approval = result.Approval
			result = planned
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	publish.Flags().StringVar(&candidatePath, "candidate", "", "approved candidate JSON")
	publish.Flags().StringVar(&consentPath, "approval-receipt", "", "exact class D approval receipt JSON")
	_ = publish.MarkFlagRequired("candidate")
	parent.AddCommand(publish)
	return parent
}

func newRCCreate(commandName, version string, jsonOutput *bool) *cobra.Command {
	var root, inputPath, outputPath string
	var apply bool
	name := "create"
	if commandName == "release.prepare" {
		name = "prepare"
	}
	command := &cobra.Command{Use: name, RunE: func(cmd *cobra.Command, _ []string) error {
		input, err := readJSON[release.Input](inputPath)
		if err != nil {
			return err
		}
		candidate, result := release.CreateCandidate(input)
		result.ToolVersion, result.Command = version, commandName
		if result.Status != domain.StatusPassed {
			return writeResult(cmd, *jsonOutput, result)
		}
		data, _ := json.MarshalIndent(candidate, "", "  ")
		plan := operation.Plan{ID: "rc-" + input.Version, Root: root, Files: []operation.FileChange{{Path: outputPath, Content: append(data, '\n'), Mode: 0o644}}}
		plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
		if err != nil {
			return err
		}
		if apply {
			applied := operation.Apply(cmd.Context(), plan)
			applied.ToolVersion, applied.Command = version, commandName
			applied.Evidence = append(applied.Evidence, result.Evidence...)
			return writeResult(cmd, *jsonOutput, applied)
		}
		planned := planResult(version, commandName+".plan", plan, "Immutable release candidate write is planned; no file changed.")
		planned.Evidence = result.Evidence
		return writeResult(cmd, *jsonOutput, planned)
	}}
	command.Flags().StringVar(&root, "root", ".", "project root")
	command.Flags().StringVar(&inputPath, "input", "", "verified release input JSON")
	command.Flags().StringVar(&outputPath, "output", ".harness/state/release-candidate.json", "candidate path relative to root")
	command.Flags().BoolVar(&apply, "apply", false, "write the candidate atomically")
	_ = command.MarkFlagRequired("input")
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
