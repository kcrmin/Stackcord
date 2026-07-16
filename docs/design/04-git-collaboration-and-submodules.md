# Git, 협업, submodule, 충돌 정책

> 상태: 확정
>
> 마지막 갱신: 2026-07-16

## 1. 기본 전략

기본은 protected `main`을 중심으로 한 short-lived branch 방식이다. 매 release마다 `develop`을 거치는 전통적 Git Flow는 기본값으로 쓰지 않는다. 장기 stabilization, 여러 release line 동시 유지, 규제상 분리된 승인 기간이 실제로 필요할 때만 `release/<major.minor>` 유지 branch를 추가한다.

- `main`은 언제나 integration 가능한 상태를 유지한다.
- 직접 push를 금지하고 required checks와 review를 통과한 PR만 merge한다.
- 작업 branch는 작고 짧게 유지하되, 제품의 의미 단위와 end-to-end 검증 가능성을 잃을 정도로 쪼개지 않는다.
- release는 immutable tag와 artifact로 고정한다.
- Git은 solo 발견 작업에는 선택일 수 있지만 둘 이상의 사람이 협업하거나 release할 때는 사실상 필수로 권장한다.

이 선택은 waterfall을 만들지 않으면서도 contract, DB, UI 기준선을 먼저 안정화하고 그 위에서 여러 vertical slice를 병렬 개발하게 한다.

## 2. Branch와 commit convention

### Branch 이름

```text
<type>/<short-description>
<type>/<work-id>-<short-description>   # 외부 work ID가 있을 때
```

허용 type:

```text
feature fix refactor perf test docs build ci chore revert
```

예시:

```text
feature/GH-142-account-recovery
fix/session-expiry-loop
docs/recovery-runbook
```

- 영어 lowercase와 hyphen을 사용한다.
- branch, commit, PR title에 `ai`, `agent`, 도구 이름 같은 생성 주체 표식을 넣지 않는다.
- ticket slug를 별도 필수 개념으로 만들지 않는다. `short-description`은 읽기 위한 설명일 뿐 stable product ID가 아니다.

### Commit과 PR 제목

Conventional Commits 형식을 사용한다.

```text
feat(identity): add account recovery challenge
fix(web): prevent repeated expired-token submission
docs(runbook): document recovery incident checks
```

- 한 commit은 하나의 검토 가능한 의도를 가진다.
- test는 구현과 같은 commit에 포함해 언제든 checkout 가능한 상태를 만든다. red/green 단계를 억지로 별도 commit으로 남기지 않는다.
- formatter나 generated file 때문에 의미 diff가 묻히지 않게 분리한다.
- shared branch를 rebase하거나 force-push하지 않는다. 개인 branch도 review가 시작된 뒤에는 history rewrite를 피한다.

## 3. Pull Request 정책

협업 변경은 초기에 Draft PR을 열어 충돌과 의도 중복을 발견한다. merge 준비가 되면 다음 항목을 충족한다.

```text
Why
Scope / out of scope
Related specs, policies, scenarios, contracts
Affected workspaces and change bundle
Risk and rollout
TDD evidence and verification commands
UI evidence / accessibility impact, when applicable
DB migration and rollback, when applicable
Security and privacy impact
Linked work item and dependency
```

기본 merge는 squash다. 의미 있는 여러 commit history가 운영·감사에 필요할 때만 merge commit을 허용한다. repository 설정과 branch protection이 이를 강제한다.

Required checks의 최소 범위:

- schema, context, contract, DBML validation
- 해당 workspace test·lint·build·security checks
- changed-path 기반 영향 workspace와 consumer compatibility test
- generated artifact freshness
- conflict claim과 migration ordering 검사
- release 대상이면 install, upgrade, rollback, provenance 검사

CODEOWNERS 또는 host의 동등 기능으로 policy, contract, migration, security, release 경계의 review owner를 지정한다. 단순 path owner만으로 제품 의미 충돌이 해결되지는 않으므로 stable ID 영향 관계도 PR에 표시한다.

## 4. Workspace와 submodule 전략

### 언제 submodule을 쓰는가

다음이 둘 이상이면 별도 repository + submodule을 적극 권장한다.

- 독립적인 release 또는 access control이 필요하다.
- 별도 team ownership과 CI가 있다.
- 기술 stack·dependency·배포 lifecycle이 뚜렷하게 다르다.
- root orchestration과 분리된 history가 가치 있다.
- 다른 제품에서도 독립 소비할 수 있다.

단지 폴더 충돌을 피하려는 이유만으로 repository를 나누지 않는다. 서로 자주 원자적으로 바뀌어야 하고 같은 team이 같은 release로 운영한다면 directory workspace가 더 낫다.

기존 프로젝트는 topology를 보존한다. 자동으로 monorepo를 submodule로 바꾸지 않고, 이점·이관 비용·history·CI·권한을 비교한 change proposal과 사용자 승인이 있을 때만 바꾼다.

### Pointer 규칙

