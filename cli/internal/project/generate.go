package project

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kcrmin/Stackcord/cli/internal/operation"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
	"go.yaml.in/yaml/v3"
)

const managedBegin = "<!-- stackcord:begin -->"
const managedEnd = "<!-- stackcord:end -->"

func render(request InitRequest) ([]operation.FileChange, error) {
	name := request.Name
	if name == "" {
		name = request.ProjectID
	}
	files := map[string]string{
		"README.md":      "# " + name + "\n\n" + managedSection("## Project harness\n\nAsk your AI assistant what to do next. It will read `.harness/entry.md`, inspect actual Git state, and continue from canonical specs and contracts.\n\nIf the project uses an independent editable UI baseline, it lives in a declared `ui/` directory or submodule. External mockups can be inspected, brought into that workspace, edited normally, and committed before frontend implementation is bound to the exact baseline."),
		"AGENTS.md":      "# Agent entry\n\n" + managedSection("Before changing the project, read `.harness/entry.md` and refresh actual context. Product meaning lives in `specs/`; obligations live in `contracts/`; coordination state lives in `.harness/`. Protected product meaning stays a proposal until `stackcord governance check --json` confirms an assigned product authority through the selected Git review provider; Git user.name and user.email never prove authority.\n\nWhen `workspace.ui` is declared, recover its exact baseline and root pointer before frontend work. Optional UI tools create inputs; committed service specifications, contracts, and the UI baseline remain authoritative."),
		".editorconfig":  "root = true\n\n[*]\ncharset = utf-8\nend_of_line = lf\ninsert_final_newline = true\ntrim_trailing_whitespace = true\n",
		".gitattributes": "* text=auto eol=lf\n*.png binary\n*.jpg binary\n*.jpeg binary\n*.gif binary\n*.pdf binary\n",
		".gitignore":     ".harness/local/\n.harness-drafts/\n.env\n.env.*\n!.env.example\n*.log\n",
		".agents/skills/use-project-harness/SKILL.md":               repoSkill,
		".agents/skills/use-project-harness/references/fallback.md": fallbackReference,
		".harness/manifest.yaml":                                    fmt.Sprintf("schema_version: 1\nid: %s\nlocale: %s\ngenerated_by: stackcord\npaths:\n  specs: specs\n  contracts: contracts\n  docs: docs\n", request.ProjectID, request.Locale),
		".harness/governance.yaml":                                  "schema_version: 1\nenabled: false\nprovider: \"\"\nrepository: \"\"\nproduct_authorities: []\nprotected_kinds: [product, policy, business, contract]\napproval:\n  minimum: 1\n  authority_self_approval: true\n",
		".harness/entry.md":                                         harnessEntry,
		".harness/profile.yaml":                                     projectProfile,
		".harness/sources.yaml":                                     "schema_version: 1\nsources:\n  - id: source.git.local\n    kind: git\n    authority: actual_state\n    access: read\n",
		".harness/workspaces.yaml":                                  fmt.Sprintf("schema_version: 1\nproject_id: %s\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n", request.ProjectID),
		".harness/work/provider.yaml":                               "schema_version: 1\nprovider: git-local\nlive_status_source: git-local\nremote: origin\ncoordination_branch: coordination\n",
		"specs/index.md":                                            "# Product specifications\n\nApproved intent, roles, capabilities, journeys, policies, scenarios, quality, architecture, and UI baselines live here.\n",
		"contracts/registry.yaml":                                   "schema_version: 1\n# product, business, behavior, interface, and data obligations are registered here.\n# The source file is canonical; fingerprint drift marks every declared dependent stale.\ncontracts: []\n",
		"contracts/product/index.md":                                "# Product obligations\n\nService commitments, boundaries, and explicit non-goals.\n",
		"contracts/business/index.md":                               "# Business obligations\n\nEligibility, rules, invariants, rejection, and failure behavior.\n",
		"contracts/behaviors/index.md":                              "# Behavior obligations\n\nObservable success, rejection, and failure behavior.\n",
		"contracts/interfaces/index.md":                             "# Interface obligations\n\nAPI, event, error, timeout, retry, and compatibility behavior.\n",
		"contracts/data/index.md":                                   "# Data obligations\n\nOwnership, classification, retention, deletion, and migration behavior.\n",
		"docs/index.md":                                             "# Project documentation\n\nGuides, runbooks, troubleshooting, and generated summaries live here.\n",
	}
	if request.DraftRoot != "" {
		checkpoint, err := schema.LoadYAML[DiscoveryCheckpoint](filepath.Join(request.DraftRoot, "checkpoint.yaml"))
		if err != nil {
			return nil, fmt.Errorf("load approved discovery checkpoint: %w", err)
		}
		if err := validateCheckpoint(checkpoint); err != nil {
			return nil, err
		}
		addDiscoveryFiles(files, checkpoint)
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := make([]operation.FileChange, 0, len(paths))
	for _, path := range paths {
		result = append(result, operation.FileChange{Path: path, Content: []byte(strings.ReplaceAll(files[path], "\r\n", "\n")), Mode: 0o644})
	}
	return result, nil
}

func managedSection(body string) string {
	return managedBegin + "\n" + strings.TrimSpace(body) + "\n" + managedEnd + "\n"
}

func mergeManaged(existing, generated string) string {
	generatedBlock := generated[strings.Index(generated, managedBegin):]
	if start := strings.Index(existing, managedBegin); start >= 0 {
		if end := strings.Index(existing[start:], managedEnd); end >= 0 {
			return existing[:start] + generatedBlock + existing[start+end+len(managedEnd):]
		}
	}
	return strings.TrimRight(existing, "\r\n") + "\n\n" + generatedBlock
}

const repoSkill = `---
name: use-project-harness
description: Use when starting, continuing, changing, coordinating, recovering, or releasing work in this repository.
---

# Use Project Harness

Treat the user's natural-language request as the entry point; do not make them memorize commands or edit ` + "`.harness/`" + `. Read ` + "`.harness/entry.md`" + `, run ` + "`stackcord status --json`" + ` when available, and inspect actual Git, workspace, and submodule state. From a child repository, resolve the actual orchestration root before asserting service-wide context. Read only canonical sources related to the request. ` + "`specs/`" + ` owns product meaning; ` + "`contracts/`" + ` owns service purpose, commitments, non-goals, business rules, failure behavior, interfaces, and data obligations.

When discovering or redefining the product, treat the initial product request as the first material answer. Infer discoverable facts, checkpoint normalized meaning rather than raw dialogue, and verify a successful apply before asking the next material question. When choices help, use 2–3 exclusive options labeled A/B/C, put the recommended option first and mark it recommended, and accept either a letter or free-form input. Keep work management proportional: a small private local edit does not need a ticket or Git work reservation. For shared, long-lived, cross-workspace, or semantically risky work, the selected task source owns live status and the Git work reservation owns exclusive semantic scope. Re-read both, check path and meaning overlap, and set ownership and merge order before parallel work. Use conventional Git names without AI markers.

Use TDD for behavior, bugs, contracts, migrations, and UI interactions; exploratory spikes may stay unmerged until evidence exists. Keep coordination internals out of normal replies. If context was compacted, settled questions repeat, or sources disagree, run a context audit before mutation. Use core release normally and enable strict release only for an explicit organizational need. If the CLI is unavailable, follow ` + "`references/fallback.md`" + ` and state reduced verification.

Before changing service purpose, policy, business rules, contracts, or governance, run ` + "`stackcord governance check --json`" + `. If governance is enabled and the selected Git provider does not identify the current account as a product authority, keep the protected change as a proposal, prepare its tests and implementation normally, and use the chosen issue or PR workflow to request review. Git user.name and user.email are display metadata, not authority. Never mark a proposal approved from cached review data; integration and release require a fresh exact-commit approval.

When an editable UI workspace exists, inspect external sources before bringing accepted whole or selected files into it. Treat them as ordinary editable files, bind approved UI to an exact published commit, and ensure frontend work names that baseline fingerprint. UI creation tools are optional inputs, not canonical service state.
`

const fallbackReference = `# Plugin-less and CLI-less fallback

1. Treat the natural-language request as the entry point. Read ` + "`AGENTS.md`" + `, ` + "`.harness/entry.md`" + `, the manifest, workspaces, profile, and selected task source; do not ask the user to operate internal files.
2. From a child repository, locate the actual orchestration root. Inspect branch, dirty state, upstream, ahead/behind/diverged state, worktrees, workspace commits, remotes, and exact submodule pointers without mutation.
3. Read only related approved ` + "`specs/`" + `; product, business, behavior, interface, and data ` + "`contracts/`" + `; product-authority policy and fresh review state; external UI authority and exact UI baseline; current work definitions; and test evidence.
4. If an external task source is selected, refresh it with a real authenticated connector or CLI. Treat cached status as unknown. Recover a Git work reservation from the coordination branch, but do not present it as fresh external status.
5. Separate confirmed facts, stale derivations, unknown external state, blockers, active ownership, and local-only work. State one safe next action. Run a context audit when settled questions repeat or sources disagree.
6. A small private local edit needs no ticket or reservation. Before shared or risky work, define the service meaning, behavioral boundary, first failing test, semantic scope, owner, dependencies, and merge order; then synchronize the selected task source and Git work reservation.
7. If UI is independent, verify its clean commit, remote availability, source fingerprints, root pointer, and the baseline fingerprint used by frontend work. Require test and integration evidence before merge. Bind technical and user validation to one release candidate. Keep strict release optional.
8. Before protected product meaning becomes canonical, verify the exact commit and fingerprint through ` + "`stackcord governance check --json`" + `. An ordinary contributor may prepare a proposal and PR, but Git user.name or user.email never proves product authority. If the selected review provider is unavailable, report approval as unknown and do not integrate or release.

Without the CLI, fingerprint, divergence, atomic remote reservation, semantic-conflict, archive-safety, and exact release-identity verification has reduced coverage. Do not report those checks as passed.
`

const harnessEntry = `# Project harness entry

1. Find the orchestration root from the actual Git superproject first, then ` + "`.harness/manifest.yaml`" + `; a standalone child must use ` + "`.harness/bridge.yaml`" + ` and report incomplete service context.
2. Refresh filesystem, Git, workspace, submodule, work, spec, contract, and evidence state read-only.
3. Treat ` + "`specs/`" + ` as product meaning, ` + "`contracts/`" + ` as obligations, and ` + "`.harness/`" + ` as coordination state. Protected meaning requires the configured product authority's fresh exact-provider approval; a non-authority may only propose it.
4. Before implementation, identify the product slice, scenario, contract, failure behavior, failing TDD test, conflict scope, ownership, and merge order.
5. Never hide pull, rebase, stash, reset, clean, force-push, external write, install, or release actions.
6. If context was compacted or appears forgotten, run a full context audit before mutation.
`

const projectProfile = "schema_version: 1\ntdd: default\ngit:\n  collaboration: strongly_recommended\n  release: required\ntask_source: git-local\nrelease: core\n"

type generatedMetadata struct {
	SchemaVersion int      `yaml:"schema_version"`
	ID            string   `yaml:"id"`
	Kind          string   `yaml:"kind"`
	Status        string   `yaml:"status"`
	Revision      int      `yaml:"revision"`
	Refs          []string `yaml:"refs"`
}

func addDiscoveryFiles(files map[string]string, checkpoint DiscoveryCheckpoint) {
	files["specs/product/summary.md"] = discoveryDocument("decision.product.summary", "decision", "approved", nil, "# Product summary\n\n"+checkpoint.Summary)
	addFacts := func(directory, kind, status string, facts []DiscoveryFact) {
		for _, fact := range facts {
			files[filepath.ToSlash(filepath.Join(directory, fact.ID+".md"))] = discoveryDocument(fact.ID, kind, status, nil, "# "+fact.ID+"\n\n"+fact.Summary)
		}
	}
	addFacts("specs/product/roles", "role", "approved", checkpoint.Roles)
	addFacts("specs/product/journeys", "journey", "approved", checkpoint.Journeys)
	addFacts("specs/product/capabilities", "capability", "approved", checkpoint.Capabilities)
	addFacts("specs/policies", "policy", "approved", checkpoint.Policies)
	addFacts("specs/quality", "quality", "approved", checkpoint.Quality)
	addFacts("specs/architecture", "architecture", "proposed", checkpoint.TechnologyNeeds)
	addFacts("specs/product/assumptions", "decision", "proposed", checkpoint.Assumptions)
	addFacts("specs/product/open-questions", "decision", "unknown", checkpoint.OpenQuestions)
	for _, scenario := range checkpoint.Scenarios {
		body := fmt.Sprintf("# %s\n\nActor: %s\n\nTrigger: %s\n\nOutcome: %s\n\nFailure: %s", scenario.ID, scenario.Actor, scenario.Trigger, scenario.Outcome, scenario.Failure)
		files["specs/scenarios/"+scenario.ID+".md"] = discoveryDocument(scenario.ID, "scenario", "approved", []string{scenario.Actor}, body)
	}
	for _, coverage := range checkpoint.UICoverage {
		body := fmt.Sprintf("# %s\n\nRequired states: %s", coverage.ID, strings.Join(coverage.States, ", "))
		files["specs/ui/"+coverage.ID+".md"] = discoveryDocument(coverage.ID, "ui", "approved", []string{coverage.RoleID, coverage.JourneyID}, body)
	}
	for _, decision := range checkpoint.Decisions {
		body := fmt.Sprintf("# %s\n\nChoice: %s\n\nRationale: %s", decision.ID, decision.Choice, decision.Rationale)
		files["specs/product/decisions/"+decision.ID+".md"] = discoveryDocument(decision.ID, "decision", "approved", nil, body)
	}
}

func discoveryDocument(id, kind, status string, refs []string, body string) string {
	metadata, _ := yaml.Marshal(generatedMetadata{SchemaVersion: 1, ID: id, Kind: kind, Status: status, Revision: 1, Refs: emptyStrings(refs)})
	return "---\n" + string(metadata) + "---\n\n" + strings.TrimSpace(body) + "\n"
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
