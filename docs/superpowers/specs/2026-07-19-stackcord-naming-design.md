# Stackcord product naming

## Decision

The public product name is **Stackcord**, pronounced “stack-cord” in English and
“스택코드” in Korean. The canonical lowercase identifier is `stackcord`.

Stackcord describes a full stack whose product meaning, repositories, people,
and coding agents remain connected without implying that the product replaces
every development tool.

## Category and message

Canonical Korean description:

> 질문으로 서비스를 정의하고, 알맞은 개발 방식과 협업 도구를 선택하며,
> 풀스택 프로젝트의 맥락을 release까지 이어주는 협업 하네스.

Canonical English description:

> A question-driven collaboration harness that connects the right development
> practices and collaboration tools and keeps full-stack project context coherent
> through release.

Short English tagline:

> From questions to release, keep the whole stack coherent.

Question-Driven Development is written out on first use. `QDD` may be used only
after that explanation because the acronym has other meanings. “Development
practices and collaboration tools” is preferred over the ambiguous phrase
“development tools.”

## Positioning

Stackcord does not recreate Superpowers, BMAD, Beads, task providers, UI tools,
or database visualization tools. It identifies a capability gap when that gap
matters, compares realistic options, connects only the option the user selects,
and prevents the external tool from silently replacing canonical project truth.

Stackcord continues to own durable service meaning, workspace topology, semantic
work reservation, actual Git and submodule identity, stale detection, and the
exact release candidate. External tools keep their natural, limited authority.

The defensible product boundary is the combination of:

- normalized question-driven service discovery;
- framework-neutral UI, frontend, backend, contract, and DBML coordination;
- multi-repository Git, submodule, and worktree continuity;
- semantic conflict reservation beyond file paths;
- clone- and compaction-safe context reconstruction; and
- one exact candidate shared by technical and user verification.

## Naming surface

The implementation should use `Stackcord` for the display name and `stackcord`
for the Plugin identifier and user-facing CLI executable. Generated project
formats remain product-neutral: `.harness/`, `specs/`, `contracts/`, and the
repo-local fallback are not renamed merely for branding. This preserves clone
continuity, avoids needless migration, and keeps the generated project usable
without the Plugin.

The public repository owner, final module import path, domains, marketplace
account, and legal trademark clearance remain publication decisions. Their
absence does not block local product renaming and verification.

