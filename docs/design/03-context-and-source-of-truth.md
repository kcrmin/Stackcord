# 컨텍스트, 원본, 압축 복구 설계

> 상태: 확정
>
> 마지막 갱신: 2026-07-16
>
> 목표는 특정 AI 대화나 한 사람의 기억에 기대지 않고, 새 작업자와 새 AI가 repository를 읽어 같은 제품 의미와 현재 작업 상태를 복구하게 하는 것이다.

## 1. 핵심 결정

- 대화와 AI memory는 source of truth가 아니다. 승인된 결과만 정규화해 repository에 저장한다.
- 제품 의미는 `specs/`, 구성 요소 사이의 의무는 `contracts/`, 조정 상태는 `.harness/`, 설명과 runbook은 `docs/`가 소유한다.
- 마지막으로 수정된 파일이 자동으로 이기지 않는다. 서로 다른 원본이 충돌하면 상태를 `unknown` 또는 `stale`로 바꾸고 의미를 조정한 뒤 다시 승인한다.
- 같은 사실을 여러 파일에 복제하지 않고 stable ID와 fingerprint로 연결한다.
- 모든 AI는 변경 전에 짧은 context refresh를 수행하고, 압축 이후나 기억이 의심될 때 전체 context audit를 수행한다.
- Hook은 편의 기능일 뿐 정확성의 전제조건이 아니다. Plugin이 없거나 Hook이 실행되지 않아도 CLI와 Markdown 절차만으로 복구할 수 있어야 한다.

## 2. 원본 우선순위

우선순위는 “무엇이 더 중요하냐”가 아니라 **어떤 종류의 사실을 누가 소유하느냐**를 뜻한다.

| 사실 종류 | 정식 원본 | 보조 자료 |
|---|---|---|
| 실제 파일, commit, branch, submodule pointer, CI 결과 | 현재 filesystem, local Git, remote Git | `.harness`의 마지막 snapshot |
| 승인된 제품 의도·정책·시나리오·품질 | `specs/` | 생성 요약, 작업 항목 |
| API·event·data·failure 의무 | `contracts/` | 구현 코드, 생성 client |
| workspace·baseline·gate·claim·change 연결 | `.harness/` | task provider |
| 작업 상태·담당자·dependency | 선택한 단 하나의 live task provider | `.harness/work/links.yaml` |
| 검증 실행 사실 | 재현 가능한 evidence와 CI | 생성 보고서 |
| 사용법·운영 절차 | `docs/`와 실행 가능한 script/command | AI 설명 |
| 현재 대화 | 원본이 아님 | 결정 후보를 추출하는 입력 |

실제 코드가 승인된 spec과 다르면 “코드가 있으므로 정책이 바뀐 것”으로 간주하지 않는다. 구현 결함인지 의도 변경인지 확인하고, 의도 변경이면 change 절차를 통해 spec과 영향을 먼저 갱신한다.

## 3. 컨텍스트 개체 모델

다음 개체가 repository 전체의 공통 언어다.

| 개체 | 의미 | 정식 위치 |
|---|---|---|
| Project | 하나의 제품과 오케스트레이션 root | `.harness/manifest.yaml` |
| Workspace | 독립 구현·검증·소유권·계약 경계 | `.harness/workspaces.yaml` |
| Role | 서비스를 사용하는 사람·시스템 역할 | `specs/product/roles.yaml` |
| Capability | 제품이 제공하는 결과 단위 | `specs/product/capabilities.yaml` |
| Journey | 역할이 목적을 달성하는 end-to-end 흐름 | `specs/product/journeys/` |
| Policy | 성공·실패·권한·예외 때 지켜야 할 규칙 | `specs/policies/` |
| Scenario | 관찰 가능한 사례와 기대 결과 | `specs/scenarios/` |
| Decision | 이유와 대안을 보존하는 승인 결정 | `specs/**/decisions/` 또는 ADR |
| Contract | 구성 요소·외부 시스템 사이의 검증 가능한 의무 | `contracts/` |
| Work item | 구현하거나 조사할 계획 단위 | live task provider 또는 내장 provider |
| Claim | 한정된 기간 동안 변경하려는 범위 선언 | `.harness/work/claims/` |
| Change | 승인된 의미·경계를 변경하는 제안 | `.harness/work/changes/` |
| Baseline | 승인된 문서 집합의 fingerprint 묶음 | `.harness/state/baselines.yaml` |
| Gate | 다음 단계로 가기 위한 검사와 승인 | `.harness/state/gates/` |
| Evidence | 검사 사실을 재현할 수 있는 영수증 | `.harness/evidence/` |
| Release candidate | 함께 검증할 정확한 commit·artifact 집합 | `.harness/state/release-candidate.yaml` |

`workspace`는 submodule과 같은 말이 아니다. `workspace.kind`는 `submodule`, `directory`, `root`, `external` 중 하나이며, 별도 repository가 필요한 신규 경계에는 submodule을 적극 권장한다.

## 4. ID, 참조, fingerprint

### 사람이 유지하는 의미 ID

