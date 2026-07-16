# 제품 범위, source repository, 배포 청사진

> 상태: 확정 — 공개 이름만 사용자의 이전 결정에 따라 보류
>
> 마지막 갱신: 2026-07-16

## 1. 제품 정의

이 제품은 특정 framework를 생성하는 boilerplate가 아니다. AI와 사람이 **서비스 전체를 발견하고, multi-repo full-stack 구조를 만들고, 같은 제품 의미와 계약을 읽으며 병렬 개발하고, 검증된 release까지 이어가게 하는 local-first orchestration product**다.

한 문장 설명:

> 빈 폴더나 기존 clone에서 시작해 제품 의도·UI·계약·DB·workspace·작업·검증을 하나의 추적 가능한 graph로 연결하고, AI가 현재 상태와 다음 일을 판단해 production release까지 돕는다.

## 2. 명확히 포함하는 것

- 긴 서비스 발견 대화를 정규화해 계속 저장하고, 이름 승인 전에도 복구
- framework·language·database·cloud를 상세 질문과 현재 공식 상태로 선택
- 신규 root와 기존 repository의 비파괴 adoption
- directory/root/submodule/external workspace 설계와 생성·연결
- 전체 제품 명세, policy, scenario, quality, architecture, UI baseline
- 외부 UI mockup import와 authority/provenance/coverage
- contract, failure behavior, DBML, dbdiagram 시각 협의
- interface skeleton과 vertical-slice TDD 개발 순서
- task provider 연결, dependency, claim, conflict preflight, merge order
- context compression·새 작업자·새 AI 복구
- Git/PR/submodule pointer/cross-repo integration
- production gate, RC, 사용자 검증, release, 운영/issue 진단

## 3. 포함하지 않는 것

- 특정 web/mobile/backend framework 고정 template
- Superpowers, BMAD, Beads를 재구현하거나 묶어 설치하는 meta-bundle
- 중앙 SaaS account, source upload, 기본 telemetry
- AI가 승인 없이 production을 배포하는 autopilot
- 모든 repository를 무조건 submodule로 바꾸는 migration tool
- 대화 전체를 그대로 저장하는 memory product
- task provider를 GitHub Issue 하나로 강제하는 정책
- design tool 자체를 대체하는 UI editor

## 4. 차별점

일반 workflow Skill은 “어떻게 생각하고 코딩할지”를 돕고, task tool은 “무슨 일이 열려 있는지”를 기록한다. 이 제품은 그 사이의 끊어진 상태를 연결한다.

```text
제품 의도·정책·시나리오
        ↕
UI·contract·DBML baseline
        ↕
workspace·submodule 실제 Git 상태
        ↕
work·claim·dependency·PR
        ↕
test evidence·RC·release
```

모든 연결에 stable ID와 fingerprint가 있어 AI memory 대신 repository에서 현재 상태를 재구성할 수 있다는 점이 핵심 강점이다.

## 5. 사용자가 정의하는 것과 제품이 정하는 것

### 사용자가 반드시 정의·승인

- 어떤 사용자에게 어떤 문제와 가치를 제공하는지
- 역할, 핵심 journey, 성공·실패·예외 정책
- 법적·privacy·security·data retention 요구
- 제품 범위와 의도적 제외 범위
- 외부 UI source의 의미와 canonical 여부
- architecture에 영향을 주는 조직·비용·운영 제약
- production RC의 실제 사용자 검증과 final publish

AI는 누락되기 쉬운 선택과 대안을 객관식 + `기타`로 제안하지만 사용자 대신 제품 가치를 발명해 승인하지 않는다.

### 제품/AI가 기본으로 정하고 알림

- 폴더 convention, stable ID, schema, fingerprint
- protected main, short-lived branch, Conventional Commits, squash 기본값
- TDD 범위와 conflict preflight
- source-of-truth 분리, context refresh, stale 전파
- safe local command, evidence, result schema
- framework 중립 생성 구조와 cross-platform encoding/path 규칙

### 조건을 보고 추천하고 사용자와 선택

