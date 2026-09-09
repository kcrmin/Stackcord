# 시작 가이드

## 준비 사항

협업에는 Git을 사용하며 추적 가능한 release candidate에는 Git이 필수입니다. Plugin을 지원하는 AI client가 서비스 발견에 편리하지만 생성된 저장소에는 독립적인 Skill과 Markdown fallback도 들어갑니다. Go 1.26 이상은 source에서 직접 build할 때만 필요합니다.

## GitHub에서 설치

```bash
codex plugin marketplace add kcrmin/Stackcord --ref v1.0.0
codex plugin add stackcord@stackcord
```

설치된 snapshot에서 여섯 Stackcord Skill과 lifecycle hook을 불러오도록 설치 후 새 Codex 대화를 시작합니다.

## 검증된 release bundle 설치

현재 platform용 Plugin zip과 `checksums.txt`를 함께 내려받아 SHA-256을 확인하고 압축을 풉니다. Bundle에는 `.agents/plugins/marketplace.json`, 6개 Skill, lifecycle hook, project template, macOS·Windows bootstrap script, `distribution/platform.json`이 들어 있습니다. Platform record는 Plugin version을 맞는 CLI asset과 checksum URL에 연결합니다.

AI에게 “이 검증된 bundle을 local에 설치해줘”라고 말하면 platform record를 확인하고 checksum-first bootstrap을 사용할 수 있습니다. 압축을 푼 Plugin을 Codex CLI로 설치할 때는 해당 directory를 local marketplace로 추가하고 표시된 Plugin을 설치합니다.

```bash
codex plugin marketplace add /absolute/path/to/unpacked/stackcord
codex plugin add stackcord@stackcord
```

Bootstrap은 test용 loopback HTTP를 제외하면 HTTPS release URL만 받고 checksum과 `doctor` smoke test를 통과한 뒤 CLI를 원자적으로 교체합니다. Hook은 software를 download하거나 설치하지 않습니다.

## CLI build

제품 저장소에서 실행합니다.

```bash
cd cli
go test ./...
go build -o ../bin/stackcord ./cmd/stackcord
```

Windows PowerShell에서는 `go build -o ..\bin\stackcord.exe .\cmd\stackcord`를 사용합니다. 생성된 binary를 `PATH`에 두거나 AI에게 절대 경로를 알려줍니다. `stackcord doctor --json`으로 Git과 선택 capability를 진단할 수 있습니다. Source build는 contributor용이며 일반 사용자는 검증된 bundle을 권장합니다.

## 선택적 Plugin 설치

Source tree를 개발할 때는 이 저장소를 local marketplace로 추가하고 **Plugins** 또는 Codex CLI에서 설치합니다.

```bash
codex plugin marketplace add /absolute/path/to/stackcord
```

Codex CLI에서는 marketplace 추가 뒤 `/plugins`를 엽니다. Plugin 설치는 선택이며 생성된 프로젝트의 repo-local 동작은 유지됩니다.

## AI와 대화로 시작

빈 parent directory에서는 “새 서비스를 같이 시작해줘”, 기존 저장소에서는 “기존 파일을 덮어쓰지 말고 이 프로젝트에 도입해줘”라고 말합니다. AI는 먼저 filesystem과 Git을 확인하고 알맞은 Skill을 읽은 뒤 독립적인 일반 질문을 권장 답변과 함께 묶어 묻습니다. 권장 답변은 제출 전까지 제안이며, 민감한 결정에는 명시적인 답변이 필요합니다. 정규화 checkpoint를 저장하고 현재 섹션·남은 섹션·대략적인 질문 수를 알려줍니다. 이미 저장한 결정은 다시 묻지 않고 복구합니다.

초기화 후에는 “지금 뭐 해야 해?”, “이 기능 만들어줘”, “Contract와 DB 영향을 확인해줘”, “Production candidate 준비해줘”처럼 요청합니다. 내부 ID나 command argument를 사용자가 관리할 필요가 없어야 합니다.

## 첫 결과 확인

