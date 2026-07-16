# 실제 사용자 경험과 생성 결과 walkthrough

> 상태: 확정
>
> 마지막 갱신: 2026-07-16
>
> 이 문서는 사용자가 명령을 외우지 않고 AI와 대화할 때 화면에 보이는 흐름, 내부 검사, 생성·수정되는 파일, Git/작업관리 결과를 한 번에 보여준다.

## 1. 새 서비스 시작

사용자:

> 새 서비스 시작하자.

AI는 현재 directory, 기존 Git과 하네스를 read-only로 확인한다. 빈 공간이고 아직 이름이 없으면 repository를 만들지 않고 `.harness-drafts/<draft-id>/`를 만든다.

AI:

> 발견 내용을 임시 draft에 계속 정리하겠습니다. 먼저 이 서비스가 가장 먼저 해결해야 할 대상은 누구인가요?
>
> A. 개인 사용자 — 한 사람이 직접 문제를 해결합니다. **권장: 현재 설명과 가장 가깝습니다.**
> B. 팀 구성원 — 여러 사람이 같은 업무를 함께 처리합니다.
> C. 운영 담당자 — 내부 운영 효율이 핵심입니다.
> D. 기업 고객 — 조직 단위 구매·관리·감사가 핵심입니다.
> 기타: 직접 설명

사용자:

> A

AI는 원문 “A”를 보존하지 않고 다음처럼 정규화한다.

```text
.harness-drafts/01J.../
├── manifest.yaml
├── state.yaml
└── specs/product/
    ├── summary.md             # primary audience: individual user
    ├── decisions.yaml         # decision.product.primary-audience
    └── open-questions.yaml    # 다음에 필요한 결정과 이유
```

AI:

> 개인 사용자가 직접 해결하는 서비스로 확정했습니다. 이 사용자가 서비스를 사용한 뒤 얻어야 할 가장 중요한 결과는 무엇인가요?
>
> A. 시간이 걸리던 일을 즉시 끝냄 — 반복 작업 단축이 핵심입니다.
> B. 어려운 결정을 더 정확히 내림 — 비교·추천·검증이 핵심입니다.
> C. 상태를 계속 추적하고 놓치지 않음 — 알림·history·복구가 핵심입니다.
> D. 다른 사람과 안전하게 교환함 — 권한·승인·협업이 핵심입니다.
> 기타: 직접 설명

질문 수를 고정하지 않는다. 작은 서비스는 수십 개, 복잡한 서비스는 수백 개가 될 수 있다. 한 번에 하나만 묻지만 답할 때마다 다음을 갱신한다.

- 확정 decision
- 검증이 필요한 hypothesis
- 아직 답하지 않은 open question
- 앞 답변과의 contradiction
- 해당 답이 막거나 여는 lifecycle gate
- roles, journeys, policies, scenarios, quality coverage

AI는 20~30개 답변마다 원문을 반복하지 않고 짧게 정리한다.

```text
현재 확정: 역할 3, 핵심 journey 5, policy 18
가설: 4
모순: 1 — guest 사용과 모든 기능의 login 필수 여부
다음 결정: guest가 결과를 저장할 수 있는지
```

context가 압축되어도 다음 AI는 draft의 `state.yaml`, normalized summary, decision과 open question부터 읽어 정확히 다음 질문으로 돌아온다.

## 2. Root와 Git 생성

핵심 사용자·문제·가치·주요 journey가 안정되고 **이 서비스 repository 이름**이 승인되면 AI가 구조를 설명한다. 이 제품 자체의 public brand 이름과 사용자가 만드는 서비스 이름은 별개다.

AI:

> 제품 중심축이 안정되었습니다. `<service-name>` root를 만들고 Git을 초기화하는 것을 권장합니다. 아직 독립 배포 경계가 확인되지 않아 workspace는 root 하나로 시작하고, 필요성이 확정되는 즉시 별도 repository와 submodule을 추가하겠습니다.
>
> A. Root와 Git 생성 — 협업·복구·release를 모두 사용할 수 있습니다. **권장**
> B. Root만 생성 — 발견과 local 문서는 가능하지만 협업 기능은 제한됩니다.
> 기타: 저장 위치나 Git 정책 직접 지정

사용자:

> A

AI가 보여주는 plan:

```text
Create: <service-name>/
Create: root Git repository on protected-main policy
Migrate: draft → specs/ and .harness/
Generate: project-local Agent Skill and neutral structure
Remote write: none
Approval class: C, already authorized by this answer
```

생성 결과:

```text
<service-name>/
├── AGENTS.md
├── .agents/skills/use-project-harness/
├── .harness/
├── specs/
├── contracts/
└── docs/
```

framework source directory는 이때 만들지 않는다.

## 3. 제품 전체 정의와 기술 stack

제품 정의 단계에서 AI는 각 role과 journey에 대해 다음 누락을 찾는다.

- normal, empty, error, timeout, retry, cancel, recovery
- authentication, permission denied, administrator override
- data collection, retention, deletion, export, consent, audit
- notification failure, duplicate prevention, opt-out
- privacy, security, accessibility, internationalization
- incident, support, backup/restore, observability

사용자 답은 다음 정식 원본으로 이동한다.

```text
specs/product/roles.yaml
specs/product/capabilities.yaml
specs/product/journeys/*.md
specs/policies/**/*.md
specs/scenarios/**/*.md
specs/quality/*
```

전체 범위를 승인한 뒤 AI는 기술 취향부터 묻지 않는다.

AI:

> `journey.customer.live-collaboration`은 두 사용자가 같은 상태를 1초 이내에 보아야 하고, 잠시 offline이어도 작성 내용을 잃지 않아야 합니다. 이 요구가 architecture를 크게 바꿉니다.
>
> A. Offline queue + realtime sync — 구현은 복잡하지만 두 요구를 모두 충족합니다. **권장**
> B. Realtime only — offline 편집은 명시적으로 지원하지 않습니다.
> C. Periodic refresh — 가장 단순하지만 1초 목표를 충족하지 못하므로 제품 목표를 바꿔야 합니다.
> 기타: 다른 운영 제약 설명

답을 받은 뒤 공식 문서, 현재 security/release/maintenance 상태, 비용, team skill을 다시 확인해 2~3개 stack 대안과 권장안을 제시한다. 선택은 `specs/architecture/stack.yaml`과 ADR에 기능 근거·대안·재검토 조건까지 기록한다.

별도 배포·권한·소유권이 필요한 경계가 확정되면 그 즉시 workspace repository와 submodule을 제안한다.

```text
root/
├── apps/web/                  # submodule workspace
├── services/identity/         # submodule workspace
└── packages/policy-engine/    # 같은 root의 directory workspace일 수도 있음
```

AI는 “frontend/backend라서 무조건 분리”하지 않고 독립 lifecycle이 실제로 있을 때만 나눈다.

## 4. 전체 UI와 외부 mockup

사용자:

> 외부에서 만든 이 mockup으로 UI를 시작해줘.

AI:

> source를 격리 검사했습니다. 기존 제품 흐름 전체를 덮어쓰는 완성 기준인지, 시작 자료인지가 결과를 바꿉니다.
>
> A. Seed — 가져온 뒤 제품 spec에 맞춰 고치고 code가 새 기준이 됩니다. **권장**
> B. Canonical — 이 exact version과의 차이를 승인 대상으로 관리합니다.
> C. Reference — 참고만 하고 다른 UI를 만들 수 있습니다.
> 기타: 화면별로 다른 권위 지정

source는 `specs/ui/sources/<source-id>.yaml`에 license, version/hash, authority, 관련 journey를 기록한다. executable/script/archive는 `.harness/local/imports/`에서 검사한다.

전체 UI는 하나의 거대한 branch가 아니라 다음처럼 나눈다.

```text
UI baseline
├── customer onboarding journey PR
├── customer recovery journey PR
├── administrator access journey PR
└── operations incident journey PR
```

각 PR이 normal/loading/empty/error/permission/responsive/accessibility state를 채우고 `specs/ui/coverage.yaml`이 전체 P30 범위를 추적한다. mock data로 모든 journey를 실행한 뒤 하나의 baseline을 승인한다.

## 5. Contract, DBML, dbdiagram

UI에서 실제 data/command 요구가 확인되면 AI가 contract와 database 질문을 시작한다.

AI:

> 복구 요청을 같은 사용자가 반복 전송할 수 있습니다. 중복 처리 정책을 먼저 정해야 API와 DB 제약이 안정됩니다.
>
> A. 같은 idempotency key는 같은 결과 — 중복 side effect를 막고 재시도에 안전합니다. **권장**
> B. 일정 시간 안의 같은 대상 요청을 하나로 합침 — client가 key를 관리하지 않아도 되지만 오판 가능성이 있습니다.
> C. 매 요청을 새 처리 — 단순하지만 중복 알림·비용 위험을 수용해야 합니다.
> 기타: 별도 business rule

결과:

```text
specs/policies/failures/recovery-request.md
specs/scenarios/identity/repeated-recovery-request.md
contracts/services/identity-recovery.yaml
contracts/api/identity.*
contracts/data/modules/identity.dbml
contracts/errors.yaml
```

AI가 DBML validation을 통과시키고 dbdiagram CLI로 diagram project에 push하면 사용자는 관계를 시각적으로 확인한다.

사용자가 dbdiagram 웹에서 column/relation을 고친 뒤 말한다.

> 다이어그램 수정했어. 반영해줘.

AI는 remote를 scratch로 pull하고 이렇게 묻는다.

> canonical DBML과 비교하면 `recovery_attempt.user_id`가 nullable로 바뀌었습니다. guest recovery를 허용하려는 변경인지, 임시 diagram 수정인지에 따라 보안·retention 정책이 달라집니다. 왜 변경했나요?

의도를 확인한 뒤에만 policy → scenario → DBML → migration → consumer 영향 순서로 반영하고 다시 push한다.

## 6. 구현 경계와 작업 생성

contract 승인 뒤 공유되는 경계만 먼저 만든다.

- generated client/server types
- handler/gateway extension point
- authentication context와 error mapping
- repository/external adapter port
- module dependency rule
- test fixture와 mock server

모든 class/method interface를 미리 만들지는 않는다.

GitHub를 task provider로 선택했다면 다음 Issue가 생성될 수 있다.

```text
Title: feat(identity): implement account recovery

Outcome
- A locked-out customer can regain access safely.

Acceptance
- scenario.identity.recovery.success
- scenario.identity.recovery.expired-token
- scenario.identity.recovery.rate-limited

Contracts
- contract.identity.recovery.v1

Workspaces
- workspace.identity (provider)
- workspace.web (consumer)

Dependencies
- contract baseline P50
- identity boundary skeleton P55

Out of scope
- administrator manual override
```

Issue는 제품 의미 원본이 아니다. acceptance와 상태를 실행하는 작업 단위이며 spec/contract stable ID를 가리킨다.

## 7. 작업 시작, branch, TDD, PR

사용자:

> 이 작업 시작해.

AI가 하는 일:

1. root/workspace remote fetch와 actual-state refresh
2. active Issue/claim/branch/PR 확인
3. path뿐 아니라 policy, scenario, contract, DB entity, UI flow, dependency overlap 계산
4. branch/worktree와 merge order plan
5. claim 공개
6. 실패 test부터 작성

결과:

```text
Branch: feature/GH-142-account-recovery
Worktree: repository 밖의 격리 경로
Claim: recovery handler, policy/contract refs, identity tables, web recovery flow
Draft PR: 즉시 열어 범위와 충돌을 공개
```

Commit 예시:

```text
feat(identity): handle recovery challenge expiry
```

test와 최소 구현을 같은 검토 가능한 commit에 넣는다. red/green을 보이기 위해 깨진 commit을 shared history에 강제하지 않고 TDD receipt가 실패/통과 command fingerprint를 보존한다.

PR 예시:

```text
Title: feat(identity): add account recovery

Why
Locked-out customers need a policy-compliant recovery path.

Scope
Identity provider behavior and web recovery connection.

Specs / Contracts
policy.identity.recovery
scenario.identity.recovery.*
contract.identity.recovery.v1

Risk / Rollout
Rate limiting, token expiry, notification failure.

Evidence
Unit, contract, integration, accessibility, migration rollback.
```

## 8. 병렬 작업 충돌 예시

두 사람이 서로 다른 backend를 구현하고 frontend가 둘을 연결하는 경우:

```text
contract additive change merge
├── workspace.identity provider PR
├── workspace.notification provider PR
└── workspace.web generated mock/client work
```

두 provider가 contract test를 통과하고 merge된 뒤 web 실제 연결과 root pointer integration PR을 합친다. frontend를 먼저 만들 필요가 있으면 canonical contract에서 만든 mock/client를 사용한다.

누군가 같은 contract를 다른 의미로 바꾸려 하면 구현 전에 다음을 표시한다.

```text
Conflict: block
Existing: GH-142 changes contract.identity.recovery.v1 retry semantics
Candidate: GH-155 changes the same timeout and compensation behavior
Recommended: contract owner가 하나의 additive v2 proposal로 합치고 provider/consumer merge order를 확정
```

## 9. 다른 사람이 clone하고 이어서 작업

새 작업자:

> 이 프로젝트 이어서 해. 지금 뭐 해야 해?

repo-local Skill이 Plugin 유무와 무관하게 root를 찾는다. AI는 다음을 직접 확인한다.