- technology stack과 새 dependency
- workspace 경계와 submodule 여부
- task/Git/design/database provider
- contract evolution과 deploy/merge order
- breaking migration, 외부 write, production action

기술 stack은 한 번에 선호만 묻지 않는다. 제품 기능, scale, real-time/offline, data consistency, team skill, hosting, cost, maintenance/security release 상태를 바탕으로 후보를 비교하고 공식 자료를 다시 확인한다.

## 6. 공개 이름 보류 규칙

사용자가 앞서 “이름은 나중에 짓자”고 정했으므로 지금 억지로 brand를 확정하지 않는다.

- 현재 source directory와 문서에서는 `fullstack-orchestrator`를 설명용 working ID로만 사용한다.
- public GitHub repository 생성, package namespace, binary 이름, Plugin name을 만들기 직전에 name clearance를 수행한다.
- 후보는 GitHub/package manager/domain/trademark/confusing similarity를 확인한다.
- 한 번 public package를 낸 뒤 이름을 바꾸면 update path와 trust가 깨지므로 1.0.0 이전에 고정한다.
- 내부 schema와 generated project는 brand에 종속되지 않는 identifier를 사용해 rename 비용을 줄인다.

이 한 항목만 의도적으로 사용자 선택을 기다리며, 나머지 설계·구현은 working ID로 진행할 수 있다.

## 7. Source repository 구조

```text
<product-repo>/
├── .codex-plugin/plugin.json
├── .agents/plugins/marketplace.json
├── .github/
│   ├── ISSUE_TEMPLATE/
│   ├── workflows/
│   ├── CODEOWNERS
│   └── pull_request_template.md
├── skills/                    # Codex/Agent Skills workflow
├── hooks/hooks.json           # optional trusted Hook
├── references/                # Skill이 필요한 때만 읽는 shared reference
├── compatibility.json         # Plugin·CLI·harness·adapter 지원 범위
├── cli/
│   ├── cmd/
│   ├── internal/
│   └── go.mod
├── schemas/                   # harness와 result JSON Schema
├── templates/                 # generated project templates
├── locales/
│   ├── en/
│   └── ko/
├── docs/
│   ├── getting-started/
│   ├── concepts/
│   ├── guides/
│   ├── reference/
│   ├── security/
│   └── contributing/
├── examples/
│   ├── starter/
│   └── multi-repo/
├── testdata/
├── scripts/                   # build/validation helpers; install 주 경로 아님
├── packaging/
│   ├── homebrew/
│   └── winget/
├── CHANGELOG.md
├── CONTRIBUTING.md
├── GOVERNANCE.md
├── LICENSE
├── README.md
├── SECURITY.md
└── SUPPORT.md
```

### 각 top-level 책임

| 폴더/파일 | 책임 |
|---|---|
| `.codex-plugin/` | Codex가 읽는 최소 plugin manifest |
| `.agents/plugins/` | GitHub repository를 plugin marketplace로 추가할 catalog |
| `skills/` | 상황별 자연어 workflow와 CLI routing |
| `hooks/` | 압축/새 session context 안내의 선택 trigger |
| `references/` | Skill 본문을 작게 유지하는 공통 정책 참조 |
| `compatibility.json` | Plugin, CLI, harness schema, adapter version 호환 범위 |
| `cli/` | deterministic Go core |
| `schemas/` | project file과 machine result의 versioned contract |
| `templates/` | 신규/adopt project에 생성하는 neutral file |
| `locales/` | English canonical과 Korean parity resource |
| `docs/` | 사용자·운영·보안·기여 documentation |
| `examples/` | 실제 사용 가능한 single/multi-repo sample |
| `testdata/` | Git topology, migration, provider, malicious input fixture |
| `packaging/` | Homebrew/WinGet release metadata |
| `.github/` | CI, release, Issue/PR collaboration |

## 8. 공개 배포 channel

| 구성 | 배포 방식 |
|---|---|
| source와 issue | public GitHub repository |
| Codex Plugin | 같은 repository의 marketplace catalog와 plugin package |
| CLI binary | GitHub Releases의 signed archive/MSI |
| macOS install | Homebrew tap |
| Windows install | WinGet; signed MSI/zip manual fallback |
| schemas/templates | CLI와 함께 embed하고 source에서도 공개 |
| docs | repository docs + versioned documentation site는 필요할 때 추가 |

