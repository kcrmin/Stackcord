package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/controlcenter"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	gh "github.com/kcrmin/Stackcord/cli/internal/github"
	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/provider"
	"github.com/kcrmin/Stackcord/cli/internal/work"
	"github.com/kcrmin/Stackcord/cli/internal/workspace"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

var issueURLPattern = regexp.MustCompile(`^https://github\.com/([A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*)/issues/([1-9][0-9]*)$`)

func parseGitHubIssueURL(value string) (string, int, error) {
	match := issueURLPattern.FindStringSubmatch(value)
	if match == nil {
		return "", 0, errors.New("mapping must use an exact github.com issue URL; migrate legacy numeric IDs explicitly")
	}
	number, err := strconv.Atoi(match[2])
	return match[1], number, err
}
func issueURL(repo string, number int) string {
	return fmt.Sprintf("https://github.com/%s/issues/%d", repo, number)
}

func readGitHubObservation(ctx context.Context, mapping provider.Mapping, definition work.Definition, now time.Time, transport gh.Transport) (externalProviderObservation, error) {
	repo, number, err := parseGitHubIssueURL(mapping.ItemID)
	if err != nil {
		return externalProviderObservation{}, err
	}
	client, err := gh.New(repo, transport)
	if err != nil {
		return externalProviderObservation{}, err
	}
	if _, err = client.Identity(ctx); err != nil {
		return externalProviderObservation{}, err
	}
	if _, err = client.Repository(ctx); err != nil {
		return externalProviderObservation{}, err
	}
	issue, err := client.Issue(ctx, number)
	if err != nil {
		return externalProviderObservation{}, err
	}
	if issue.HTMLURL != mapping.ItemID {
		return externalProviderObservation{}, errors.New("GitHub issue URL changed")
	}
	dependencies, err := client.IssueDependencies(ctx, number)
	if err != nil {
		return externalProviderObservation{}, err
	}
	if len(issue.Assignees) > 1 {
		return externalProviderObservation{}, errors.New("multiple GitHub assignees cannot establish one work owner")
	}
	snapshot := provider.Snapshot{SchemaVersion: 1, Provider: "github", ItemID: issue.HTMLURL, DefinitionFingerprint: definition.Fingerprint,
		FetchedAt: now, Source: "connector-live", Dependencies: []string{}, Capabilities: provider.Capabilities{Dependencies: true, Claim: "advisory", Revision: true}}
	switch issue.State {
	case "closed":
		snapshot.Status = "done"
	case "open":
		snapshot.Status = "ready"
		if definition.Readiness == work.Draft {
			snapshot.Status = "proposed"
		}
	default:
		return externalProviderObservation{}, errors.New("unknown GitHub issue state")
	}
	if len(issue.Assignees) == 1 {
		snapshot.Owner = issue.Assignees[0].Login
		if snapshot.Status == "ready" {
			snapshot.Status = "in_progress"
		}
	}
	for _, dependency := range dependencies {
		if _, _, err := parseGitHubIssueURL(dependency.HTMLURL); err != nil {
			return externalProviderObservation{}, err
		}
		snapshot.Dependencies = append(snapshot.Dependencies, dependency.HTMLURL)
	}
	raw, _ := json.Marshal(struct {
		Issue        gh.Issue
		Dependencies []gh.Issue
	}{issue, dependencies})
	digest := sha256.Sum256(raw)
	snapshot.RawHash = "sha256:" + hex.EncodeToString(digest[:])
	snapshot.Revision = snapshot.RawHash
	state := provider.Reconcile(provider.Expectation{WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Dependencies: definition.Dependencies}, mapping, snapshot, now)
	return externalProviderObservation{Mapping: mapping, Snapshot: snapshot, State: state}, nil
}

