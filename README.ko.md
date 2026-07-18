# 풀스택 프로젝트 하네스

> 현재 패키지용 작업 이름입니다. 공개 제품 이름은 실제 배포 전에 결정합니다.

[English](./README.md)

사용자가 AI와 서비스를 자세히 정의하고, framework를 미리 강제하지 않은 채 풀스택 저장소를 만들거나 기존 저장소에 도입하며, 여러 사람·clone·submodule·worktree·AI context 압축을 거쳐서도 안전하게 개발을 이어가게 하는 제품입니다.

운영 원칙은 단순합니다. **대화와 판단은 Skill이, 실제 상태·동일성·안전·충돌 검증은 Go CLI가 담당합니다.** 사용자는 명령을 외우기보다 AI에게 자연어로 말합니다.

## 실제 사용 모습

AI에게 다음처럼 말하면 됩니다.

- “새 서비스를 같이 시작해줘.”
- “이 프로젝트 이어서 해. 지금 뭐 해야 해?”
- “계정 복구 기능 만들어줘.”
- “DB 다이어그램이 바뀌었어. 이유를 확인하고 migration을 계획해줘.”
- “프로젝트 context를 복구하고 production candidate를 준비해줘.”

AI는 알맞은 Skill을 읽고 저장소와 실제 상태를 검사합니다. 결과를 크게 바꾸는 질문만 하나씩 묻고, 중요한 답변 뒤에는 정규화한 제품 지식을 저장합니다. 원본 대화나 사용자의 말투는 저장하지 않습니다.

## 제공 기능

- 긴 서비스 발견과 revision checkpoint: 제품 요약, 역할, journey, 정책, scenario, 품질, UI coverage, 결정, 가정, 기술 요구, 미해결 질문을 계속 정리합니다.
- Framework-neutral 신규 프로젝트 생성과 기존 저장소 비파괴 도입.
- Repo-local 지침, stable ID, fingerprint, context index, stale 감지를 이용한 clone 후 복구. Plugin이 없어도 가능합니다.
- 실제 Git 진단: branch, dirty, upstream, ahead/behind/diverged, worktree, 관례적인 branch 계획.
- 정확한 submodule 진단: root가 기록한 pointer, checkout된 HEAD, 누락, dirty, 불일치, 안전한 초기화 계획.
- 작업 전 충돌 검사: path뿐 아니라 정책, scenario, contract, DB entity, migration, UI flow, dependency major, root pointer의 의미 충돌까지 확인합니다.
- 조정이 필요한 변경에만 쓰는 기한 있는 작업 선점, 실제 소유권을 넘길 때의 handoff, 호환성 우선 통합 순서, TDD evidence. 작은 개인 변경에는 issue나 선점을 강제하지 않습니다.
- Git-local·GitHub·Jira·Beads 또는 관찰 가능한 provider 중 선택한 task-status 원본 하나를 가짜 native adapter 없이 배타적인 Git compare-and-swap 의미 선점에 연결.
- 제품 정책·실패 동작·contract·DBML·migration·외부 UI mockup·dbdiagram 협업 흐름.
- 기술 검증과 사용자 검증이 똑같은 digest를 가리켜야 하는 production candidate.
- SBOM, provenance, signature, publication receipt가 필요한 조직을 위한 선택적 strict-release profile.

## 구조

| 계층 | 역할 |
| --- | --- |
| 사용자에게 보이는 5개 Skill | 자연어 의도 이해, 제품 발견, 적절한 시점의 외부 도구·기술 추천, 결과 설명 |
| Cross-platform Go CLI | Git·submodule·fingerprint·충돌·contract·DBML·UI import·통합·release 동일성을 결정적으로 검증 |
| 저장소가 소유하는 원본 | 다른 사람이나 AI가 clone 또는 context 압축 뒤에도 복구하도록 정규화된 결정과 상태를 보존 |

5개 Skill은 다음과 같은 안정적인 package 이름과 겹치지 않는 역할을 가집니다.

1. `start-project`: 프로젝트 시작 또는 기존 프로젝트 도입
2. `continue-project`: 프로젝트 이어가기와 다음 작업 선택
3. `plan-project-work`: 변경을 계획하고 조정이 필요할 때 등록·선점·작업 시작
4. `coordinate-project-work`: Contract·DBML·UI·소유권·통합·충돌 조정
5. `recover-and-release-project`: Context 복구·production 강화·release 준비와 검증

## 생성되는 프로젝트 구조