제품 의미는 경로나 제목이 바뀌어도 유지되는 영어 lowercase dot namespace를 사용한다.

```text
role.customer
capability.account.recovery
journey.customer.restore-access
policy.account.recovery.rate-limit
scenario.account.recovery.expired-token
contract.identity.recovery.v1
```

- 외부 ticket 번호나 branch 이름을 stable ID로 쓰지 않는다.
- 흔히 말하는 “ticket slug”를 필수 개념으로 두지 않는다. 읽기 좋은 branch 설명은 필요하지만 제품 의미 ID와 분리한다.
- work, claim, operation처럼 반복 생성되는 instance는 정렬 가능한 ULID를 사용한다.
- 파일 이름이 ID를 반영할 수는 있지만 파일 경로 자체가 identity는 아니다.

### 모든 정식 문서의 공통 metadata

문서 유형에 맞는 다음 field를 frontmatter 또는 YAML에 둔다.

```yaml
schema_version: 1
id: policy.account.recovery.rate-limit
kind: policy
status: approved
revision: 3
owners: [workspace.identity]
refs:
  - scenario.account.recovery.rate-limited
sources:
  - source: decision.product.account-recovery
updated_at: 2026-07-16T00:00:00Z
```

`created_by: ai` 같은 표시는 branch, commit, ID에 넣지 않는다. 필요한 경우 생성 provenance는 evidence에만 기록한다.

### fingerprint

- text는 UTF-8, LF, trailing whitespace 정규화 후 SHA-256을 계산한다.
- structured data는 key order와 line ending을 canonicalize한 뒤 계산한다.
- baseline fingerprint는 포함된 stable ID, revision, content fingerprint의 정렬된 목록으로 계산한다.
- 생성 요약에는 자신을 만든 source fingerprint를 반드시 넣는다. 다르면 즉시 `stale`이다.

## 5. 파일 형식

| 형식 | 사용처 | 이유 |
|---|---|---|
| Markdown + YAML frontmatter | 정책, 시나리오, 결정, 설명 | 맥락이 긴 사람이 작성하는 내용과 기계 metadata를 함께 보존 |
| YAML | inventory, policy configuration, 연결, 승인 상태 | 사람이 검토하고 작은 diff를 만들기 쉬움 |
| JSON | 생성 index, 검사 결과, machine output | parser 간 모호성을 줄이고 schema 검증하기 쉬움 |
| JSON Schema | YAML·JSON의 기계 검증 | CLI와 AI가 같은 validation rule 사용 |
| DBML과 표준 계약 형식 | database와 API/event contract | 외부 도구와 호환되고 diff 가능 |

생성 JSON을 직접 고치지 않는다. 원본을 수정하고 refresh로 재생성한다. narrative를 억지로 거대한 YAML 하나에 넣지 않는다.

## 6. 추가되는 컨텍스트 파일

```text
.harness/
├── sources.yaml
├── state/
│   ├── context-index.json
│   └── impact-graph.json
├── local/
│   └── state/
│       └── current.json
└── work/
    └── branches/
        └── <branch-key>.yaml
```

| 파일 | 책임 |
|---|---|
| `sources.yaml` | local/remote Git, task provider, dbdiagram, UI source 등 외부 원본과 권한·freshness 정책을 등록한다. credential은 넣지 않는다. |
| `context-index.json` | stable ID → path, revision, fingerprint, status, references를 생성한다. |
| `impact-graph.json` | spec, policy, scenario, contract, workspace, test, work item 사이의 영향을 생성한다. |
| `local/state/current.json` | 현재 worktree의 branch, workspace commit, dirty 상태, pointer 차이, active work·claim과 다음 gate를 cache한다. Git에서 제외하며 삭제해도 정확성이 변하지 않는다. |
| `work/branches/<branch-key>.yaml` | 현재 branch가 다루는 work, baseline, workspace branch, claim, checkpoint만 기록한다. 다른 branch의 내용을 읽어 현재 결정으로 사용하지 않는다. |

branch record는 handoff 문서가 아니라 **현재 작업 경계와 복구 checkpoint**다. merge 뒤에는 검증 영수증으로 축약해 evidence에 보관하고 active record는 정리한다. 각자가 맡은 일을 계속하는 평상시에도 context 유지에 쓰며, 실제 담당 변경이 있을 때만 handoff field를 추가한다.

## 7. Context refresh 절차

AI나 사람은 “무엇을 해야 해?”라고 묻기만 하면 된다. CLI 또는 Skill이 다음을 수행한다.

1. 현재 경로에서 가장 가까운 `.harness/manifest.yaml`을 찾고 repository trust 상태를 확인한다.
2. schema version과 migration 필요 여부를 확인한다.
3. filesystem, root Git, 각 workspace Git, remote tracking, submodule pointer를 read-only로 읽는다.
4. `sources.yaml`에서 현재 작업에 필요한 외부 원본만 조회한다. 인증이 없거나 offline이면 `unknown`으로 표시한다.
5. authored 문서를 schema validation하고 context index, impact graph, baseline fingerprint를 다시 계산한다.
6. live task status와 local link의 차이, dirty file, stale generated document, 만료 claim, contract mismatch를 찾는다.
7. 사실·가정·unknown·blocker를 구분한 짧은 context pack과 안전한 다음 행동 하나를 제시한다.

