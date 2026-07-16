# 생성되는 프로젝트 구조와 각 파일의 책임

> 상태: 확정
>
> 마지막 갱신: 2026-07-16
>
> 이 문서는 신규 프로젝트에 생성하거나 기존 프로젝트에 비파괴적으로 추가할 오케스트레이션 하네스, 제품 명세, 계약, 설명 문서, workspace 연결 파일의 기본 구조를 정의한다. 특정 framework, language, database, Git hosting provider를 전제로 하지 않는다.

## 1. 핵심 설계 결정

파일을 사람이 읽는지 AI가 읽는지에 따라 나누지 않는다. 실제 사용자는 대부분 자연어로 AI에게 질문하고 AI가 파일을 읽어 요약·수정할 가능성이 높다. 따라서 다음 네 가지 책임에 따라 분리한다.

| 영역 | 질문 | 책임 |
|---|---|---|
| `specs/` | 서비스는 무엇을 의미하고 어떻게 행동해야 하는가 | 제품 의도, 정책, 시나리오, 품질, UI, architecture의 규범적 명세 |
| `contracts/` | 구성 요소와 외부 시스템은 서로 무엇을 약속하는가 | API, event, data, authentication, error, service-level 상호작용 계약 |
| `.harness/` | 프로젝트를 지금 어떻게 조정하고 검증해야 하는가 | lifecycle, baseline, workspace, task 연결, 충돌 예약, 승인, 증거 |
| `docs/` | 사람과 AI가 이해·운영하기 위해 무엇을 설명받아야 하는가 | 안내서, runbook, 문제 해결, 생성된 현재 상태 요약 |

이 분리는 가시성보다 **의미와 변경 책임**을 기준으로 한다.

- 제품 정책을 `.harness/` 상태 파일에 숨기지 않는다.
- API schema에 모든 business rule을 억지로 넣지 않는다.
- 같은 사실을 Markdown, YAML, task provider에 중복 저장하지 않는다.
- AI가 설명할 때는 관련 `spec`, `contract`, `state`의 stable ID와 현재 fingerprint를 함께 제시한다.
- 사용자가 직접 파일을 수정해도 되지만 AI는 schema와 영향 범위를 검사하고 의미가 달라지면 확인한다.

## 2. 검토한 구조 대안

### 모든 파일을 `.harness/`에 두는 방식

한곳에 모이지만 제품 정책과 운영 상태가 섞이고, 다른 도구가 표준 계약과 문서를 찾기 어렵다. 채택하지 않는다.

### 모든 파일을 `docs/`에 두는 방식

읽기는 쉽지만 lifecycle, fingerprint, 계약 호환성, 충돌 예약 같은 기계 상태를 신뢰성 있게 검사하기 어렵다. 채택하지 않는다.

### 규범 명세·계약·제어 상태·설명 문서를 분리하는 방식

제품 의미와 기술 경계, 오케스트레이션 상태가 서로 덮어쓰지 않으며 각 영역을 독립적으로 검증할 수 있다. 이 방식을 기본으로 채택한다.

## 3. 신규 프로젝트 기본 구조

```text
project-root/
├── README.md
├── AGENTS.md
├── .editorconfig
├── .gitattributes
├── .gitignore
├── .gitmodules                         # submodule 사용 시에만
├── .agents/
│   └── skills/
│       └── use-project-harness/
│           ├── SKILL.md
│           └── references/
│               └── fallback.md
│
├── .harness/
│   ├── manifest.yaml
│   ├── entry.md
│   ├── sources.yaml
│   ├── workspaces.yaml
│   ├── state/
│   ├── policies/
│   ├── work/
│   ├── evidence/
│   ├── integrations/
│   ├── templates/
│   └── local/
│
├── specs/
│   ├── index.md
│   ├── product/
│   ├── policies/
│   ├── scenarios/
│   ├── quality/
│   ├── architecture/
│   └── ui/
│
├── contracts/
│   ├── registry.yaml
│   ├── services/
│   ├── api/
│   ├── events/
│   ├── data/
│   ├── schemas/
│   ├── auth/
│   └── errors.yaml
│
├── docs/
│   ├── index.md
│   ├── guides/
│   ├── runbooks/
│   ├── troubleshooting/
│   └── generated/
│
├── <workspace-path-a>/
├── <workspace-path-b>/
│
└── <provider-specific-files>/          # 선택한 provider에만
```

`frontend/`, `backend/`, `mobile/` 같은 이름을 고정하지 않는다. 실제 제품 책임과 architecture에 따라 workspace ID와 경로를 정하고 `.harness/workspaces.yaml`에서 연결한다.

## 4. 이름 확정 전 발견 작업 공간