- root는 각 submodule의 정확한 commit SHA를 pin한다.
- 평상시 `git submodule update --remote`로 최신을 따라가지 않는다.
- consumer clone은 root pointer 그대로 checkout한다. detached HEAD는 소비자에게 정상이다.
- contributor가 submodule을 수정하려면 먼저 그 repository에서 branch를 만들고 tracking 상태를 확인한다.
- detached HEAD에 local commit이나 변경이 있으면 자동 sync를 중단하고 복구 선택지를 제시한다.
- root pointer는 workspace 변경이 integration 준비가 되었을 때 한 번에 묶어 갱신한다. 각 작은 commit마다 pointer를 바꾸지 않는다.
- nested submodule은 기본 금지다. 필요하면 architecture decision으로 승인한다.

## 5. Cross-repository change bundle

root와 여러 workspace가 함께 바뀌는 하나의 제품 변경은 `.harness/work/changes/<change-id>.yaml`의 change bundle로 묶는다.

```text
change
├── affected spec / policy / scenario
├── contract compatibility plan
├── workspace A branch / PR / target commit
├── workspace B branch / PR / target commit
├── root integration branch / PR
├── merge order
└── verification and rollback
```

Git은 여러 repository를 원자적으로 merge하지 못한다. 따라서 “동시에 merge하면 된다”는 계획을 만들지 않는다.

### 호환 가능한 변경

1. contract에 backward-compatible addition을 먼저 추가한다.
2. provider와 consumer를 어느 순서로 배포해도 동작하도록 구현한다.
3. 각 workspace PR을 merge하고 검증된 commit을 고정한다.
4. root integration PR에서 submodule pointer와 baseline을 함께 갱신한다.

### Breaking change

1. old/new contract version을 잠시 함께 제공한다.
2. provider를 dual-compatible 상태로 배포한다.
3. consumer를 새 version으로 이동한다.
4. root RC가 모두 새 version을 사용함을 확인한다.
5. 별도 후속 change에서 old version을 제거한다.

실제로 version 병행이 불가능하면 maintenance window, deploy order, rollback point, 사용자 영향이 명시된 release plan과 별도 승인을 요구한다.

## 6. Worktree와 clone

- 같은 repository의 독립 branch를 병렬 작업할 때는 `git worktree`를 우선 사용한다.
- root와 여러 submodule을 함께 바꾸는 큰 change를 독립시킬 때는 전체 orchestration repository를 별도 clone하는 것이 더 안전하다.
- 같은 root checkout의 한 submodule directory를 두 작업자가 서로 다른 branch로 공유하지 않는다.
- worktree 또는 clone 생성 위치는 repository 밖의 도구 전용 directory로 두고 `.gitignore`에 의존하지 않는다.
- 종료 전 dirty 상태, unpushed commit, active claim을 검사한다. 자동 삭제하지 않는다.

Worktree는 파일 충돌을 줄이지만 제품 정책·contract 의미 충돌은 해결하지 못한다. 그 문제는 claim, impact graph, review로 해결한다.

## 7. 작업 시작 전 conflict preflight

AI가 `start work`를 실행하면 다음 범위를 선언하고 기존 active work와 비교한다.

- repository, workspace, path glob, module
- spec, policy, scenario stable ID
- contract와 DBML entity/migration range
- UI route, flow, component token
- dependency·toolchain·CI configuration
- 예상 root pointer와 merge order

위험 등급:

| 등급 | 의미 | 행동 |
|---|---|---|
| clear | 알려진 겹침 없음 | claim 후 진행 |
| coordinate | 같은 영역이나 호환 가능한 순서 존재 | 담당 범위·merge order를 합의 후 진행 |
| block | 같은 semantic owner, 같은 migration slot, incompatible contract 등 | 설계를 조정하기 전 구현 금지 |
| unknown | remote/provider를 확인하지 못함 | 사용자에게 불확실성을 알리고 보수적으로 범위 축소 |

claim은 잠금이 아니라 의도를 보이는 lease다. owner, scope, baseline, branch, expiry, contact를 가진다. 만료 claim을 자동 삭제하지 않고 owner/remote 상태를 확인해 release한다.

외부 task provider가 없을 때 claim은 feature branch의 첫 작은 commit에 branch record와 함께 저장하고 즉시 push하거나 Draft PR로 공개하는 것을 기본으로 한다. conflict check는 fetch된 remote refs에서 이 파일들을 checkout 없이 읽는다. 완료 전에는 `main`에 active claim을 합치지 않으며, PR 준비 시 compact evidence로 전환한다. push 권한이 없거나 offline이면 다른 작업자에게 보인다고 보장할 수 없으므로 상태를 `unknown`으로 표시하고 multi-user 작업에는 외부 provider 또는 remote branch 공개를 권장한다.

## 8. 대표 충돌 시나리오와 해결