```text
project/
├── README.md
├── AGENTS.md
├── .agents/skills/use-project-harness/
│   ├── SKILL.md
│   └── references/fallback.md
├── .harness/
│   ├── entry.md
│   ├── manifest.yaml
│   ├── profile.yaml
│   ├── sources.yaml
│   ├── workspaces.yaml
│   └── work/provider.yaml
├── specs/index.md
├── contracts/registry.yaml
└── docs/index.md
```

`specs/`는 제품 의미와 정책, `contracts/`는 component 사이 의무와 실패 동작을 소유합니다. `.harness/`는 작고 기계가 읽을 수 있는 조정 상태입니다. 선택한 task source는 `.harness/work/provider.yaml`에 기록합니다. 사용자가 `.harness/`를 직접 다룰 일은 거의 없으며 AI가 필요한 의미만 요약하고 수정합니다. `contracts/registry.yaml`은 각 규약을 원본과 dependent에 연결하고 Plugin 없는 복구는 `.agents/skills/use-project-harness/`에서 시작합니다.

처음 writable context audit을 실행하면 `context-index.json`과 `impact-graph.json`을 Git에서 제외되는 `.harness/local/context/` 아래에 다시 만듭니다. 이 파일은 local cache이며 clone 복구 evidence나 초기 tracked 구조가 아닙니다.

## 개발 흐름

Waterfall이 아니라 다음 순서를 반복합니다.

1. 저장소와 사용 가능한 도구를 진단합니다.
2. 서비스를 발견하며 중요한 답변을 계속 checkpoint합니다.
3. 기술을 성급히 정하지 않고 새 프로젝트를 초기화하거나 기존 프로젝트에 도입합니다.
4. 제품 전체 의미와 UI coverage를 세운 뒤 역할·도메인·journey 단위로 나눕니다.
5. 병렬 구현의 모호함을 줄이는 공유 경계·contract·DBML을 먼저 합의합니다.
6. 작은 수직 변경을 TDD로 만들고 계속 통합합니다.
7. Provider를 consumer보다 먼저 통합하고 child commit이 검토 가능해진 뒤 root submodule pointer를 갱신합니다.
8. Production을 강화하고 하나의 candidate를 만든 뒤 동일 digest를 기술 검증과 사용자 검증에 사용하여 release·운영으로 갑니다.

기술은 제품 기능·품질·팀·운영 조건이 드러난 뒤 선택합니다. 실제 선택 시점에는 AI가 공식 유지보수·보안·release 상태를 다시 확인해야 합니다.

## Git·submodule·worktree

Git은 협업에 매우 강하게 권장하며 검증 가능한 release에는 필수입니다. Branch와 commit은 `feature/account-recovery`, `feat(account): add recovery challenge` 같은 일반 convention을 사용하고 AI 표시는 넣지 않습니다.

작업 전 CLI가 local/upstream 상태와 active 작업 선점을 비교합니다. 동시에 여러 branch가 필요하면 worktree로 격리할 수 있습니다. Multi-repo 프로젝트에서 각 child workspace는 자기 저장소에서 commit·review하고, root 저장소는 수용한 정확한 child commit을 기록합니다. Root pointer는 매 local commit마다가 아니라 호환 가능한 child 작업이 준비된 뒤 통합합니다.

GitHub Issues, Jira, Beads를 선택하면 해당 도구만 live assignee와 status 원본이 됩니다. AI가 실제 설치된 connector로 갱신하고 정확한 관찰 revision을 reconcile한 뒤 CLI가 별도의 Git coordination branch로 서비스 의미를 선점합니다. 이것은 두 번째 task board가 아니라 같은 policy·contract·DB entity·UI flow·migration slot·dependency boundary·submodule pointer를 두 사람이 동시에 바꾸는 것을 막는 장치입니다. 자세한 흐름은 [작업 관리와 작업 선점](./docs/guides/task-management-ko.md)에 있습니다.

충돌을 발견하면 AI는 의미가 겹치는 지점을 설명하고 소유권 분리, contract 선합의, provider/consumer 순차 통합, 공유 경계 먼저 merge, 의도적 직렬화 중 현실적인 방법을 권합니다. Dirty tree, divergence, detached submodule, 공개되지 않은 child commit을 파괴적으로 자동 복구하지 않습니다.

## DBML·dbdiagram·외부 UI

Git에 추적되는 DBML이 원본입니다. dbdiagram은 격리된 시각화와 semantic diff 공간이며 remote 변경을 자동으로 원본에 올리지 않습니다. AI가 변경 이유를 묻고 entity 단위 차이를 보여준 뒤 수용한 변경을 contract와 migration에 연결합니다.

외부 mockup은 quarantine에 가져온 뒤 `reference`, `seed`, `canonical` 중 하나로 등록합니다. License, 출처, 크기, 내용을 검사하기 전에는 제품 파일을 바꾸지 않습니다.

