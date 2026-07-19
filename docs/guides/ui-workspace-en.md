# UI workspace and external mockups

`ui/` is an optional editable UI baseline. It owns screens, states, interactions, tokens, accessibility, approved assets, and provenance. `frontend/` implements an exact UI baseline commit as the production product.

## When should it be separate?

Use a `ui/` submodule when UI has independent ownership, history, permissions, or review cadence. Register an ordinary directory when that is simpler for a small team or single repository. Select a framework or executable prototype only when it is actually needed.

## A. No UI exists yet

Establish product roles, journeys, and UI coverage, then create small flow slices in the UI workspace. The team may choose Figma, Penpot, a UI Skill such as MengTo/Skills, or direct editing. External-tool output is a candidate; the approved `ui/` commit is the baseline.

## B. A partial mockup exists

Inspect it as a `seed` and compare it with current UI. Bring in the whole source or selected files, then edit them as ordinary files. If existing files or product meaning conflict, do not overwrite them; decide which behavior remains first.

```text
stackcord ui import --root . --archive mockup.zip --id ui.external.checkout --authority seed --license MIT --apply
stackcord ui promote --root . --id ui.external.checkout --workspace workspace.ui --mode selected --path screens/checkout.html --apply
```

## C. An external design is already approved

Register it as `canonical` and bring in an appropriate whole or selected export. Canonical means current decision authority, not immutability. The team can add missing error, loading, permission, responsive, and accessibility states, then commit a new baseline.

## Create the UI submodule

First create the UI remote in the selected provider. The CLI safely adds only an existing remote; it does not create remotes, commit, or push.

```text
stackcord git submodule add --root . --remote https://example.com/team/product-ui.git --path ui --apply
stackcord workspace register --root . --id workspace.ui --kind submodule --path ui --remote https://example.com/team/product-ui.git --responsibility ui-baseline --consumer workspace.frontend --initialize ui --apply
```

Use `--kind directory` when the UI workspace is not a submodule.

## Connect the baseline to frontend work

Edit UI files, commit and push with ordinary Git conventions, then bind the baseline.

```text
stackcord ui baseline bind --root . --id ui.baseline.checkout --workspace workspace.ui --source ui.external.checkout --ref ui.checkout --consumer workspace.frontend --apply
```

Frontend work records this baseline fingerprint. A changed UI commit, source, or root pointer makes older frontend work and evidence stale. For a submodule, integrate the baseline record and new UI gitlink as one reviewable root change.

## Conflicts and safety

- Import checks path traversal, symlinks, executables, secret-like content, size, and license.
- Quarantine is an internal temporary safety boundary, not user-managed storage.
- Promotion never silently overwrites edited UI files.
- Different files still require coordination when they change the same flow, state, token, or policy.
- A UI baseline must be a clean commit recoverable from its remote.
- Frontend TDD evidence covers interaction, failure, and accessibility—not screenshots alone.

## Continue after clone

When the user says “Continue this project,” the Skill checks UI checkout, source authority, baseline commit, root pointer, and frontend fingerprint together. A missing submodule is initialized only at the exact root pointer. Dirty, local-only, or diverged state is reported rather than discarded.