func newGitHubIssuesCommand(version string, jsonOutput *bool) *cobra.Command {
	return newGitHubIssuesWithTransport(version, jsonOutput, nil)
}
func newGitHubIssuesWithTransport(version string, jsonOutput *bool, transport gh.Transport) *cobra.Command {
	var root, repo string
	parent := &cobra.Command{Use: "issues", Short: "Read and explicitly connect GitHub Issues to canonical work"}
	parent.PersistentFlags().StringVar(&root, "root", ".", "project root or child path")
	parent.PersistentFlags().StringVar(&repo, "repo", "", "GitHub owner/name")
	connect := func(ctx context.Context) (*gh.Client, error) {
		if repo == "" {
			repo = controlcenter.DetectRepository(ctx, root)
		}
		c, e := gh.New(repo, transport)
		if e != nil {
			return nil, e
		}
		if _, e = c.Identity(ctx); e != nil {
			return nil, e
		}
		if _, e = c.Repository(ctx); e != nil {
			return nil, e
		}
		return c, nil
	}
	list := &cobra.Command{Use: "list", Short: "Read issues directly through the authenticated GitHub client", RunE: func(cmd *cobra.Command, _ []string) error {
		c, e := connect(cmd.Context())
		if e != nil {
			return e
		}
		issues, e := c.Issues(cmd.Context(), "all")
		if e != nil {
			return e
		}
		result := issueResult(version, "issues.list", "GitHub issues read live.")
		for _, issue := range issues {
			result.Facts = append(result.Facts, domain.Item{Code: "github.issue", Message: issue.Title, Refs: []string{issue.HTMLURL, issue.State}})
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	var id, title, bodyFile string
	var createApply bool
	create := &cobra.Command{Use: "create", Short: "Preview or explicitly create one retry-safe GitHub issue", RunE: func(cmd *cobra.Command, _ []string) error {
		if id == "" || title == "" || bodyFile == "" {
			return errors.New("id, title and body-file required")
		}
		body, e := os.ReadFile(bodyFile)
		if e != nil {
			return e
		}
		if repo == "" {
			repo = controlcenter.DetectRepository(cmd.Context(), root)
		}
		if _, e = gh.New(repo, transport); e != nil {
			return e
		}
		result := issueResult(version, "issues.create", "Issue creation preview; use --apply to create the remote issue.")
		result.Facts = []domain.Item{{Code: "github.issue-title", Message: title, Refs: []string{id, repo}}, {Code: "github.issue-body", Message: string(body)}}
		if createApply {
			c, e := connect(cmd.Context())
			if e != nil {
				return e
			}
			issue, e := c.EnsureIssue(cmd.Context(), id, title, string(body))
			if e != nil {
				return e
			}
			result.Summary = "GitHub issue created or recovered by stable ID."
			result.Evidence = []domain.Item{{Code: "github.issue", Message: issue.HTMLURL}}
		}
		return writeResult(cmd, *jsonOutput, result)
	}}
	create.Flags().StringVar(&id, "id", "", "stable retry ID")
	create.Flags().StringVar(&title, "title", "", "issue title")
	create.Flags().StringVar(&bodyFile, "body-file", "", "UTF-8 issue body file")
	create.Flags().BoolVar(&createApply, "apply", false, "create the remote issue explicitly")
	for _, mode := range []string{"link", "migrate"} {
		mode := mode
		var workID, mappingFile string
		var number int
		var apply bool
		command := &cobra.Command{Use: mode, Short: "Preview or apply stable issue mappings without replacing work definitions", RunE: func(cmd *cobra.Command, _ []string) error {
			located, e := workspace.FindRoot(cmd.Context(), root)
			if e != nil {
				return e
			}
			definitions, e := work.LoadDefinitions(located.Path)
			if e != nil {
				return e
			}
			links := map[string]int{}
			if mode == "link" {
				if workID == "" || number < 1 {
					return errors.New("id and positive number required")
				}
				links[workID] = number
			} else {
				data, e := os.ReadFile(mappingFile)
				if e != nil {
					return e
				}
				if e = json.Unmarshal(data, &links); e != nil {
					return e
				}
				if len(links) != len(definitions) {
					return errors.New("migration requires an issue mapping for every work definition")
				}
			}
			if _, e := connect(cmd.Context()); e != nil {
				return e
			}
			plan, e := githubMappingPlan(cmd.Context(), located.Path, repo, definitions, links, transport)
			if e != nil {
				return e
			}
			result := issueResult(version, "issues."+mode, "GitHub mapping preview; definitions and semantic dependencies are preserved.")
			for _, file := range plan.Files {
				result.Changes = append(result.Changes, domain.Item{Code: "github.mapping-planned", Message: file.Path})
			}
			if apply {
				result = operation.Apply(cmd.Context(), plan)
				result.ToolVersion = version
				result.Command = "issues." + mode
			}
			return writeResult(cmd, *jsonOutput, result)
		}}
		command.Flags().BoolVar(&apply, "apply", false, "apply local mappings and select GitHub as live source")
		if mode == "link" {
			command.Flags().StringVar(&workID, "id", "", "canonical work ID")
			command.Flags().IntVar(&number, "number", 0, "existing GitHub issue number")
		} else {
			command.Flags().StringVar(&mappingFile, "mapping", "", "JSON object mapping every work ID to an issue number")
		}
		parent.AddCommand(command)
	}
	parent.AddCommand(list, create)
	return parent
}

func issueResult(version, command, summary string) domain.Result {
	return domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: command, OperationID: "github-issues-read-only", Status: domain.StatusPassed, ExitCode: domain.ExitSuccess, Summary: summary}
}

func githubMappingPlan(ctx context.Context, root, repo string, definitions []work.Definition, links map[string]int, transport gh.Transport) (operation.Plan, error) {
	plan := operation.Plan{Root: root, ID: "github-issues-migration"}
	if len(links) == 0 {
		return plan, errors.New("at least one work mapping required")
	}
	selected, err := loadTaskProvider(root)
	if err != nil {
		return plan, err
	}
	seen := map[int]bool{}
	for id, number := range links {
		if _, found := findDefinition(definitions, id); !found || number < 1 || seen[number] {
			return plan, errors.New("unknown work ID, invalid issue number or duplicate issue mapping")
		}
		seen[number] = true
	}
	for _, definition := range definitions {
		number, ok := links[definition.ID]
		if !ok {
			continue
		}
		mapping := provider.Mapping{SchemaVersion: 1, WorkID: definition.ID, DefinitionFingerprint: definition.Fingerprint, Provider: "github", ItemID: issueURL(repo, number), DependencyItems: map[string]string{}}
		for _, dependency := range definition.Dependencies {
			if target, ok := links[dependency]; ok {
				mapping.DependencyItems[dependency] = issueURL(repo, target)
			} else {
				path := filepath.Join(root, ".harness", "work", "mappings", dependency+".yaml")
				if e := provider.ValidateCanonicalMappingLocation(root, path); e != nil {
					return plan, e
				}
				existing, e := provider.LoadMapping(path)
				if e != nil {
					return plan, e
				}
				if existing.Provider != "github" {
					return plan, errors.New("dependency must already map to GitHub")
				}
				mapping.DependencyItems[dependency] = existing.ItemID
			}
		}
		observation, e := readGitHubObservation(ctx, mapping, definition, time.Now().UTC(), transport)
		if e != nil {
			return plan, e
		}
		if observation.State.Confidence != provider.Confirmed {
			return plan, errors.New("live GitHub issue dependencies or state differ from canonical work; reconcile on GitHub first")
		}
		data, e := yaml.Marshal(mapping)
		if e != nil {
			return plan, e
		}
		snapshot, e := yaml.Marshal(observation.Snapshot)
		if e != nil {
			return plan, e
		}
		partial, e := providerReconcilePlan(root, mapping, data, snapshot)
		if e != nil {
			return plan, e
		}
		plan.Files = append(plan.Files, partial.Files...)
	}
	selected.Provider = "github"
	selected.LiveStatusSource = "github"
	data, err := yaml.Marshal(selected)
	if err != nil {
		return plan, err
	}
	plan.Files = append(plan.Files, operation.FileChange{Path: ".harness/work/provider.yaml", Content: data, Mode: 0o644})
	digest := sha256.Sum256(data)
	for _, file := range plan.Files {
		digest = sha256.Sum256(append(digest[:], file.Content...))
	}
	plan.ID += "-" + hex.EncodeToString(digest[:8])
	plan.InitialStateFingerprint, err = operation.StateFingerprint(plan)
	return plan, err
}
