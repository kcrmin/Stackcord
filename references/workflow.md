# Product workflow

The AI owns product conversation and judgment. The CLI owns observable state, safe local writes, fingerprints, conflict outcomes, Git compare-and-swap work reservations, and release identity.

1. Run combined status and inspect discoverable facts before asking questions.
2. Treat the initial product request as the first material answer. Persist its normalized meaning before the next question, then checkpoint every later material answer; store no raw conversation or tone. Discover one decision at a time. When choices help, use A/B/C with the recommended option first and marked, plus free-form input.
3. Define product-wide purpose, non-goals, roles, journeys, capabilities, service policies, failure behavior, quality, and UI coverage. Keep this baseline evolving through small role/domain/journey slices; do not use waterfall.
4. Treat service purpose, guarantees, prohibited behavior, business rules, authorization, failure, retry, compensation, API behavior, events, and data obligations as contracts. Keep readable policy meaning and machine interfaces distinct but impact-linked.
5. Accept external UI as `reference`, `seed`, or `canonical`. Even a broad UI baseline integrates through small owned changes.
6. Select technology only when product, quality, team, and operating needs justify it. Detect tools, verify current official evidence, compare 2–3 candidates, record the dated decision, and connect only the selected option.
7. Define shared behavioral interfaces and compatibility order before parallel implementation. Do not freeze unrelated internals.
8. Keep work management proportional. A small private local edit needs no ticket or reservation. For shared, long-lived, cross-workspace, or semantically risky work, choose exactly one live task source, save the executable checklist, refresh external state when applicable, then acquire the Git work reservation before creating a conventional branch/worktree.
9. Use TDD for behavior, bugs, contracts, migrations, and UI interactions. Bind failing and passing evidence to current meaning and commits.
10. Integrate compatible contracts, providers, consumers, UI, migrations, and the reviewed root submodule pointer in that order when applicable.
11. Prepare one exact release candidate. Bind technical checks and user validation to the same digest; publish only as a separate explicit action.

Reopen an earlier decision whenever later evidence invalidates it.
