---
schema_version: 1
id: scenario.account.recovery.rate-limited
kind: scenario
status: approved
revision: 1
refs: [policy.account.recovery.rate-limit]
---

After six attempts in a rolling hour, the next request fails without revealing account existence and tells an eligible client when retry becomes safe.
