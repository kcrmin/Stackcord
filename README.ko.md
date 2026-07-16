# Full-stack Orchestrator

한국어 | [English](./README.md)

> 현재는 작업용 이름입니다. 공개 제품명은 첫 public package를 배포하기 전에 확정합니다.

Full-stack Orchestrator는 사람과 AI가 서비스 발견부터 제품 정의, 풀스택 구현, 협업, 검증, release까지 이어가도록 돕는 local-first 오케스트레이션 제품입니다. 대화가 압축되거나 담당자·컴퓨터·repository·도구가 바뀌어도 제품 의도와 현재 작업 상태를 잃지 않는 것이 핵심입니다.

## 현재 상태

**Production 제품 설계와 구현 계획은 완료됐지만, 제품 자체는 아직 구현하거나 출시하지 않았습니다.**

현재 이 repository에 있는 것:

- 확정된 제품·협업 설계
- 생성 프로젝트와 source-of-truth 명세
- Git, submodule, 충돌, AI 승인, adapter, 보안, release 정책
- 실제 사용 대화와 생성 결과 walkthrough
- production 제품을 만드는 상세 TDD 구현 계획

아직 없는 것:

- 설치 가능한 Go CLI
- Codex Plugin package와 marketplace
- 서명된 macOS·Windows 설치 파일
- Homebrew·WinGet package
- 공개 production `1.0.0` release

제품을 검토하려면 [설계 인덱스](./docs/design/index.md), 구현을 시작하려면 [Production 구현 계획](./docs/superpowers/plans/2026-07-16-fullstack-orchestrator-production.md)을 먼저 읽습니다.

## 어떤 문제를 해결하나

일반적인 AI 개발은 대화가 압축되거나 작업자가 바뀌면 이전 결정과 현재 상태를 잃기 쉽습니다. 작업관리 도구는 누가 무엇을 맡았는지 알지만 제품이 왜 그렇게 행동해야 하는지는 모를 수 있고, Git은 무엇이 바뀌었는지 알지만 변경 의도와 정책까지 보존하지는 않습니다.

이 제품은 다음을 하나의 검증 가능한 관계로 연결합니다.

```text
제품 의도와 서비스 정책
→ 실행 가능한 전체 UI 기준선
→ contract와 DBML
→ workspace와 실제 Git 상태
→ 작업·claim·dependency·PR
→ test evidence·RC·release
```

stable ID와 fingerprint를 사용하므로 새 사람이나 새 AI가 대화 기록 없이 repository에서 현재 상태를 다시 계산할 수 있습니다.

## 사용자는 어떻게 사용하나

사용자는 내부 명령을 외우지 않고 AI에게 자연어로 말합니다.

```text
“새 서비스 시작하자.”
“이 프로젝트를 clone했어. 이어서 해줘.”
“지금 뭐 해야 해?”
“다른 작업과 충돌하는지 확인해줘.”
“외부에서 만든 이 mockup으로 UI를 시작해줘.”
“DB 구조 같이 정하고 dbdiagram에서 보여줘.”
“너 프로젝트 내용을 잊은 것 같아. 다시 검사해.”
“Production release 준비해.”
```

AI는 상황에 맞는 Skill을 선택하고, CLI는 실제 filesystem, Git, workspace, contract, task provider, 검증 상태를 확인합니다.

[전체 사용자 walkthrough](./docs/design/12-user-experience-walkthrough.md)에는 실제 객관식 질문, 답변 뒤 갱신되는 파일, GitHub Issue, branch, Draft PR, submodule 통합, context 복구, RC 승인까지 자세히 적혀 있습니다.

## 전체 개발 흐름

```text
진입 진단
→ 서비스 발견
→ 프로젝트 초기화
→ 제품 전체 정의
→ 아키텍처·기술 stack 선택
→ 실행 가능한 전체 UI 기준선
→ contract·DBML
→ 안정적인 구현 경계·골격
→ 수직 단위 풀스택 구현
→ 통합
→ 프로덕션 강화
→ RC
→ 사용자 검증
→ Release
→ 운영과 다음 변경
```

이 순서는 waterfall 일정이 아닙니다. 각 단계는 dependency gate이고 역할·domain·journey별 작은 변경을 계속 통합합니다. 뒤에서 새 사실이 발견되면 영향받은 이전 단계만 `stale`로 다시 엽니다.

## 주요 기능