Plugin만 설치해도 질문·workflow는 보이지만 deterministic 검사와 생성에는 CLI 설치를 추천한다. CLI가 없으면 Skill이 설치를 강제하지 않고 기능 제한과 Markdown fallback을 설명한다.

## 9. 사용 흐름

### 새 프로젝트

```text
Plugin/CLI 설치
→ “새 서비스 시작하자”
→ 이름 전 draft에 발견 결과 계속 저장
→ 핵심 가치·journey 승인
→ root와 Git/workspace 계획 생성
→ 전체 제품·stack·UI·contract·DBML 정의
→ boundary와 vertical slice TDD 구현
→ integration·production gate
→ RC 고정
→ 사용자 검증
→ final release
```

### 다른 사람이 clone

```text
root clone --recurse-submodules
→ “이 프로젝트 이어서 해. 지금 뭐 해야 해?”
→ repo-local Skill이 actual Git·pointer·context·task 상태 audit
→ missing tool/auth와 local divergence만 알림
→ conflict 없는 다음 work 추천 또는 맡은 work 복구
→ 같은 policy·contract·TDD·PR 흐름으로 계속 개발
```

### 기존 프로젝트 adoption

```text
existing repo에서 “이 구조로 이어서 관리해”
→ 비파괴 inventory와 risk report
→ 기존 AGENTS/docs/Git/topology 보존
→ 관리 파일을 diff로 제안
→ 승인한 영역부터 baseline 생성
→ 미정 영역은 unknown/stale로 남기고 점진적으로 안정화
```

## 10. 개발 순서

1. versioned domain model과 JSON Schema
2. read-only doctor/status/context core
3. generated project template와 repo-local Skill
4. Git/submodule/worktree actual-state adapter
5. work/claim/conflict/contract/DBML/UI 검사
6. plan/apply journal과 approval engine
7. init/adopt/resume lifecycle
8. Codex Skills와 optional Hook
9. GitHub/task/dbdiagram adapters
10. RC/release/supply-chain pipeline
11. macOS·Windows E2E와 docs parity
12. independent security/release review

각 단계는 vertical end-to-end fixture를 통과하며, 모든 core behavior는 TDD로 구현한다. 전체 UI나 backend를 한꺼번에 끝낸 뒤 연결하는 waterfall로 진행하지 않는다.

## 11. 공개 전 필수 repository 설정

- public name/namespace clearance와 repository 생성
- Apache-2.0 license와 third-party notice
- branch protection, required checks, CODEOWNERS
- private vulnerability reporting와 SECURITY.md
- OIDC 기반 최소 권한 release workflow
- signing identity와 key recovery/rotation
- Homebrew/WinGet publisher setup
- issue/discussion/support policy
- maintainer governance와 release owner
- domain/documentation URL은 선택 사항이지만 package identity는 고정

## 12. 출시 완료 정의

다음 모두가 충족돼야 “배포 완료”다.

- source만 있는 것이 아니라 signed CLI와 install channel이 실제 동작
- GitHub URL로 Plugin marketplace 설치 가능
- 새 프로젝트와 기존 clone continuation E2E 성공
- Plugin 없는 fallback 성공
- macOS/Windows clean install·upgrade·uninstall 성공
- multi-repo/submodule conflict와 contract change journey 성공
- DBML↔dbdiagram 격리 sync 성공
- production readiness와 security gate 통과
- 사용자 검증한 동일 RC를 1.0.0으로 공개
- issue/support/rollback 운영 가능

## 13. 현재 설계와 실제 출시 사이

이 문서 묶음은 product specification과 implementation blueprint를 확정한 것이다. 아직 Go CLI source, Plugin package, signed artifact, marketplace, public repository를 만든 것은 아니다. 공개 이름을 정한 뒤 implementation plan을 TDD로 실행하고 release gate를 통과해야 실제 출시다.
