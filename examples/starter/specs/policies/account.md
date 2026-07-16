---
schema_version: 1
id: policy.account.recovery.rate-limit
kind: policy
status: approved
revision: 1
refs: [scenario.account.recovery.rate-limited]
---

An account may request recovery six times per rolling hour. The seventh request fails without revealing whether the account exists.
