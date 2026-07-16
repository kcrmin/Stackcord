# 시작하기

일반 사용자는 AI에게 “새 서비스를 시작해”, “이 clone을 이어서 해”, “다음에 뭘 해야 해?”라고 말하면 됩니다. Plugin이 의도를 적절한 Skill로 연결하고 CLI가 결정적인 근거를 제공합니다. CLI를 직접 사용할 수도 있습니다.

## CLI 로컬 빌드

Git 2.40+와 Go 1.26+가 필요합니다. 특정 프레임워크·DB·클라우드·Node.js·daemon·계정·telemetry는 필요하지 않습니다.

macOS/Linux shell:

```sh
cd cli
go test ./...
go build -trimpath -o ../bin/orchestrator ./cmd/orchestrator
../bin/orchestrator doctor --json
```

Windows PowerShell:

```powershell
Set-Location cli
go test ./...
go build -trimpath -o ..\bin\orchestrator.exe .\cmd\orchestrator
..\bin\orchestrator.exe doctor --json
```

## GitHub marketplace에서 Plugin 설치

공개 owner/repository 정체성이 확정된 뒤 다음처럼 설치합니다.

```sh
codex plugin marketplace add OWNER/REPOSITORY --ref main
codex plugin add fullstack-orchestrator@fullstack-orchestrator
```

로컬 checkout을 시험할 때는 `OWNER/REPOSITORY` 대신 저장소 루트 경로를 등록합니다. 설치·업데이트 후 ChatGPT desktop app을 다시 시작하고 새 task에서 확인합니다.

## 첫 대화

“새 full-stack 서비스를 시작해”라고 말하면 AI는 `.harness-drafts/`에 정규화된 발견 결과를 계속 저장하고 중요한 질문을 한 번에 하나씩 묻습니다. 서비스 요약과 저장소 이름을 승인하기 전에는 정식 root를 만들지 않으며 기술을 미리 강제하지 않습니다.

기존 clone에서는 “이 프로젝트 이어서 해”라고 말합니다. AI는 read-only context audit으로 dirty/diverged/submodule 상태, 현재 담당 범위, stale contract, 다음 안전 행동을 알려줍니다. pull·rebase·stash·reset·pointer 이동을 숨겨서 실행하지 않습니다.

## 이 저장소 검증

```sh
cd cli && go test ./... && go vet ./...
cd .. && sh scripts/validate-plugin.sh
```

실제 공개 출시는 이름 확정, macOS/Windows native CI, 서명된 RC artifact, 같은 RC digest에 대한 사용자 확인이 끝나야 합니다.
