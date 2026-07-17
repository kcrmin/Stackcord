# Privacy

## Stored project knowledge

Discovery checkpoints contain normalized summaries, decisions, policies, scenarios, quality requirements, assumptions, and open questions. They do not preserve raw prompts, speech style, private reasoning, or full transcripts. Store personal or production data only when it is genuinely part of an approved product specification and minimize it.

## Credentials and external tools

Credentials belong in environment variables or operating-system credential stores. DB visualization, task providers, Git hosting, and publication tools are optional and are connected only after detection, trade-off review, and user selection. External content is quarantined and provenance is recorded before promotion.

## Diagnostics and evidence

Use compact fingerprints, stable error codes, commands, result summaries, and links to controlled CI evidence. Do not store raw logs, tokens, home paths, provider payloads, or user conversations. Review any diagnostic bundle before sharing because project identifiers and repository names can still be sensitive.

## Removal and retention

Removing the Plugin or CLI never deletes repository-owned specs, contracts, DBML, or Git history. Teams define retention for local quarantine, stale claims, generated candidates, and external provider data according to their own policy. Safe cleanup must show the exact paths and never run as a hidden context-recovery action.
