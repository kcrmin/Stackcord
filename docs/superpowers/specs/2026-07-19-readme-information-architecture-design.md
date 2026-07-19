# README Information Architecture Design

## Goal

Help a recently graduated developer understand within three minutes what Stackcord solves, how it is used through Codex, and how it coordinates a multi-repository full-stack project.

## Chosen approach

Use a problem-first README. Preserve the nine established product problems near the top, then explain the product once through concrete conversations and one end-to-end flow. Do not repeat the same capability in separate problem, feature, and benefit tables.

## Korean and English structure

1. Product name, one-line description, and language link
2. What Stackcord is and the boundary between Skill judgment and deterministic verification
3. Nine problems and the corresponding change Stackcord provides
4. Three short conversation examples:
   - service discovery with recommended choices and free input
   - external-tool recommendation at the point of need
   - protected policy change proposed by a contributor
5. One question-to-release workflow table and one compact diagram
6. Multi-repository Git/submodule/worktree collaboration structure
7. What is verified deterministically and what still depends on a Git provider
8. Installation through Codex first, with a manual fallback
9. Only the generated files users may need to recognize
10. A compact guide index

## Editing rules

- Target readers are junior developers, not children and not Stackcord maintainers.
- Prefer plain development terminology; explain Stackcord-specific terms at first use.
- Keep user-facing natural-language usage before CLI or internal paths.
- Keep all nine established problem statements, QDD, full-stack, tool discovery, context recovery, product authority, and framework neutrality.
- Retain the five-Skill model without requiring users to memorize Skill names.
- Avoid unsupported claims: Stackcord detects and gates product authority, while provider branch rules enforce the final merge restriction.
- Keep Korean and English headings, tables, examples, and claims semantically aligned.
- Reduce both READMEs from 196 lines to roughly 130–160 lines without removing essential behavior.

## Verification

- Run documentation parity and Plugin validation.
- Check every local README link.
- Confirm the package-facing README still describes only shipped capabilities.
- Run `git diff --check`.