refresh는 read-only가 기본이다. `local/state/current.json` cache는 안전하게 다시 만들 수 있지만 결과 정확성은 cache write에 의존하지 않는다. 추적되는 `context-index.json`과 `impact-graph.json`은 authored source가 승인된 checkpoint에서만 계획·갱신해 일상적인 status 조회가 Git diff를 만들지 않게 한다.

## 8. 압축·기억 손실 복구

Repository에는 `audit-project-context` Skill과 Markdown fallback을 둔다.

### 평상시

- AI는 `AGENTS.md` → `.harness/entry.md` → actual state와 `local/state/current.json` cache → 현재 work/claim → 관련 spec/contract 순으로 읽는다.
- `current.json`의 source fingerprint가 일치하면 전체 문서를 매번 다시 읽지 않는다.
- 답변에는 중요한 결정의 stable ID를 사용하고 추측은 명시한다.

### 다음 경우 전체 audit

- 대화 context가 압축되었거나 새 AI·새 사람이 시작했다.
- AI가 이미 정한 질문을 반복하거나 현재 branch/workspace를 확신하지 못한다.
- pull, branch 전환, submodule pointer 변경, contract 변경, task provider 변경이 있었다.
- 생성 요약 fingerprint가 원본과 다르다.
- 사용자가 “기억 못 하는 것 같다”, “현재 상태 다시 확인해”라고 말한다.

### 선택 Hook

Codex Plugin에서는 신뢰된 repository에 한해 SessionStart와 PostCompact Hook이 refresh 필요 여부를 알릴 수 있다. Hook이 파일을 임의 수정하거나 외부 명령을 설치하지는 않는다. Hook 미지원 client에서는 Skill이 동일한 절차를 직접 수행한다.

## 9. 변경과 stale 전파

변경 전 `impact-graph.json`으로 직접·간접 영향을 계산한다.

| 변경 | 자동 stale 대상 예시 |
|---|---|
| product policy | 관련 scenario, UI state, contract obligation, test, release gate |
| contract | provider/consumer workspace, generated client, mock, compatibility check |
| DBML | migration plan, data policy, repositories, seed/test fixture, rollback evidence |
| UI canonical baseline | flow coverage, interaction test, accessibility evidence |
| workspace pointer | root RC, cross-repo change bundle, generated status |
| technology decision | architecture baseline, build/CI/security/operations evidence |

stale는 실패가 아니라 “이전 승인이 현재 변경을 아직 검증하지 않았다”는 상태다. 영향이 없음을 증명하면 다시 승인할 수 있고, 의미가 바뀌면 관련 원본부터 갱신한다.

## 10. 외부 원본과 offline 규칙

- task status의 live source는 동시에 하나만 둔다. GitHub Issue와 Jira를 동시에 상태 원본으로 쓰지 않는다.
- 외부 service의 last-known snapshot은 cache일 뿐 원본이 아니다.
- offline에서도 local spec, contract, work link, Git 상태로 가능한 일을 계속하고 외부 상태는 `unknown`으로 표시한다.
- dbdiagram의 시각적 수정은 Git DBML을 자동 덮어쓰지 않는다. scratch로 pull하여 semantic diff와 수정 이유를 확인한 뒤 change로 반영한다.
- Figma나 외부 mockup도 authority가 `reference`, `seed`, `canonical` 중 무엇인지 명시한다.

## 11. 예시: 새 작업자가 clone한 뒤 이어서 개발

사용자: “이 프로젝트에서 지금 뭐 해야 해?”

AI가 진단한 결과:

```text
현재 제품 baseline은 승인됨.
workspace.identity는 origin/main보다 2 commits 뒤이며 local 변경 없음.
contract.identity.recovery.v1 변경이 workspace.web의 generated client를 stale로 만듦.
work item GH-142는 진행 중이고 claim은 workspace.identity의 recovery handler와 contract만 포함함.
다음 안전 행동: 모든 workspace를 정확한 root pointer로 동기화한 뒤 contract compatibility test를 실행.
```

AI는 사용자가 `git submodule update`나 context 파일을 직접 다룰 것을 요구하지 않는다. 동기화가 fast-forward가 아니거나 local 변경을 건드릴 가능성이 있으면 이유와 선택지를 보여주고 멈춘다.

## 12. 수용 기준

- 대화 기록 없이 clone한 사람이 현재 단계, 승인 baseline, 진행 work, 막힌 항목을 복구할 수 있다.
- AI가 같은 질문을 반복하지 않고 기존 결정과 open question을 구분한다.
- 실제 Git 상태와 생성 요약이 다르면 불일치를 숨기지 않는다.
- 제품 정책, contract, task status, orchestration state가 서로의 원본을 침범하지 않는다.
- Plugin, Hook, 외부 provider 없이도 repository 파일과 CLI/Markdown만으로 복구된다.
- Windows와 macOS에서 동일 fingerprint와 stable ID validation 결과가 나온다.