프로젝트 이름과 root를 만들기 전에도 긴 발견 대화의 결과를 계속 저장한다.

```text
<selected-parent>/.harness-drafts/<draft-id>/
├── manifest.yaml
├── state.yaml
└── specs/
    └── product/
        ├── summary.md
        ├── decisions.yaml
        └── open-questions.yaml
```

| 파일 | 책임 |
|---|---|
| `manifest.yaml` | stable draft ID, 생성 시점, locale, schema version, 선택한 상위 경로를 기록한다. |
| `state.yaml` | 발견 진행 상태, 마지막 정상 저장, 모순과 다음 질문 위치를 기록한다. |
| `summary.md` | 원본 답변이 아니라 현재까지 정규화된 사용자·문제·가치·주요 흐름을 기록한다. |
| `decisions.yaml` | 확정된 결정과 근거, 승인 시점, 영향을 기록한다. |
| `open-questions.yaml` | 아직 결정되지 않은 항목, 질문 이유, 막고 있는 후속 단계를 기록한다. |

원본 대화, 사용자 말투, 불필요한 개인 정보는 source of truth로 저장하지 않는다. 이름이 승인되면 정식 root를 만들고 draft를 검증 가능한 방식으로 `specs/`와 `.harness/`에 이관한다. 새 위치의 fingerprint를 확인하기 전에는 draft를 삭제하지 않는다.

## 5. 루트 진입 파일

| 파일 | 책임 | 변경 규칙 |
|---|---|---|
| `README.md` | 프로젝트 목적, 시작 방법, workspace, 문서와 지원 경로를 사람에게 설명한다. | 기존 파일을 덮어쓰지 않고 필요한 section만 제안한다. |
| `AGENTS.md` | 모든 AI에게 `.harness/entry.md`와 현재 project refresh를 먼저 수행하도록 안내하는 짧은 지도다. | 상세 정책을 복제하지 않는다. 기존 파일에는 관리 구간만 diff로 추가한다. |
| `.editorconfig` | encoding, indentation, line ending의 cross-platform 기본값을 제공한다. | 기존 설정과 충돌하면 자동 덮어쓰지 않는다. |
| `.gitattributes` | text normalization, binary, generated artifact와 merge 속성을 정의한다. | macOS·Windows 검사를 통과한 항목만 적용한다. |
| `.gitignore` | secret, cache, raw log, 격리 import, local-only 상태를 제외한다. | 추적 파일의 충돌을 숨기는 용도로 사용하지 않는다. |
| `.gitmodules` | 별도 repository workspace의 URL과 path를 Git이 기록한다. | submodule 사용 시에만 Git 명령으로 관리한다. |

`.agents/skills/use-project-harness/SKILL.md`는 project에 남는 작은 repo-local Skill이다. Plugin이 설치되어 있으면 해당 workflow와 CLI로 연결하고, 없으면 `references/fallback.md`를 통해 `.harness/entry.md`, actual Git 상태, 관련 spec·contract를 읽는 복구 순서를 제공한다. 제품 정책을 Skill에 복사하지 않는다.

AI client별 진입 파일이 추가로 필요하면 `AGENTS.md` 내용을 복사하지 않고 `.harness/entry.md`를 가리키는 작은 adapter를 생성한다. Windows에서 동작이 달라지는 symlink는 사용하지 않는다.

## 6. `.harness/` — 오케스트레이션 제어 영역

```text
.harness/
├── manifest.yaml
├── entry.md
├── sources.yaml
├── workspaces.yaml
├── state/
│   ├── lifecycle.yaml
│   ├── baselines.yaml
│   ├── context-index.json
│   ├── impact-graph.json
│   ├── gates/
│   │   └── <stage-id>.yaml
│   └── release-candidate.yaml
├── policies/
│   ├── development.yaml
│   ├── tdd.yaml
│   ├── conflicts.yaml
│   ├── approvals.yaml
│   ├── security.yaml
│   └── release.yaml
├── work/
│   ├── provider.yaml
│   ├── links.yaml
│   ├── items/                       # 내장 provider일 때만
│   ├── claims/                      # 내장 provider일 때만
│   ├── changes/
│   └── branches/
├── evidence/
│   ├── receipts/
│   │   └── <work-id>/
│   │       └── tdd.yaml
│   └── gates/
├── integrations/
│   ├── git-host.yaml
│   ├── tasks.yaml
│   └── dbdiagram.yaml
├── templates/
│   ├── work-item.yaml
│   ├── scope-claim.yaml
│   ├── change-proposal.yaml
│   ├── handoff.yaml
│   └── adr.md
└── local/                           # Git에서 제외
    ├── state/
    │   └── current.json
    ├── cache/
    ├── logs/
    ├── imports/
    ├── dbdiagram/
    └── operations/
```