## 기본 mode와 strict release

기본 mode는 저장소 identity, artifact fingerprint, TDD evidence, integration evidence, 해당되는 migration/rollback evidence, 정확한 candidate digest에 연결된 사용자 확인을 요구합니다. 공개 작업은 하지 않습니다.

Strict release는 [`profiles/strict-release`](./profiles/strict-release/README.md)의 선택 profile입니다. SBOM·provenance·signature·조직 gate를 추가하지만 평범한 프로젝트 흐름에는 강제하지 않습니다. 공개 계정 생성, signing identity, 되돌릴 수 없는 배포, package channel 소유권은 자동 local 흐름 밖에 둡니다.

## Build와 test

Go 1.26 이상이 필요합니다.

```bash
cd cli
go test ./...
go build -o ../bin/orchestrator ./cmd/orchestrator
```

Windows PowerShell에서는 다음과 같습니다.

```powershell
cd cli
go test ./...
go build -o ..\bin\orchestrator.exe .\cmd\orchestrator
```

`orchestrator doctor --json`으로 local capability를 확인할 수 있습니다. 일반적으로 AI가 Skill을 통해 CLI를 사용하고, 직접 확인할 때는 `orchestrator --help`를 사용합니다.

## Plugin 설치와 공유

Plugin은 선택 사항입니다. 생성된 저장소는 repo-local Skill과 Markdown fallback만으로도 이어갈 수 있습니다.

Local 개발에서는 이 저장소를 marketplace source로 추가하고 desktop app을 재시작한 뒤 **Plugins**에서 설치합니다. Codex CLI에서는 marketplace 추가 후 `/plugins`를 엽니다.

```bash
codex plugin marketplace add /absolute/path/to/fullstack-orchestrator
```

GitHub로 배포할 때는 저장소를 공개한 뒤 `codex plugin marketplace add owner/repo`를 사용할 수 있습니다. 팀 저장소의 `.agents/plugins/marketplace.json`으로 공유하거나 ChatGPT workspace 안에서 설치한 local Plugin을 공유할 수도 있습니다. 자세한 local 절차는 [시작 가이드](./docs/getting-started/ko.md)에 있습니다.

## 문서

- [시작 가이드](./docs/getting-started/ko.md)
- [핵심 개념](./docs/concepts/ko.md)
- [신규 프로젝트](./docs/guides/new-project-ko.md)
- [기존 프로젝트](./docs/guides/existing-project-ko.md)
- [Submodule과 협업](./docs/guides/submodules-ko.md)
- [작업 관리와 작업 선점](./docs/guides/task-management-ko.md)
- [DBML과 dbdiagram](./docs/guides/dbdiagram-ko.md)
- [Release](./docs/guides/release-ko.md)
- [문제 해결](./docs/guides/troubleshooting-ko.md)
- [집중된 설계](./docs/design/index.md)

## 하지 않는 것

Framework generator나 범용 프로젝트 관리 플랫폼이 아닙니다. Superpowers·BMAD·Beads·GitHub Issues·Jira·Linear를 제품의 source of truth로 만드는 묶음도 아닙니다. 외부 도구는 실제 상태를 감지하거나 trade-off와 함께 제시하며, 사용자가 선택한 live task-status 원본 하나만 연결합니다.

## 차별점

Superpowers는 agent의 brainstorming·test·debug·review 방법을 강화하고 BMAD는 formal planning 역할을 더할 수 있습니다. Issue tracker는 팀 진행 상황을 보여주고 Memory 도구는 대화 recall을 도울 수 있습니다. 하지만 어느 하나만으로 frontend clone, backend submodule, 비즈니스 규칙, DB migration, UI flow, task owner, release candidate가 여전히 같은 서비스 상태를 가리키는지 증명하지는 못합니다.

이 제품은 정규화 제품 발견, 서비스 규약, root·child Git identity, 의미 작업 선점, 외부 provider reconciliation, commit-bound TDD evidence, 정확한 사용자 검증 release identity를 연결합니다. 실행 가능한 [dogfood report](./dogfood/report.md)는 현재 선언한 결정적 시나리오 9/9와 assertion 23/23을 통과합니다. 사람의 생산성이나 hosted provider 신뢰성 수치는 주장하지 않습니다.

## 외부 공개 전 결정

외부 결정 없이 local 구현과 검증은 끝낼 수 있습니다. 실제 공개에는 최종 제품 이름과 identifier, 공개 저장소/account, strict artifact를 약속할 경우 signing 소유권, 되돌릴 수 없는 release 실행 승인이 필요합니다.
