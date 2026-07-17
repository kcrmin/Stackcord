# Product workflow

The AI owns conversation and judgment. The CLI owns observable state, safe local writes, fingerprints, conflict outcomes, and candidate identity.

1. Diagnose the repository before asking questions.
2. Discover one material product decision at a time and checkpoint normalized meaning after each answer.
3. Define product-wide roles, journeys, policies, failure behavior, quality, and UI coverage. Split delivery into small role/domain/journey changes and integrate continuously; do not use waterfall.
4. Select technology only after functional, quality, team, and operating needs are known. Check current official maintenance, security, and release information at selection time.
5. Establish shared behavioral interfaces and compatibility order before parallel implementation. Do not freeze every internal interface prematurely.
6. Use TDD by default for behavior, bugs, contracts, migrations, and UI interactions. Keep the failing test and passing evidence identifiable.
7. Integrate additive contracts, providers, consumers, UI connection, and the exact root submodule pointer in that order when applicable.
8. Prepare one immutable release candidate, complete technical checks, and bind user validation to that exact digest.

Reopen an earlier decision whenever later evidence invalidates it.
