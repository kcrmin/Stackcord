# Privacy

## Stored project knowledge

Discovery checkpoints contain normalized summaries, decisions, policies, scenarios, quality requirements, assumptions, and open questions. They do not preserve raw prompts, speech style, private reasoning, or full transcripts. Store personal or production data only when it is genuinely part of an approved product specification and minimize it.

## Credentials and external tools

Credentials belong in environment variables or operating-system credential stores. DB visualization, task providers, Git hosting, and publication tools are optional and are connected only after detection, trade-off review, and user selection. A raw provider payload is neither committed nor copied into diagnostics. The connector supplies a minimal normalized local observation under ignored state; it includes a source hash for identity but not comments, descriptions, tokens, or unrelated profile data. External content is quarantined and provenance is recorded before promotion.

## Diagnostics and evidence

Use compact fingerprints, stable error codes, commands, result summaries, and links to controlled CI evidence. Do not store raw logs, tokens, home paths, raw provider payloads, or user conversations. Provider observations are not committed and a clone intentionally treats external state as unknown until it is refreshed. Review any diagnostic bundle before sharing because project identifiers and repository names can still be sensitive.

## Removal and retention

Removing the Plugin or CLI never deletes repository-owned specs, contracts, DBML, or Git history. Teams define retention for local quarantine, expired work reservations, generated candidates, and external provider observations according to their own policy. Safe cleanup must show the exact paths and never run as a hidden context-recovery action.