### 핵심 파일 책임

| 파일 | 책임 |
|---|---|
| `manifest.yaml` | project ID, harness schema version, locale, 기본 경로, 생성·migration version을 기록한다. |
| `entry.md` | AI가 읽을 최소 순서, project refresh, 금지된 파괴적 행동, fallback 방법을 정의한다. |
| `sources.yaml` | local/remote Git, task provider, dbdiagram, UI source 등 실제 상태 원본과 권한·freshness 정책을 등록한다. credential은 저장하지 않는다. |
| `workspaces.yaml` | workspace ID, kind, path, repository, remote, 책임, dependency 관계를 연결한다. |
| `state/lifecycle.yaml` | 각 생명주기 단계의 상태와 현재 진행 지점을 기록한다. |
| `state/baselines.yaml` | product, architecture, UI, contract, implementation-boundary fingerprint를 기록한다. |
| `state/context-index.json` | stable ID와 path, revision, fingerprint, status, reference의 생성 index다. 직접 수정하지 않는다. |
| `state/impact-graph.json` | spec, policy, scenario, contract, workspace, test, work의 stale 전파 graph를 생성한다. 직접 수정하지 않는다. |
| `state/gates/<stage-id>.yaml` | 단계별 검사, warning, 승인, commit과 무효화 조건을 기록한다. |
| `state/release-candidate.yaml` | 검증 중인 root·workspace·contract commit과 artifact checksum을 고정한다. |
| `policies/development.yaml` | 작업 시작·수정·검토·통합의 기본 순서와 금지 행동을 기계적으로 정의한다. |
| `policies/tdd.yaml` | test-first 필수 범위, 허용 예외, 필요한 evidence, CI·release 차단 조건을 정의한다. |
| `policies/conflicts.yaml` | 충돌 범위, claim, 만료, 경고 등급, 가능한 조정 전략을 정의한다. |
| `policies/approvals.yaml` | AI 자동 실행, 사전 알림, 사용자 승인 행동을 분류한다. |
| `policies/security.yaml` | untrusted repository, external command, secret, 외부 import의 기본 안전 정책을 정의한다. |
| `policies/release.yaml` | RC와 release의 필수 증거, 사용자 검증과 변경 무효화를 정의한다. |
| `work/provider.yaml` | 현재 live task source of truth와 adapter를 지정한다. secret은 저장하지 않는다. |
| `work/links.yaml` | 외부 task ID와 관련 spec, contract, workspace, branch, PR의 연결만 기록한다. |
| `work/items/<work-id>.yaml` | 외부 task provider가 없을 때만 작업 의도, 상태, 담당, dependency를 저장한다. |
| `work/claims/<claim-id>.yaml` | 내장 provider에서 작업자가 건드릴 path, module, contract, UI flow와 baseline을 예약한다. |
| `work/changes/<change-id>.yaml` | 제품·architecture·contract·workspace 변경 제안과 stale 영향 범위를 기록한다. |
| `work/branches/<branch-key>.yaml` | 현재 branch의 work, baseline, workspace branch, claim, checkpoint만 기록한다. handoff가 아니라 평상시 context 복구 기록이다. |
| `evidence/receipts/<work-id>/tdd.yaml` | 실패하는 검사와 최종 통과 검사의 command, fingerprint, 결과를 작게 기록한다. |
| `evidence/gates/` | raw log 대신 CI URL, checksum, 요약 결과 같은 재현 가능한 compact receipt를 저장한다. |
| `integrations/*.yaml` | Git host, task provider, dbdiagram project ID와 동작 설정을 저장한다. credential은 저장하지 않는다. |
| `templates/` | Plugin이 없어도 다른 AI가 동일한 work, claim, change, handoff, ADR 형식을 만들 수 있게 한다. |
| `local/state/current.json` | 현재 checkout의 branch, workspace commit, dirty/pointer 차이, active work·claim, 다음 gate를 매 refresh에서 생성하는 local cache다. worktree마다 달라지므로 Git에서 제외한다. |
| `local/` | 다시 만들 수 있는 current/cache, raw log, 외부 UI·dbdiagram 격리 import와 operation journal만 저장하며 Git에서 제외한다. |

`.harness/`에는 서비스의 business rule이나 사용자 정책을 직접 쓰지 않는다. 해당 내용은 stable spec ID로 `specs/`에 두고 `.harness`는 현재 승인 상태와 fingerprint만 가리킨다.

내장 provider의 active item·claim·branch record는 feature branch의 첫 commit과 Draft PR로 다른 작업자에게 공개한다. CLI는 fetch된 remote refs에서 이를 checkout 없이 읽는다. merge 전에는 active record를 compact evidence로 바꿔 `main`에 만료된 작업 상태가 쌓이지 않게 한다.