- root와 workspace remote/pointer
- branch/upstream/ahead/behind/diverged/dirty/detached
- current baseline과 stale graph
- active Issue/claim/PR
- contract provider/consumer version
- 다음 dependency gate

예시 응답:

```text
현재 product·UI·contract baseline은 승인되었습니다.
workspace.identity는 root pointer와 일치하고 clean합니다.
workspace.web는 pointer commit보다 한 commit 앞서지만 push되지 않았습니다.
GH-142 claim과 겹치므로 그 변경을 보존하기 전에는 sync하지 않겠습니다.
다음 안전 행동: local web commit의 목적을 확인해 기존 branch에 push하거나 별도 branch로 보존.
```

사용자는 pull/submodule 명령을 직접 선택할 필요가 없다. AI는 안전한 경우 수행 내용을 알리고 실행하며, 손실 위험이 있으면 한 가지 결정을 묻는다.

## 10. AI가 내용을 잊은 것 같을 때

사용자:

> 너 지금 이 프로젝트 내용 잊은 것 같아. 다시 확인해.

`audit-project-context` Skill이 실행된다.

```text
1. mutation 중지
2. actual Git/workspace/source registry 재조회
3. spec/contract stable ID와 fingerprint 재검증
4. generated summary stale 확인
5. active work/claim과 현재 branch 연결
6. 사실·불일치·unknown·다음 행동 보고
7. 원래 작업 재개
```

선택 Hook은 session 시작이나 context 압축 뒤 이 audit 필요를 알려줄 뿐, 자동 pull이나 file 수정을 하지 않는다.

## 11. Handoff가 실제로 필요한 경우

평상시 각자 맡은 일을 계속하는 것은 handoff가 아니다. branch work record와 context refresh가 현재 상태를 유지한다.

담당자가 휴가·퇴사·역할 변경으로 실제 책임을 넘길 때만:

> GH-142를 민수에서 지연에게 넘겨줘.

AI는 old/new owner, remote branch, baseline, 완료 범위, failing/passing test, 미확정 결정, next action을 보여주고 새 담당자의 수락을 기록한다. local-only commit이 있으면 책임 이전 전에 push 또는 보존이 필요하다.

## 12. Integration, RC, release

workspace PR이 모두 merge되면 root integration PR이 다음을 한 번에 고정한다.

- merged workspace commit
- submodule pointer
- contract fingerprint
- generated client freshness
- clean clone build/test
- full-system journey

사용자:

> release 준비해.

AI는 security, install/update/uninstall, migration/rollback, performance, accessibility, docs, SBOM/signature/provenance까지 검사한다. blocker가 없을 때 exact root/workspace/plugin/CLI artifact를 RC로 고정한다.

사용자가 같은 RC로 새 프로젝트와 clone continuation journey를 직접 확인한다. source가 바뀌면 승인이 무효화되고 새 RC가 필요하다.

final publish 직전:

```text
Release 1.0.0
RC digest: sha256:...
User validation: same digest
Artifacts: macOS arm64/x86_64, Windows arm64/x86_64
Rollback: verified
External action: public GitHub release, Plugin marketplace, Homebrew, WinGet
```

이 exact target에 사용자가 최종 승인한 뒤에만 공개한다.

## 13. 기능이 제공되는 위치

| 사용자가 보는 기능 | 제공 계층 | 프로젝트에 남는 결과 |
|---|---|---|
| 질문과 서비스 발견 | Skill + CLI draft | normalized spec/decision/open question |
| “지금 뭐 해야 해?” | context Skill + CLI | actual-state report와 next action |
| project/root/workspace 생성 | CLI plan/apply | harness, repo-local Skill, Git/submodule |
| 충돌 경고 | CLI + provider adapter | claim, change bundle, evidence |
| 작업 관리 | 선택 provider + links | task ID와 spec/contract/PR 연결 |
| DB diagram 협의 | database Skill + CLI adapter | canonical DBML과 sync fingerprint |
| 외부 UI import | UI Skill + CLI quarantine | source authority/provenance/coverage |
| context 복구 | repo-local/Plugin Skill + CLI | index, impact, local current cache |
| PR·통합 | Git/host adapter + CLI gate | branch/PR/pointer/evidence |
| RC·release | CLI + release adapter | immutable candidate, artifact, support |

사용자는 주로 AI와 말하고, Skill은 올바른 흐름을 선택하며, CLI는 추측을 검사하고, repository는 다음 사람에게 실제 컨텍스트를 전달한다.
