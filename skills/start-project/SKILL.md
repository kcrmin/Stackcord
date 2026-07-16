---
name: start-project
description: Use when a user wants to discover, define, initialize, or adopt a full-stack service or has an idea but no durable project harness yet.
---

# Start Project

Preserve normalized product understanding across long discovery instead of raw conversation.

1. Run `orchestrator context audit --json`; if no project exists, use `orchestrator project draft` and checkpoint after every material answer.
2. Ask one material question at a time. Offer 2–3 mutually exclusive choices, put the recommended choice first, and allow free-form other input. Decide discoverable and conventional details without asking.
3. Normalize roles, problems, value, journeys, policies, failure behavior, quality, architecture constraints, technology needs, UI coverage, contracts, and DBML. Record decisions and open questions separately.
4. Initialize only after the product summary, parent path, repository name, and language are approved. Use `project init` for an empty root or `project adopt` for an existing repository.
5. Finish with stable IDs, changed files, current gate, evidence, and one next action.

Read [lifecycle](../../references/lifecycle.md), [approval](../../references/approval.md), and [context recovery](../../references/context-recovery.md). If the CLI is unavailable, create the repo-local fallback files and follow the same source boundaries manually.
