package project

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"fullstack-orchestrator/cli/internal/operation"
)

const managedBegin = "<!-- orchestrator:begin -->"
const managedEnd = "<!-- orchestrator:end -->"

func render(request InitRequest) []operation.FileChange {
	name := request.Name
	if name == "" {
		name = request.ProjectID
	}
	files := map[string]string{
		"README.md":      "# " + name + "\n\n" + managedSection("## Project harness\n\nAsk your AI assistant what to do next. It will read `.harness/entry.md`, inspect actual Git state, and continue from canonical specs and contracts."),
		"AGENTS.md":      "# Agent entry\n\n" + managedSection("Before changing the project, read `.harness/entry.md` and refresh actual context. Product meaning lives in `specs/`; obligations live in `contracts/`; coordination state lives in `.harness/`."),
		".editorconfig":  "root = true\n\n[*]\ncharset = utf-8\nend_of_line = lf\ninsert_final_newline = true\ntrim_trailing_whitespace = true\n",
		".gitattributes": "* text=auto eol=lf\n*.png binary\n*.jpg binary\n*.jpeg binary\n*.gif binary\n*.pdf binary\n",
		".gitignore":     ".harness/local/\n.harness-drafts/\n.env\n.env.*\n!.env.example\n*.log\n",
		".agents/skills/use-project-harness/SKILL.md":               repoSkill,
		".agents/skills/use-project-harness/references/fallback.md": fallbackReference,
		".harness/manifest.yaml":                                    fmt.Sprintf("schema_version: 1\nid: %s\nlocale: %s\ngenerated_by: orchestrator\npaths:\n  specs: specs\n  contracts: contracts\n  docs: docs\n", request.ProjectID, request.Locale),
		".harness/entry.md":                                         harnessEntry,
		".harness/sources.yaml":                                     "schema_version: 1\nsources:\n  - id: source.git.local\n    kind: git\n    authority: actual_state\n    access: read\n",
		".harness/workspaces.yaml":                                  "schema_version: 1\nworkspaces:\n  - id: workspace.root\n    kind: root\n    path: .\n    responsibilities: [orchestration]\n    dependencies: []\n",
		".harness/state/lifecycle.yaml":                             "schema_version: 1\ncurrent_stage: entry_diagnosis\nstages: []\n",
		".harness/state/baselines.yaml":                             "schema_version: 1\nbaselines: []\n",
		".harness/state/context-index.json":                         "{\n  \"schema_version\": 1,\n  \"index\": {}\n}\n",
		".harness/state/impact-graph.json":                          "{\n  \"schema_version\": 1,\n  \"impact\": {}\n}\n",
		".harness/state/release-candidate.yaml":                     "schema_version: 1\nstatus: absent\n",
		".harness/policies/development.yaml":                        developmentPolicy,
		".harness/policies/tdd.yaml":                                tddPolicy,
		".harness/policies/conflicts.yaml":                          conflictPolicy,
		".harness/policies/approvals.yaml":                          approvalPolicy,
		".harness/policies/security.yaml":                           securityPolicy,
		".harness/policies/release.yaml":                            releasePolicy,
		".harness/work/provider.yaml":                               "schema_version: 1\nprovider: git-local\nlive_status_source: git-local\n",
		".harness/work/links.yaml":                                  "schema_version: 1\nlinks: []\n",
		".harness/integrations/dbdiagram.yaml":                      "schema_version: 1\nenabled: false\ncanonical: contracts/data\nsecret_environment: DBDIAGRAM_TOKEN\n",
		".harness/integrations/git-host.yaml":                       "schema_version: 1\nprovider: auto\n",
		".harness/integrations/tasks.yaml":                          "schema_version: 1\nprovider: git-local\n",
		".harness/templates/work-item.yaml":                         "schema_version: 1\nid: work.<ulid>\nstatus: proposed\nrefs: []\ndependencies: []\n",
		".harness/templates/scope-claim.yaml":                       "schema_version: 1\nid: claim.<ulid>\npaths: []\ncontract_ids: []\nexpires_at: <rfc3339>\n",
		".harness/templates/change-proposal.yaml":                   "schema_version: 1\nid: change.<ulid>\nstatus: proposed\nrefs: []\nworkspace_order: []\nverification: []\nrollback: []\n",
		".harness/templates/handoff.yaml":                           "schema_version: 1\nwork_id: work.<ulid>\ncurrent_state: ''\nnext_action: ''\nevidence: []\n",
		".harness/templates/adr.md":                                 "---\nschema_version: 1\nid: decision.<id>\nkind: decision\nstatus: proposed\nrevision: 1\nrefs: []\n---\n\n# Decision\n",
		"specs/index.md":                                            "# Product specifications\n\nApproved intent, roles, capabilities, journeys, policies, scenarios, quality, architecture, and UI baselines live here.\n",
		"contracts/registry.yaml":                                   "schema_version: 1\ncontracts: []\n",
		"contracts/errors.yaml":                                     "schema_version: 1\nerrors: []\n",
		"docs/index.md":                                             "# Project documentation\n\nGuides, runbooks, troubleshooting, and generated summaries live here.\n",
	}
	for _, directory := range trackedDirectories {
		files[filepath.ToSlash(filepath.Join(directory, ".gitkeep"))] = ""
	}
	if request.DraftRoot != "" {
		files["specs/product/discovery-source.md"] = "# Discovery migration\n\nNormalized discovery was approved and migrated from the draft at `" + filepath.ToSlash(request.DraftRoot) + "`.\n"
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
	return result
}

var trackedDirectories = []string{
	".harness/state/gates", ".harness/work/items", ".harness/work/claims", ".harness/work/changes", ".harness/work/branches", ".harness/evidence/receipts", ".harness/evidence/gates",
	"specs/product/journeys", "specs/policies", "specs/scenarios", "specs/quality", "specs/architecture", "specs/ui",
	"contracts/services", "contracts/api", "contracts/events", "contracts/data", "contracts/schemas", "contracts/auth",
	"docs/guides", "docs/runbooks", "docs/troubleshooting", "docs/generated",
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

func marshalJSON(value any) []byte {
	data, _ := json.MarshalIndent(value, "", "  ")
	return append(data, '\n')
}

const repoSkill = `---
name: use-project-harness
description: Use when starting, resuming, planning, changing, integrating, or releasing work in this repository; refreshes durable project context before action.
---

# Use Project Harness

Read ` + "`.harness/entry.md`" + `, inspect actual Git/workspace state, and run ` + "`orchestrator context audit --json`" + ` when available. Read only the related specs, contracts, work claim, and evidence. Never treat chat memory, task titles, or generated summaries as product truth. If the CLI is unavailable, follow ` + "`references/fallback.md`" + `.
`

const fallbackReference = `# Context recovery fallback

1. Read ` + "`AGENTS.md`" + ` and ` + "`.harness/entry.md`" + `.
2. Inspect the current root, branch, dirty state, remotes, worktrees, and exact submodule pointers without mutation.
3. Read the current branch record and claim, then only referenced specs and contracts.
4. Compare source fingerprints to generated checkpoints; label stale and unknown state.
5. State the current gate, blockers, evidence, and one safe next action before mutation.
`

const harnessEntry = `# Project harness entry

1. Find the nearest ` + "`.harness/manifest.yaml`" + ` and establish repository trust.
2. Refresh filesystem, Git, workspace, submodule, work, spec, contract, and evidence state read-only.
3. Treat ` + "`specs/`" + ` as product meaning, ` + "`contracts/`" + ` as obligations, and ` + "`.harness/`" + ` as coordination state.
4. Before implementation, identify scenario, contract, failure behavior, TDD test, conflict scope, and merge order.
5. Never hide pull, rebase, stash, reset, clean, force-push, external write, install, or release actions.
6. If context was compacted or appears forgotten, run a full context audit before mutation.
`

const developmentPolicy = "schema_version: 1\nmain_protected: true\nbranch_pattern: '<type>/<description>'\ncommit_convention: conventional-commits\ndefault_merge: squash\npermanent_develop: false\n"
const tddPolicy = "schema_version: 1\nrequired: true\nsequence: [red, green, refactor]\nexceptions: [documentation, pure-design-assets, deterministic-generated-files, non-merged-spikes, formatting]\nevidence_required: true\n"
const conflictPolicy = "schema_version: 1\nscopes: [path, module, policy, scenario, contract, database-entity, migration-slot, ui-flow, dependency, workspace, root-pointer]\nlevels: [clear, coordinate, block, unknown]\nclaims_are_locks: false\n"
const approvalPolicy = "schema_version: 1\nclasses:\n  A: read-only\n  B: requested-local-write\n  C: shared-or-external-write\n  D: destructive-production-or-secret\nclass_d_always_exact: true\n"
const securityPolicy = "schema_version: 1\nsecrets_in_repository: false\nuntrusted_hooks: false\nexternal_import_quarantine: true\npath_escape: block\n"
const releasePolicy = "schema_version: 1\nimmutable_rc: true\nsame_artifact_user_validation: true\nrequired: [tests, security, licenses, sbom, signatures, rollback, user-confirmation]\n"