`README.md`, `AGENTS.md`, `.agents/skills/use-project-harness/`, `.harness/`, `specs/`, `contracts/`, `docs/`가 있는지 확인합니다. AI에게 context audit과 Git inspect를 요청합니다. Audit은 저장소 파일을 근거로 사용하고 unknown이나 stale을 지어내지 말고 그대로 알려야 합니다.

## 다음 가이드

[핵심 개념](../concepts/ko.md)을 읽고 [신규 프로젝트](../guides/new-project-ko.md) 또는 [기존 프로젝트](../guides/existing-project-ko.md)로 갑니다. 병렬 협업 전에는 [작업 관리와 작업 선점](../guides/task-management-ko.md)을 봅니다. Clone·context·Git·선택 도구 상태가 불명확하면 [문제 해결](../guides/troubleshooting-ko.md)을 사용합니다.

편집 가능한 외부 목업이나 별도 UI 저장소가 필요하면 [UI workspace와 외부 목업](../guides/ui-workspace-ko.md)에서 directory와 submodule 중 맞는 경계를 선택합니다.

## 질문 진행 상황 복구

“저장된 질문을 이어서 해줘”라고 말하면 확정된 결정, 현재 섹션, 완료·남은 섹션과 대략적인 남은 질문 수를 보여줍니다. 독립적인 일반 질문은 권장 답변과 함께 묶어서 제시하고, 민감한 질문이나 먼저 답해야 할 질문은 명시적으로 구분합니다. 권장 답변은 제출된 답변이 아닙니다.

직접 확인하려면 하네스 생성 전에는 `stackcord project discovery --draft <draft-root> --json`, 생성 후에는 `stackcord project discovery --root <project-root> --json`을 사용합니다. `--json`을 빼면 읽기 쉬운 요약을 표시하고, `--locale en` 또는 `--locale ko`로 저장된 언어를 바꿔 표시할 수 있습니다. 이 명령은 파일을 변경하지 않습니다. 진행 정보가 없는 기존 checkpoint는 진행 상황을 알 수 없다고 표시합니다.

`stackcord project checkpoint --help`에서 선택적인 `discovery` 입력 전체를 확인할 수 있습니다. 수락한 결정의 저장과 답변한 질문의 제거를 함께 처리하고 섹션별 예상 수를 갱신합니다. 하네스 생성 후 질문·결정의 원문은 `specs/product/`, 섹션 계획은 `.harness/discovery.yaml`에 있어 원래 초안 없이 clone에서도 복구됩니다. 하네스 생성 준비는 범위가 정해지고 표시된 필수 질문이 남지 않았다는 뜻이며, 정책 승인을 부여하거나 하네스를 자동 생성하지 않습니다.

## 선택형 관리 UI

`stackcord dashboard --root .`를 실행하고 출력된 로컬 URL을 여세요. 내장 브라우저 UI는 Node 런타임이나 별도 호스팅 계정이 필요 없습니다. 질문 진행, 실시간 GitHub 이슈, PR 링크, 내게 할당된 이슈와 요청된 리뷰, 설정과 진단을 보여 줍니다. 명령을 종료하면 세션도 끝나며, 닫힌 동안에는 알림을 보내지 않습니다.

`stackcord setup --json`으로 로컬 UI 선택을 확인하거나 `stackcord setup --ui enable --apply`로 저장하세요(`disable`, `ask`도 지원). 플러그인은 첫 사용 때 이 선택을 안내하며, 설치 훅이 소프트웨어를 설치하거나 창을 열지 않습니다. Codex와 Claude는 같은 프로젝트 파일과 CLI를 쓰고 호스트별 매니페스트와 훅 어댑터를 사용합니다. Claude 패키지 유효성 검사를 수행했으며, 실행 가능한 모의 검증이 모든 호스트 버전의 실제 세션 동작까지 증명하지는 않습니다.

설정은 미리보기 후 적용합니다. 공유 설정은 작업 트리의 변경 제안이 되고 개인 언어·UI 선택은 로컬에 남습니다. 다른 세션에서 이미 변경한 오래된 리비전은 거부합니다. 중단된 작업은 새 상태를 덮어쓰지 않고 영수증·잠금 기록을 남겨 확인하게 합니다. 공유 변경은 기존 기능 브랜치와 PR 절차로 커밋·검토하세요.