## 7. `specs/` — 서비스 의미와 정책의 규범적 원본

```text
specs/
├── index.md
├── product/
│   ├── summary.md
│   ├── roles.yaml
│   ├── capabilities.yaml
│   ├── journeys/
│   └── glossary.md
├── policies/
│   ├── index.yaml
│   ├── business/
│   ├── access/
│   ├── data/
│   ├── failures/
│   └── notifications/
├── scenarios/
│   ├── index.yaml
│   └── <domain>/
├── quality/
│   ├── targets.yaml
│   ├── security.md
│   ├── privacy.md
│   └── accessibility.md
├── architecture/
│   ├── overview.md
│   ├── stack.yaml
│   ├── workspaces.md
│   └── decisions/
│       └── <adr-id>.md
└── ui/
    ├── sources/
    │   └── <source-id>.yaml
    ├── coverage.yaml
    ├── flows/
    ├── states.yaml
    ├── decisions/
    └── reference/                  # 필요한 snapshot만
```

### 제품과 정책 파일 책임

| 파일 | 책임 |
|---|---|
| `index.md` | spec 영역의 읽기 순서와 ID 규칙을 설명한다. |
| `product/summary.md` | 사용자, 문제, 가치, 범위, 성공 기준을 중립적으로 요약한다. |
| `product/roles.yaml` | 사용자·관리자·운영자·system actor와 권한 책임을 구조화한다. |
| `product/capabilities.yaml` | 서비스가 제공해야 할 전체 capability, 상태, dependency와 release 범위를 기록한다. |
| `product/journeys/` | 각 역할의 시작, 성공, 취소, 예외, 복구까지의 end-to-end 흐름을 기록한다. |
| `product/glossary.md` | domain 용어와 금지된 모호한 표현을 정의한다. |
| `policies/index.yaml` | 모든 정책 ID, 상태, owner, 적용 범위, 관련 scenario·contract를 색인한다. |
| `policies/business/` | 가격, 자격, 승인, 취소, 환불, 상태 전이 등 business rule을 기록한다. |
| `policies/access/` | 인증, 권한, 역할, 관리자 override와 거부 동작을 기록한다. |
| `policies/data/` | 수집, 보존, 삭제, export, 동의, 감사 정책을 기록한다. |
| `policies/failures/` | timeout, partial failure, retry, compensation, 사용자 안내, 운영 escalation 정책을 기록한다. |
| `policies/notifications/` | 발송 조건, 채널, 중복 방지, 실패와 opt-out 정책을 기록한다. |
| `scenarios/` | Given·When·Then 또는 동등한 형식으로 정상·실패·권한·복구 acceptance를 기록한다. |
| `quality/targets.yaml` | latency, availability, recovery, capacity, 호환성 같은 측정 가능한 목표를 기록한다. |
| `quality/*.md` | 보안·privacy·접근성의 규범적 원칙과 예외 조건을 기록한다. |
| `architecture/overview.md` | system boundary, component와 data flow를 설명한다. |
| `architecture/stack.yaml` | 선택 기술, version 범위, 기능 근거, 지원·재검토 시점, 대안을 기록한다. |
| `architecture/workspaces.md` | 각 workspace를 분리한 이유와 배포·소유권 경계를 설명한다. |
| `architecture/decisions/` | 선택·기각 대안, trade-off와 무효화 조건을 ADR로 기록한다. |

### 정책 문서의 필수 구조

서비스 정책은 자유로운 회의록이 아니라 stable ID를 가진 규범 문서다.

```text
id: policy.<domain>.<name>
status: proposed | approved | deprecated
owner: <role-or-team>
applies_to: <roles-and-capabilities>

Intent
Rules
Failure behavior
Exceptions
Observability and audit
Related scenarios
Related contracts
```

예를 들어 `결제가 실패하면 무엇을 해야 하는가`는 `specs/policies/failures/`에 사용자 상태, 재시도, 중복 결제 방지, compensation, 안내, 운영 escalation을 기록한다. API status와 error code는 해당 policy ID를 참조하는 `contracts/`에 기록한다.

## 8. `contracts/` — 구성 요소 사이의 검증 가능한 약속

```text
contracts/
├── registry.yaml
├── services/
│   └── <interaction-id>.yaml
├── api/
├── events/
├── data/
│   ├── schema.dbml
│   └── modules/
├── schemas/
├── auth/
└── errors.yaml
```