| 시나리오 | 사전 감지 | 기본 해결 |
|---|---|---|
| 둘이 같은 파일의 다른 기능 수정 | path/module claim | 파일 경계를 나누거나 merge order 지정; 필요하면 skeleton을 먼저 merge |
| 파일은 다르나 같은 정책을 다르게 해석 | policy/scenario ID overlap | 정책 원본과 scenario를 먼저 조정하고 양쪽 test를 갱신 |
| backend 둘이 같은 contract 수정 | contract ID와 compatibility diff | contract owner가 additive version을 먼저 merge; consumer 순서 고정 |
| frontend가 old contract에 연결 | generated client fingerprint | mock/client를 canonical contract에서 재생성하고 compatibility test |
| 두 DB migration이 같은 sequence 사용 | migration namespace/parent | rebase가 아니라 새 sequence 재할당과 clean database replay |
| 같은 entity를 서로 다른 뜻으로 수정 | DBML semantic diff + policy refs | 모델 의미 합의 후 하나의 migration plan으로 통합 |
| 한 작업이 dependency major를 올림 | manifest/lockfile/toolchain claim | 별도 enabling PR을 먼저 merge하고 모든 workspace 검증 |
| root pointer만 먼저 변경 | child commit/CI 존재 검사 | workspace merge·검증 후 root integration PR에서만 pointer 변경 |
| submodule remote는 갱신됐으나 clone은 old pointer | actual pointer와 remote 비교 | root가 pin한 상태는 오류가 아님; change bundle이 요구할 때만 갱신 |
| local dirty 상태에서 pull/sync | status preflight | 자동 stash/reset 금지; 변경 보존·commit·별도 worktree 선택지 제시 |
| task provider에는 완료, PR은 미merge | live status와 Git mismatch | operational truth를 기준으로 provider status를 조정 |
| 외부 UI mockup과 현재 canonical UI 충돌 | source authority와 coverage diff | reference면 참고만, canonical이면 change approval 후 baseline 교체 |

`.gitignore`는 충돌 회피 수단이 아니다. local cache와 secret만 제외하고, 협업해야 하는 상태를 숨기지 않는다.

## 9. Backend 경계와 전체 UI 이후 개발

제품 전체 정의와 실행 가능한 UI baseline 뒤에 모든 backend를 한꺼번에 만들지 않는다.

1. vertical slice에 필요한 policy, scenario, interface/contract, failure behavior를 먼저 확정한다.
2. compile 가능한 boundary skeleton과 test fixture를 작은 enabling PR로 merge할 수 있다.
3. 독립 구현이 필요한 backend가 여러 개면 각각 contract test를 기준으로 병렬 구현한다.
4. 두 backend가 모두 호환된 뒤 frontend 연결 PR을 merge하거나, 안정된 mock/generated client로 frontend를 먼저 구현한다.
5. slice를 end-to-end 검증하고 다음 slice로 간다.

전체 UI baseline은 한 번에 거대한 merge를 하지 않는다. 역할·domain·journey 단위의 작은 PR로 canonical baseline에 합치고 coverage map이 전체 범위를 추적한다.

## 10. Fetch, pull, push 자동화 경계

- `fetch`와 status 조회는 read-only 진단으로 자동 수행할 수 있다.
- fast-forward update도 작업 tree 변경이므로 실행 전 무엇이 바뀌는지 알린다.
- hidden pull, automatic rebase, automatic stash, reset, force push를 금지한다.
- push와 PR 생성은 현재 사용자의 명시적 요청 또는 범위가 명확한 standing consent가 있을 때만 수행한다.
- 다른 사람의 branch를 임의 수정하지 않는다.
- root와 workspace의 remote/branch가 예상과 다르면 push를 중단한다.

## 11. Hotfix와 release line

- production hotfix는 해당 release tag 또는 유지 중인 maintenance branch에서 분기한다.
- 가장 작은 회귀 test와 수정으로 release하고 `main`에 반드시 forward-port한다.
- 여러 release line을 유지할 때만 `release/<major.minor>`를 둔다.
- release tag, checksum, SBOM, provenance는 immutable하게 보존한다.
- rollback은 pointer, artifact, migration compatibility를 함께 검사한다. Git revert만으로 database가 자동 복구된다고 가정하지 않는다.

## 12. Git을 사용하지 않을 때

발견과 local prototype은 가능하지만 다음 기능이 제한된다.

- 협업 claim의 remote 일치 확인
- 변경 history와 review
- cross-repo pointer 고정
- RC의 정확한 source commit 증명
- signed release provenance

따라서 release 단계 진입 전에는 Git repository와 immutable source revision이 반드시 필요하다.

## 13. 수용 기준

- 두 사람이 같은 orchestration clone에서 각자 별도 worktree/clone과 scope claim으로 작업할 수 있다.
- submodule pointer가 의도치 않게 최신 remote를 따라가지 않는다.
- contract breaking change가 merge 순서 없이 동시에 배포되지 않는다.
- local dirty change를 자동 stash/reset하지 않는다.
- branch와 commit에 AI 흔적을 강제하지 않고 일반 Git convention만 사용한다.
- `main` 하나로 평상시 개발이 가능하며 정말 필요한 경우에만 release branch가 추가된다.