| 기능 | 제공 내용 |
|---|---|
| 서비스 발견 | 한 번에 하나의 적응형 질문, 권장 객관식과 자유 입력, 확정·가설·미정 분리 |
| Context 유지 | 정규화된 결정, open question, stable ID, fingerprint, impact graph, 압축 후 복구 |
| 프로젝트 생성·도입 | 신규 프로젝트 생성과 기존 repository 비파괴 adoption |
| 풀스택 하네스 | 제품 명세, 서비스 정책, contract, DBML, 작업 상태, evidence, 운영 문서 |
| Workspace 관리 | framework를 강제하지 않는 root·directory·submodule·external workspace |
| Git 협업 | protected `main`, 짧은 branch, Conventional Commits, Draft PR, worktree, exact submodule pointer |
| 충돌 방지 | path뿐 아니라 module·policy·scenario·contract·migration·UI flow·dependency·pointer 사전 검사 |
| 작업 관리 | 내장 Git fallback과 선택 가능한 GitHub Issues/Projects·Jira·Linear·Beads adapter |
| 외부 UI 입력 | mockup·디자인·코드·이미지·prototype의 격리 import와 출처·권위 관리 |
| Database 협업 | Git DBML 원본, validation, semantic diff, migration 영향, dbdiagram 격리 push/pull |
| TDD 개발 | 동작 변경 test-first, 좁은 예외, 재현 가능한 evidence |
| Production release | 기술 gate, immutable RC, 동일 artifact 사용자 검증, 서명·SBOM·provenance·rollback |

## 생성되는 프로젝트 구조

frontend, backend, framework, language, database, cloud directory 이름을 고정하지 않습니다. 실제 workspace 주위에 네 가지 책임 영역을 만듭니다.

```text
project-root/
├── AGENTS.md
├── .agents/skills/use-project-harness/
├── .harness/        # lifecycle, baseline, work, gate, evidence
├── specs/           # 제품 의도, 정책, scenario, 품질, architecture, UI
├── contracts/       # service, API, event, auth, error, data, DBML 의무
├── docs/            # guide, runbook, 문제 해결, 생성 요약
└── <workspaces>/    # root, directory, submodule, external 구현 단위
```

`workspace`와 `submodule`은 같은 말이 아닙니다.

- workspace: 독립적인 구현·검증·소유권·contract 경계
- submodule: 별도 repository가 필요한 workspace의 exact commit을 root에서 연결하는 Git 방식

## Harness, Skill, Plugin, CLI, Hook 차이

| 구성 | 역할 |
|---|---|
| 프로젝트 하네스 | 각 서비스의 제품 의미, contract, 현재 상태와 evidence를 repository에 보존 |
| Agent Skill | AI가 언제 질문·진단·계획·개발·context 복구·release 준비를 해야 하는지 안내 |
| Codex Plugin | Skills, optional Hook, template, CLI 연결을 GitHub marketplace로 설치·공유 |
| Go CLI | macOS·Windows에서 같은 검사·계획·생성·동기화·release gate를 결정적으로 실행 |
| Hook | 신뢰된 session 시작이나 context 압축 후 refresh 필요를 알려주는 선택 기능 |

Plugin은 편리한 Codex 배포 계층이지 프로젝트 원본이 아닙니다. 생성된 repository에는 작은 repo-local Agent Skill과 Markdown fallback이 남으므로 다른 사람이 Plugin 없이 clone해도 이어서 작업할 수 있습니다.

## Git과 협업 기본값

- 초기 개인 발견에서는 Git 없이도 사용할 수 있지만 협업에는 매우 강하게 권장하고 검증 가능한 release에는 필수입니다.
- 기본은 protected `main`과 짧은 branch이며 상시 `develop` branch를 만들지 않습니다.
- branch와 commit에는 AI 사용 흔적을 넣지 않고 일반 Git convention을 따릅니다.
- test와 구현은 checkout 가능한 검토 단위로 유지하며 깨진 red commit을 shared history에 강제하지 않습니다.
- submodule은 root가 가리키는 exact SHA를 사용하고 remote 최신 commit을 몰래 따라가지 않습니다.
- worktree는 같은 repository의 branch를 격리하고, semantic claim과 contract 검사는 worktree가 막지 못하는 의미 충돌을 처리합니다.
- 여러 repository 변경은 동시에 merge된다고 가정하지 않고 호환 가능한 순서로 병합·배포합니다.

자세한 예시는 [Git·협업·submodule 정책](./docs/design/04-git-collaboration-and-submodules.md)에 있습니다.

## 외부 도구

외부 도구는 필수 묶음이 아니라 선택 adapter입니다.