| 파일 | 책임 |
|---|---|
| `registry.yaml` | contract ID, 형식, version, owner, provider, consumer, policy·scenario reference를 색인한다. |
| `services/<interaction-id>.yaml` | 구성 요소 사이의 사전조건, 제공 의무, 결과, timeout, retry, idempotency, partial failure와 compensation을 정의한다. |
| `api/` | 선택한 표준 형식의 request·response·endpoint 계약을 저장한다. |
| `events/` | event payload, ordering, delivery, duplicate, consumer 책임을 저장한다. |
| `data/schema.dbml` | 통합된 canonical database 구조를 제공한다. |
| `data/modules/` | domain별 DBML 원본을 분리해 충돌을 줄인다. |
| `schemas/` | 여러 workspace가 공유하는 validation·serialization schema를 저장한다. |
| `auth/` | identity, token, scope, permission propagation 계약을 저장한다. |
| `errors.yaml` | machine-readable error code, retryability, 사용자 노출과 policy reference를 저장한다. |

`contracts/`는 단순한 type 모음이 아니다. consumer와 provider가 성공·실패·지연·중복·부분 성공 상황에서 무엇을 해야 하는지를 검증 가능하게 정의한다.

다만 제품 전체의 의미를 계약에 중복하지 않는다.

- `specs/`: 사용자가 어떤 권리를 갖고 서비스가 어떤 정책으로 행동해야 하는지
- `contracts/`: 그 정책을 workspace와 외부 시스템 사이에서 어떤 protocol과 실패 의미로 지킬지
- test: spec scenario와 contract가 실제 구현에서 지켜지는지

각 contract는 관련 policy와 scenario ID를 참조하고, 각 test는 관련 contract 또는 scenario ID를 보고한다.

## 9. `docs/` — 설명, 운영, 파생된 읽기 화면

```text
docs/
├── index.md
├── guides/
│   ├── getting-started.md
│   ├── contributing.md
│   └── local-development.md
├── runbooks/
│   ├── deployment.md
│   ├── rollback.md
│   ├── backup-restore.md
│   └── incident-response.md
├── troubleshooting/
└── generated/
    ├── current-state.md
    ├── traceability.md
    └── release-readiness.md
```

`docs/`는 제품 정책의 별도 원본이 아니다. 사람이 수행할 운영 절차와 설명, 그리고 AI·CLI가 `specs`, `contracts`, `.harness`를 요약해 만든 읽기 화면을 제공한다.

`docs/generated/`는 수동 편집하지 않는다. Plugin이나 CLI가 없는 AI도 빠르게 현재 상태를 복구할 수 있도록 의미 있는 checkpoint에서만 재생성하고 Git에 포함한다. 매 commit마다 다시 만들어 불필요한 충돌을 만들지 않는다.

각 generated 문서는 generator version, 생성 시점, 참조한 source fingerprint를 header에 포함한다. 현재 `specs`, `contracts`, `.harness` fingerprint와 다르면 AI는 요약을 사실로 사용하지 않고 원본을 다시 읽거나 재생성한다.

## 10. Workspace의 정확한 정의

workspace는 다음 조건을 가진 **독립적인 구현·검증 단위**다.

- 명확한 제품 또는 기술 책임이 있다.
- 자체 build, test, lint, run 또는 이에 준하는 검증 명령이 있다.
- 소유자와 변경 범위를 지정할 수 있다.
- 하나 이상의 spec·contract를 구현, 제공 또는 소비한다.
- 다른 workspace와 구분되는 상태, 변경 범위와 검증 결과를 확인할 수 있다.

### 관련 용어와 차이

| 개념 | 의미 |
|---|---|
| workspace | 오케스트레이션이 작업·검사·소유권·계약을 관리하는 단위 |
| repository | 독립 Git history와 remote를 가진 저장 단위 |
| submodule | 별도 repository의 정확한 commit을 root에서 가리키는 Git 연결 방식 |
| module·package | 한 workspace 내부의 code organization 단위 |
| work item | workspace 일부를 변경하는 작업 단위 |

workspace는 submodule일 수 있지만 항상 submodule인 것은 아니다.

| workspace kind | 사용 상황 |
|---|---|
| `submodule` | 별도 repository, 권한, 배포, release 주기가 필요한 신규 workspace의 기본 권장 |
| `directory` | 하나의 monorepo 안에서 독립 build·test 경계를 가진 package 또는 app |
| `root` | 기존 repository root에 이미 제품 코드가 있어 root 자체가 구현 단위인 경우 |
| `external` | 구조 전환 전의 기존 sibling repository처럼 remote는 있지만 아직 submodule이 아닌 경우. 신규 프로젝트의 기본값으로 사용하지 않는다. |

예를 들어 web app, mobile app, API service, background worker, shared library, infrastructure가 각각 workspace가 될 수 있다. 작은 backend의 domain module을 무조건 별도 workspace나 repository로 만들지는 않는다.

새 프로젝트에서 독립 repository가 필요하다고 확정되면 submodule 연결을 기본 권장한다. 기존 프로젝트는 현재 topology를 보존하고 실질적인 이점이 있을 때만 전환한다.

