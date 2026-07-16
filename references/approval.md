# Approval reference

- A: read-only discovery and diagnosis.
- B: local requested writes with a visible plan.
- C: shared repository, submodule, push, PR, or external-system writes using current scoped consent.
- D: destructive, production, credential, secret, or irreversible actions; require exact target confirmation every time.

Never infer broader authorization from a terminal condition. Never hide pull, rebase, stash, reset, clean, force-push, install, external write, or production action. Return the CLI approval reason and exact target to the user.