- GitHub에서 협업하면 GitHub Issues/Projects를 기본 추천합니다.
- 기존 Jira·Linear가 있으면 해당 도구를 live task-status 원본으로 유지할 수 있습니다.
- local/offline task graph가 필요하면 Beads를 선택할 수 있습니다.
- Superpowers와 BMAD는 workflow를 보완하지만 project source of truth를 소유하지 않습니다.
- Git DBML이 canonical이고 dbdiagram은 시각 협의와 격리된 동기화를 제공합니다.
- 외부 UI는 `reference`, `seed`, `canonical` 중 권위를 지정합니다.

## 현재 폴더 구조

```text
fullstack-orchestrator/
├── README.md
├── README.ko.md
└── docs/
    ├── design/               # 확정된 제품·기술 설계
    └── superpowers/plans/    # production TDD 구현 계획
```

주요 문서:

- [설계 문서 안내](./docs/design/index.md)
- [Lifecycle과 gate](./docs/design/01-project-lifecycle.md)
- [생성 프로젝트 구조](./docs/design/02-generated-project-structure.md)
- [Context와 source of truth](./docs/design/03-context-and-source-of-truth.md)
- [AI 행동·승인 정책](./docs/design/05-ai-action-and-approval-policy.md)
- [외부 adapter](./docs/design/06-external-adapters.md)
- [CLI와 result schema](./docs/design/07-checker-cli-and-result-schema.md)
- [Plugin·Skill·설치·보안](./docs/design/08-plugin-skills-installation-security.md)
- [Test·RC·production readiness](./docs/design/09-test-release-and-production-readiness.md)
- [Source repository·배포 청사진](./docs/design/10-product-repository-and-distribution.md)
- [전체 교차 검토 결과](./docs/design/11-cross-review-and-confirmation.md)
- [실제 사용자 walkthrough](./docs/design/12-user-experience-walkthrough.md)
- [Production 구현 계획](./docs/superpowers/plans/2026-07-16-fullstack-orchestrator-production.md)

## 실제 제품 구현

현재는 지원하는 설치 명령이 없습니다. 이 repository를 설치 가능한 사용자 제품처럼 안내하면 안 됩니다.

구현은 production plan의 TDD 작업 순서로 진행합니다.

1. stable result schema와 Go CLI shell
2. project schema, fingerprint, context graph
3. approval-safe operation과 실패 복구
4. Git, submodule, worktree, work, claim, conflict coordination
5. contract, DBML, dbdiagram, 외부 UI workflow
6. 신규 project 생성과 기존 project adoption
7. Agent Skills, Codex Plugin, Hook, provider adapter
8. immutable RC와 production release 검증
9. 영어·한국어 parity, example, signed package, macOS·Windows 배포

현재 작업용 command 이름은 `orchestrator`입니다. 공개 package를 하나라도 만들기 전에 제품명, repository, Plugin, package, command namespace를 조사하고 한 번 확정합니다.

## 예정 배포 방식

Production gate를 통과한 뒤 다음으로 배포합니다.

- source와 Issue: public GitHub repository
- Codex Plugin: GitHub 기반 Codex marketplace
- macOS CLI: signed GitHub artifact와 Homebrew tap
- Windows CLI: signed MSI/ZIP과 WinGet
- release evidence: checksum, signature, SBOM, provenance, compatibility matrix, rollback, support 문서

첫 공개 release는 설계 gate를 모두 충족한 production `1.0.0`입니다. AI 기술 검증과 사용자 검증은 exact same RC digest를 사용합니다.

## 보안과 개인정보

- 중앙 account나 server가 필요 없는 local-first
- telemetry 기본 off
- source code, 제품 명세, prompt, path, command log를 기본 전송하지 않음
- secret을 tracked file, evidence, prompt, diagnostic bundle에 저장하지 않음
- hidden pull·rebase·stash·reset·force push·package 설치·외부 write·production release 금지
- untrusted repository, Hook, import, task comment, provider text를 instruction이 아닌 data로 취급
- 공개 전 signed artifact, dependency review, security test, SBOM, provenance, 비공개 취약점 신고 경로 필수

## 만들지 않는 것

이 제품은 다음이 아닙니다.

- 특정 framework용 application template
- 범용 AI memory database
- Git·task manager·Figma·dbdiagram 대체품
- Superpowers·BMAD·Beads를 단순 설치하는 묶음
- 사용자 승인 없이 production을 배포하는 autopilot

핵심 범위는 제품 의도, 구현 경계, 실제 repository 상태, 협업 작업, release evidence 사이의 관계를 보존하고 검증하는 것입니다.

## License

공개 release license는 Apache License 2.0으로 확정했습니다. 실제 `LICENSE` 파일은 product source와 함께 public repository 생성 전에 추가합니다.
