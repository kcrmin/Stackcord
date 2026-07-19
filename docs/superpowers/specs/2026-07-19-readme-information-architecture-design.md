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
6. A short distinction between product specifications and implementation contracts, using the reservation policy from the conversation
7. Installation through Codex first, with a manual fallback
8. Only the generated files users may need to recognize
9. A compact guide index

## Editing rules

- Target readers are junior developers, not children and not Stackcord maintainers.
- Prefer plain development terminology; explain Stackcord-specific terms at first use.
- Keep user-facing natural-language usage before CLI or internal paths.
- Keep all nine established problem statements, QDD, full-stack, tool discovery, context recovery, product authority, and framework neutrality.
- Retain the five-Skill model without requiring users to memorize Skill names.
- Keep detailed Git-state fields, collision matrices, branch conventions, provider enforcement, and deterministic check inventories in the linked task-management, submodule, governance, and troubleshooting guides instead of the README.
- Explain `specs/` as what and why the product does something, and `contracts/` as the obligations implementations must obey. Show both sides of one reservation-confirmation example.
- Keep Korean and English headings, tables, examples, and claims semantically aligned.
- Keep both READMEs concise without removing essential product behavior.

## Verification

- Run documentation parity and Plugin validation.
- Check every local README link.
- Confirm the package-facing README still describes only shipped capabilities.
- Run `git diff --check`.
