# Multi-repository dogfood report

This report compares deterministic checks, not team productivity. No elapsed-time or human-performance claim is made from this local fixture.

- Harness observations: **9/9** scenarios detected
- Manual Git + static docs baseline: **2/9** scenarios have a deterministic native check
- Dogfood assertions: **23/23** passed
- TDD proof: **2 expected failing test runs** were observed before their implementations passed
- Result: **PASS**

| Scenario | Harness | Manual Git + static docs | Boundary |
|---|---:|---:|---|
| `clean-clone-next-action` | detected | not-deterministic | Git restores commits but does not combine product meaning, live work, and one safe next action. |
| `answered-product-context-recovery` | detected | not-deterministic | Static notes can be read manually but provide no fingerprinted stale or unknown audit. |
| `false-live-claim-prevention` | detected | not-deterministic | Branches alone do not provide an atomic service-scope ownership reservation. |
| `external-task-semantic-reservation` | detected | not-deterministic | An issue assignee and branch do not atomically reserve contract, policy, database, UI, dependency, and pointer meaning or reject stale connector reads. |
| `path-disjoint-semantic-conflict` | detected | not-deterministic | Native merge checks do not compare business policies, contracts, UI flows, or DB meaning across different paths. |
| `wrong-workspace-evidence` | detected | not-deterministic | Git does not know which service workspace is allowed to satisfy a work definition. |
| `submodule-pointer-drift` | detected | deterministic-check | Native Git can expose a changed gitlink, although it does not explain the service integration impact. |
| `local-only-recoverability` | detected | deterministic-check | A careful remote-ref containment check can determine whether a commit is published. |
| `exact-release-candidate` | detected | not-deterministic | Git and prose alone do not bind technical evidence and user validation to one candidate digest. |

The manual column does not mean a careful engineer cannot discover the problem. It records whether ordinary Git plus static documentation provides a deterministic, service-aware check without this harness.

The dogfood fixture uses local bare remotes and public-looking placeholder URLs. It proves repository behavior without claiming hosted GitHub or Jira writes, network performance, or production load capacity.
