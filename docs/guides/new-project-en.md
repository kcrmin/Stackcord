# New project journey

1. Say “Start a new service.” Discovery is checkpointed under `.harness-drafts/<id>/` as a normalized summary, decisions, and open questions—never raw conversation.
2. Answer one recommended multiple-choice question at a time or supply a free-form answer. The AI explores roles, value, complete journeys, success/failure policy, quality, security, operations, and important possibilities the user may not have considered.
3. Approve the product summary and repository name. `project init` creates a framework-neutral root with the repo-local Skill, harness, specs, contracts, and docs.
4. Choose technologies only after required capabilities and constraints are known. Verify current maintenance, security, and release status at selection time.
5. Establish product-wide executable UI coverage. External mockups can be imported as reference, seed, or canonical sources.
6. Define contracts, failure behavior, DBML, and shared implementation boundaries. Then build small vertical slices with red-green-refactor evidence and conflict claims.
7. Integrate compatibility-first, harden production, freeze one RC, have the user validate that exact digest, and only then publish.

The AI checkpoints after each material decision, so context compression or a new contributor does not restart discovery.
