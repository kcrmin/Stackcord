# 시작 가이드

## 준비 사항

협업에는 Git을 사용하고 CLI build에는 Go 1.26 이상을 사용합니다. 추적 가능한 release candidate에는 Git이 필수입니다. Plugin을 지원하는 AI client가 서비스 발견에 편리하지만 생성된 저장소에는 독립적인 Skill과 Markdown fallback도 들어갑니다.

## CLI build

제품 저장소에서 실행합니다.

```bash
cd cli
go test ./...
go build -o ../bin/orchestrator ./cmd/orchestrator
```

Windows PowerShell에서는 `go build -o ..\bin\orchestrator.exe .\cmd\orchestrator`를 사용합니다. 생성된 binary를 `PATH`에 두거나 AI에게 절대 경로를 알려줍니다. `orchestrator doctor --json`으로 Git과 선택 capability를 진단할 수 있습니다.

## 선택적 Plugin 설치

이 저장소를 local marketplace로 추가하고 ChatGPT desktop app을 재시작한 뒤 **Plugins**에서 설치합니다.

```bash
codex plugin marketplace add /absolute/path/to/fullstack-orchestrator
```

Codex CLI에서는 marketplace 추가 뒤 `/plugins`를 엽니다. GitHub에 둔 marketplace는 `codex plugin marketplace add owner/repo`를 사용합니다. Plugin 설치는 선택이며 생성된 프로젝트의 repo-local 동작은 유지됩니다.

## AI와 대화로 시작

빈 parent directory에서는 “새 서비스를 같이 시작해줘”, 기존 저장소에서는 “기존 파일을 덮어쓰지 말고 이 프로젝트에 도입해줘”라고 말합니다. AI는 먼저 filesystem과 Git을 확인하고 알맞은 Skill을 읽은 뒤 결과를 바꾸는 질문을 하나씩 묻습니다. 발견이 이어지는 동안 정규화 checkpoint를 계속 저장합니다.

초기화 후에는 “지금 뭐 해야 해?”, “이 기능 만들어줘”, “Contract와 DB 영향을 확인해줘”, “Production candidate 준비해줘”처럼 요청합니다. 내부 ID나 command argument를 사용자가 관리할 필요가 없어야 합니다.

## 첫 결과 확인

`README.md`, `AGENTS.md`, `.agents/skills/use-project-harness/`, `.harness/`, `specs/`, `contracts/`, `docs/`가 있는지 확인합니다. AI에게 context audit과 Git inspect를 요청합니다. Audit은 저장소 파일을 근거로 사용하고 unknown이나 stale을 지어내지 말고 그대로 알려야 합니다.

## 다음 가이드

[핵심 개념](../concepts/ko.md)을 읽고 [신규 프로젝트](../guides/new-project-ko.md) 또는 [기존 프로젝트](../guides/existing-project-ko.md)로 갑니다. Clone·context·Git·선택 도구 상태가 불명확하면 [문제 해결](../guides/troubleshooting-ko.md)을 사용합니다.