## 11. 각 workspace에 들어가는 연결 파일

```text
<workspace>/
├── AGENTS.md
└── .harness/
    ├── workspace.yaml
    ├── commands.yaml
    ├── contracts.lock.yaml
    ├── ownership.yaml
    └── quality.yaml
```

| 파일 | 책임 |
|---|---|
| `AGENTS.md` | 이 workspace만 열었을 때도 orchestration root와 local harness를 먼저 읽도록 안내한다. |
| `workspace.yaml` | workspace ID, kind, 책임, root repository 위치, 제공·소비 contract를 기록한다. |
| `commands.yaml` | build, test, lint, typecheck, run, generate 명령을 shell-independent argv 형태로 연결한다. |
| `contracts.lock.yaml` | 구현이 기준으로 삼은 contract ID, version, fingerprint를 고정한다. 계약 원본을 복제하지 않는다. |
| `ownership.yaml` | 담당 domain, module, path, generated path와 보호할 공통 경계를 기록한다. |
| `quality.yaml` | TDD, test 종류, coverage, contract·integration check와 CI 필수조건을 연결한다. |

source, test, migration, generated directory의 실제 이름은 선택한 기술 스택의 표준을 따른다. 오케스트레이션 제품이 모든 framework에 같은 directory를 강제하지 않는다.

workspace만 단독으로 clone했는데 root가 없다면 AI는 계약이나 제품 정책을 추측하지 않는다. `workspace.yaml`의 root remote를 안내하고 orchestration root를 먼저 복원하도록 한다.

## 12. 외부 UI 목업과 디자인 입력

외부 UI 입력은 link, image, PDF, video, design file, HTML·CSS, component code, 별도 frontend project를 지원한다.

`specs/ui/sources/<source-id>.yaml`은 다음을 기록한다.

- source type과 provider
- URL 또는 tracked snapshot 경로
- immutable version, frame ID 또는 file hash
- owner, 접근 권한, license
- `reference`, `seed`, `canonical` 중 authority mode
- 관련 role, journey, screen과 state
- import·sync 방법과 마지막 확인 시점

### Authority mode

| mode | 의미 |
|---|---|
| `reference` | 구현 판단에 참고하지만 차이를 허용한다. 차이는 UI decision으로 설명한다. |
| `seed` | 처음 가져온 뒤 product code가 새 원본이 된다. 외부 변경을 자동 반영하지 않는다. |
| `canonical` | 지정한 외부 version이 UI 기준이다. 변경 시 source diff와 P40 재검토가 필요하다. |

AI가 source 성격에 따라 mode를 권장하고 의미가 애매할 때만 사용자에게 묻는다.

### Import 흐름

1. source와 authority 등록
2. untrusted file·script·dependency를 `.harness/local/imports/`에서 격리 검사
3. 역할·journey·normal·loading·empty·error·permission·responsive·accessibility coverage 비교
4. 제품 정책과 충돌하거나 빠진 상태를 보고
5. 역할·domain별 작은 단위로 UI workspace에 통합
6. interaction, accessibility, visual regression과 scenario test 수행
7. 외부 source가 바뀌면 기존 fingerprint와 비교하고 의미 변경은 사용자에게 확인

큰 binary snapshot은 필요한 경우에만 Git LFS를 사용한다. URL과 immutable version으로 재현할 수 있으면 repository에 중복 복사하지 않는다. 외부에서 완성된 UI를 받았더라도 P40 전체 UI gate는 동일하게 적용한다.

`canonical` source가 외부 서비스에만 있고 접근할 수 없으면 일치 여부를 `pass`로 간주하지 않는다. 재현 가능한 export가 있으면 검증하고, 없으면 P40 gate를 `unknown`으로 두어 제한사항을 알린다.

## 13. TDD 파일과 강제 범위

TDD는 모든 파일 수정에 적용하는 형식적 규칙이 아니라 **동작 변경과 버그 수정에 적용하는 release 필수 정책**이다.

### 필수 적용

- 새로운 product behavior
- bug fix와 regression
- API·event·service contract 변경
- database migration과 rollback
- 인증·권한·보안 정책
- UI interaction과 state transition
- infrastructure behavior와 failure recovery

### 명시적 예외

- 문서만 바뀌는 변경
- interaction·layout behavior를 바꾸지 않는 pure design asset
- canonical schema에서 결정적으로 생성되는 파일
- production에 병합하지 않는 feasibility spike
- 동작과 의미가 바뀌지 않는 mechanical formatting

### 증거

작업별 `tdd.yaml`은 다음을 기록한다.

