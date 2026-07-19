# Stackcord README information design

## Audience and outcome

The README is for a developer who has recently graduated and understands Git,
frontend/backend boundaries, tests, and pull requests, but has not used an AI
development harness, submodule orchestration, semantic work reservation, or
release-candidate identity.

Within five minutes, that reader must understand:

1. what problem Stackcord solves;
2. what they say to Codex and what Stackcord does in response;
3. what is stored in the project;
4. how multiple repositories and contributors stay coordinated;
5. how complementary practices and collaboration tools are recommended; and
6. how to install the Plugin and begin without learning internal CLI commands.

The tone is concise and professional. Explain unfamiliar terms once, but do not
use childlike metaphors or omit engineering boundaries.

## Information order

Both Korean and English READMEs use this order:

1. Stackcord name, category, and one-sentence value;
2. concrete problems and before/after outcomes;
3. a short Question-Driven Development conversation with recommended A/B/C
   choices and free-form input;
4. the files produced by that conversation;
5. one connected question-to-release flow using a compact table and one Mermaid
   diagram;
6. a short external-practice or collaboration-tool recommendation conversation;
7. a feature table organized by situation, action, and result;
8. multi-repository Git, submodule, worktree, reservation, and integration
   behavior;
9. installation by asking Codex to use the public repository link, with manual
   marketplace commands as a fallback;
10. generated project structure, five Skills, core versus strict release, and
    links to detailed guides.

The README should stay roughly within 140–180 lines per language. Tables replace
repeated prose when they improve scanning.

## User interaction

The main example starts with “Start a reservation service with me.” Stackcord
asks one material question, presents two or three mutually exclusive choices,
puts the recommended choice first, and accepts free-form input. The example then
shows how the answer becomes a normalized service policy, scenario, decision, or
open question rather than raw conversation.

The external-tool example begins only after a concrete need appears, such as
three contributors splitting UI, frontend, and backend work. Stackcord inspects
the existing repository and explains a short choice such as GitHub Issues plus
Git reservation, Beads plus Git reservation, or Git-local. Superpowers and BMAD
are described as optional development practices, not bundled dependencies or
project truth.

## Installation and implementation boundary

The primary installation path is a natural-language request containing the
public Stackcord repository or marketplace link. Codex may configure the Git
marketplace and installation command, while the user completes any required
security confirmation and starts a new chat when needed. Manual commands are a
fallback, not the first experience.

End users do not need Go or direct CLI knowledge. The README explains only that
Stackcord includes a deterministic local verifier for Git, submodules, conflict
scope, stale state, and exact release identity. Source builds and contributor
verification remain in `CONTRIBUTING.md`.

Generated project formats remain product-neutral. The README retains the tested
paths and public contracts required for Plugin-less continuation.

## Acceptance criteria

- A reader can state the product purpose without reading the detailed guides.
- QDD, full-stack scope, external-tool selection, context recovery, and exact
  release verification are all visible without repetitive sections.
- The examples show where the conversation happens and what changes afterward.
- The Go CLI is not presented as a user prerequisite.
- Installation clearly separates the natural-language path from manual fallback.
- Korean and English structures match semantically.
- Documentation parity, public-contract validation, and documented-command
  validation continue to pass.