- 관련 work, scenario, policy, contract ID
- 실패한 test command와 failure fingerprint
- 최소 구현 후 통과한 command와 result fingerprint
- 전체 regression 결과
- 사용한 예외 code와 근거

CI는 최종 동작과 test mapping, 통과 결과, 예외 유효성을 강제한다. 최종 repository snapshot만으로 test가 실제로 먼저 작성됐는지 완벽하게 증명할 수는 없다. AI가 수행하는 작업은 구현 전에 red evidence를 기록하고, 외부 PR은 behavior change에 대응하는 test와 regression evidence가 없으면 merge를 막는다.

긴급 incident에서는 사용자 영향 복구를 먼저 수행할 수 있지만 regression test와 evidence 없이는 RC와 release를 만들 수 없다.

## 14. 작업 충돌 정보와 조정

충돌 감지는 단순 path overlap만 보지 않는다.

- 파일과 directory
- module·package
- policy·scenario·UI flow
- API·event·DBML과 migration 순서
- shared type과 generated artifact
- dependency와 configuration
- workspace와 submodule pointer
- 이미 생성된 RC와 release baseline

### Scope claim

내장 provider의 `work/claims/<claim-id>.yaml` 또는 외부 task provider의 대응 필드는 다음을 가진다.

- claim ID와 work ID
- owner
- branch와 baseline commit
- workspace
- path·module·policy·scenario·contract·UI flow 범위
- dependency와 예상 merge 순서
- 생성, 마지막 확인, 만료 시점
- 상태와 충돌 해결 전략

branch 이름을 파일명으로 사용하지 않는다. slash와 rename 문제를 피하기 위해 stable claim ID를 사용하고 branch는 필드로 기록한다.

### 작업 시작 전 흐름

1. remote, branch, workspace, task와 contract 상태 refresh
2. 예상 변경 범위 계산
3. active claim, branch, PR, contract change와 비교
4. 충돌 위험과 의미를 구현 전에 사용자와 관련 담당자에게 알림
5. 누가 어디를 수정할지와 merge 순서를 정함
6. claim을 기록하고 baseline fingerprint를 고정
7. 작업 중 실제 범위가 넓어지면 재검사
8. PR 직전과 merge 직전에 다시 검사

### 선택 가능한 해결 전략

- 공통 policy·contract를 먼저 병합하고 각 workspace가 같은 기준에서 병렬 구현
- 여러 backend provider를 먼저 구현·병합한 뒤 frontend를 실제 연결
- frontend는 generated client와 mock server로 먼저 구현하고 provider 병합 뒤 실제 연결
- module, path, journey 또는 contract별로 소유 범위를 분리
- 충돌이 집중되는 공통 파일은 한 담당자가 순차 수정
- 불확실성이 큰 경우 adapter·feature flag 뒤에 대안을 구현하고 검증 후 하나를 선택
- 의미 충돌을 해결할 수 없으면 병렬 작업을 중단하고 dependency 순서로 진행

TDD는 합쳐진 구현이 같은 behavior를 지키는지 검증하지만 Git text conflict 자체를 예방하지 않는다. scope claim, ownership, contract baseline, merge preview와 test를 함께 사용한다.

## 15. 생성·추적·로컬 파일 규칙

| 분류 | 예 | Git 정책 |
|---|---|---|
| 규범 원본 | `specs/`, `contracts/`, 승인된 policy | 추적하고 review 필수 |
| 제어 원본 | manifest, workspace, policy, lifecycle baseline | 추적하고 schema validation 필수 |
| compact evidence | gate receipt, TDD receipt, checksum | 추적하되 raw log는 제외 |
| 파생 요약 | `docs/generated/` | checkpoint에서만 재생성하고 수동 편집 금지 |
| 기술 생성 코드 | client, server stub, shared type | 기술별 정책에 따라 추적 여부를 정하되 canonical source와 생성 명령 필수 |
| local-only | cache, raw log, 격리 import, credential | 추적 금지 |
| binary reference | 필요한 UI snapshot | 최소화하고 필요하면 Git LFS 사용 |

secret, token, private key, 개인별 절대 경로는 어느 추적 파일에도 저장하지 않는다. 경로는 project-relative 또는 workspace-relative로 기록하고 separator를 정규화한다.

## 16. 단계별 파일 생성 시점

| 생명주기 | 생성·갱신 영역 |
|---|---|
| 서비스 발견 | `.harness-drafts/`의 normalized summary, decision, open question |
| 프로젝트 초기화 | root 진입 파일, `.harness/manifest.yaml`, `entry.md`, 초기 `specs/` |
| 제품 기준선 | product, policies, scenarios, quality spec |
| architecture·기술 스택 | architecture spec, workspace registry, ADR, workspace repository·submodule |
| 전체 UI | UI source, coverage, flow, state, external import provenance |
| 계약·DBML | contract registry, service/API/event/data/auth/error contract |
| 구현 경계·골격 | workspace local harness, contract lock, ownership, generated stub 설정 |
| 기능 구현 | work claim, change proposal, TDD receipt와 실제 test |
| 통합·강화 | gate evidence, generated traceability, runbook |
| RC·사용자 검증 | release candidate manifest, readiness와 사용자 승인 evidence |
| 운영 | incident, runbook 개선, 다음 change와 stale 영향 |

빈 placeholder 파일을 처음부터 모두 만들지 않는다. 해당 단계에서 실제 정보가 생길 때 schema-valid한 파일을 만들며 index와 registry가 누락을 탐지한다.

이 표는 directory를 한 번에 채우는 waterfall 일정을 의미하지 않는다. 역할·domain·capability별 작은 변경으로 파일을 만들고 즉시 통합하며, 새 사실이 생기면 관련 spec·contract·state만 다시 연다.

## 17. 기존 프로젝트 도입

기존 프로젝트에는 위 directory를 무조건 강제하지 않는다.

1. 기존 `docs`, contract, architecture, task, Git 구조를 읽기 전용으로 inventory한다.
2. 이미 source of truth가 있는 내용은 새 위치에 복제하지 않고 `.harness/manifest.yaml`의 path mapping으로 연결한다.
3. 기존 `.harness`, `specs`, `contracts` 이름이 다른 용도로 사용 중이면 덮어쓰지 않고 collision report와 대체 path를 제안한다.
4. `README.md`, `AGENTS.md`, CI 파일은 관리 구간 단위 diff로만 변경한다.
5. 구조 통일의 이점이 migration 비용보다 클 때만 이동을 제안하고 사용자 승인 후 수행한다.

기존 프로젝트에서도 개념적 책임은 유지한다. 실제 directory 이름은 달라도 product semantics, boundary contracts, orchestration state, explanatory docs의 source of truth가 각각 하나여야 한다.

## 18. 실제 사용 예시

사용자가 `결제 실패 정책을 바꿔줘`라고 말하면 AI는 다음을 수행한다.

1. 관련 `policy.*`와 scenario를 찾아 현재 규칙을 요약한다.
2. payment API·event·service contract와 관련 backend·frontend workspace를 찾는다.
3. active claim과 PR을 확인해 충돌 위험을 먼저 알린다.
4. 정책 변경안을 사용자에게 설명하고 승인받는다.
5. policy와 scenario를 먼저 변경해 후속 baseline을 `stale`로 만든다.
6. contract change와 DBML 영향이 있으면 canonical contract를 변경한다.
7. generated stub과 workspace contract lock을 갱신한다.
8. 실패하는 regression test를 만들고 TDD evidence를 기록한다.
9. backend provider를 구현·병합하고 frontend를 실제 연결한다.
10. 전체 scenario, contract, integration test로 정책이 지켜지는지 확인한다.

사용자는 어느 파일을 직접 열어야 하는지 몰라도 된다. AI는 수정한 policy, contract, workspace와 검증 결과를 사람이 이해할 수 있는 문장으로 보고하고 요청 시 diff를 보여준다.

## 19. 이번 설계에서 확정하는 결정

- 기본 구조는 `specs/`, `contracts/`, `.harness/`, `docs/` 네 책임 영역으로 나눈다.
- 분리 기준은 사람과 AI가 아니라 제품 의미, 경계 계약, 오케스트레이션 상태, 설명·운영 책임이다.
- 서비스 정책과 실패 행동은 stable policy·scenario ID로 `specs/`에 저장한다.
- 구성 요소 사이의 성공·실패·timeout·retry·idempotency·compensation 의무는 `contracts/services/`에 저장하고 policy ID를 참조한다.
- workspace는 독립 구현·검증 단위이며 submodule은 가능한 workspace kind 중 하나다.
- 신규 독립 repository workspace에는 submodule을 기본 권장하고 기존 topology는 보존한다.
- 외부 UI는 reference, seed, canonical mode로 등록하고 격리 검사와 전체 UI coverage gate를 통과한다.
- 동작 변경과 bug fix에는 TDD를 필수로 적용하고 좁고 명시적인 예외만 허용한다.
- 작업 시작 전에 path뿐 아니라 policy, contract, migration, UI flow, workspace 충돌을 검사하고 scope claim을 기록한다.
- 생성 파일과 source of truth, compact evidence, 파생 요약, local-only 파일을 구분한다.
- 이름 확정 전에도 draft capsule에 정규화된 프로젝트 이해를 계속 저장한다.
- 기존 프로젝트에는 default directory 이름보다 source of truth 보존과 비파괴 도입을 우선한다.
