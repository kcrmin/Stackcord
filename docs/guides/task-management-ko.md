# 작업 관리와 작업 선점

## Live status 원본 하나만 유지

담당자, workflow 상태, 팀에 보이는 진행 상황은 **live status 원본 하나**만 선택합니다. 기본값은 Git-local입니다. 팀은 인증된 connector나 실제 CLI로 읽고 쓸 수 있을 때만 GitHub Issues, Jira, Beads 또는 기존 provider를 선택할 수 있습니다. 선택 결과는 `.harness/work/provider.yaml`에 기록하며 사용할 수 없는 adapter가 동작한 것처럼 꾸미지 않습니다.

저장소의 work definition은 결과, 성공·실패 acceptance scenario, 영향받는 workspace, 의미 범위, dependency, merge 순서, 첫 실패 test, 필요한 evidence를 담는 오래가는 실행 checklist입니다. 이것은 두 번째 task board가 아닙니다. 선택한 provider가 live status를 소유하고 저장소는 제품 의미와 구현 경계를 소유합니다.

이 장치는 coordination 또는 복구 가치가 있을 때만 사용합니다. 작은 개인 local edit는 hosted issue나 Git 선점 없이 진행할 수 있습니다. 공유 ownership, 긴 중단, 여러 workspace, 서비스 규칙·contract 변경, migration, 공유 UI flow, 병렬 충돌 가능성이 있으면 오래가는 checklist와 선점 흐름을 사용합니다. 작업 크기와 무관하게 merge 전에는 해당 동작 test와 review가 필요합니다.

## 자연어 흐름으로 사용

기여자는 “계정 복구 기능 만들어줘” 또는 “지금 뭐 해야 해?”라고 말합니다. AI는 사용자에게 내부 ID를 관리시키지 않고 다음 흐름을 수행합니다.

1. 저장소, 선택 provider, Git remote, branch, worktree, submodule, contract, 기존 결정을 검사합니다.
2. 결과를 바꾸는 제품 결정만 확인합니다. 조정이 필요한 작업은 `stackcord work define`으로 정규화된 실행 checklist를 저장하고 작은 개인 변경은 일반 change와 test 안에서 계획합니다.
3. 외부 provider를 선택했다면 고른 connector로 팀에 보이는 issue를 만들거나 갱신합니다. Issue 연결을 기록하고 작업자를 배정하며 매핑된 `in_progress` 상태로 옮깁니다.
4. 정확한 issue를 다시 읽습니다. Item ID, revision, status, owner, dependency, capability, work fingerprint, 조회 시각, payload hash를 정규화한 뒤 `stackcord work provider reconcile --apply`를 실행합니다.
5. 조정이 필요한 작업은 `stackcord work start --apply`를 실행합니다. CLI는 path·policy·scenario·contract·DB entity·migration·UI flow·dependency·submodule pointer 범위를 비교하고 Git coordination branch에 compare-and-swap을 수행합니다. 이 선점이 성공한 뒤에만 AI가 일반 branch나 worktree를 만듭니다.
6. Checklist와 TDD로 개발합니다. 상태나 담당자를 바꾸기 전 evidence를 먼저 검증하고, connector로 외부 provider를 갱신한 뒤 새 revision을 다시 읽고 reconcile합니다. 마지막으로 `stackcord work transition --apply` 또는 `stackcord work handoff --apply`로 동기화합니다.

Issue 담당자 지정 자체는 배타적 lock이 아닙니다. GitHub는 여러 assignee를 표현할 수 있고 issue system은 서비스 contract나 DB 의미를 이해하지 못합니다. Git compare-and-swap 선점이 두 번째 live status 원본이 되지 않으면서 여러 저장소의 의미 범위를 배타적으로 보호합니다.

## 상황에 맞는 provider 선택

| Provider | 적합한 상황 | Trade-off |
| --- | --- | --- |
| Git-local | 새 프로젝트, 작은 팀, offline 또는 provider-neutral 흐름 | Hosted board가 없으며 팀 단위 안전한 선점에는 Git remote가 필요함 |
| GitHub Issues | 이미 GitHub에서 review와 계획을 하는 팀 | 실제 인증 GitHub connector가 필요하고 assignee 상태만으로는 advisory임 |
| Jira | 기존 workflow·권한·reporting을 사용하는 팀 | 선택한 Atlassian 연동 같은 실제 Jira connector가 필요하며 field·status mapping을 명시해야 함 |
| Beads | Git 친화적 분산 task graph를 의도적으로 선택한 팀 | 감지된 Beads CLI와 별도 운영 model이 필요하며 기본 bundle이 아님 |
| 기존 provider | 이미 다른 working system이 있는 저장소 | Live read/write, ownership, dependency mapping, revision evidence를 관찰할 수 있을 때만 선택 |

GitHub Issues를 고정 기본값으로 두지 않으며 문서에 Jira나 Beads 이름이 있다는 이유로 활성화하지 않습니다. AI는 설치된 connector와 CLI를 먼저 감지하고, 선택이 필요한 시점의 공식 유지보수·보안 정보를 확인하며, 현실적인 후보 두세 개와 trade-off를 설명한 뒤 사용자가 고른 것만 연결합니다.

## 저장소 Git convention 유지

Branch, commit, PR, issue convention을 Stackcord에 한 번 알려줍니다. `use-git-conventions` Skill은 먼저 `CONTRIBUTING.md`, `AGENTS.md`, 기존 PR·issue template을 확인합니다. 규칙이 일치하면 정규화한 내용만 `.harness/git-conventions.yaml`에 저장하고 이후 Git 작업 전에 다시 사용합니다.

```yaml
schema_version: 1
branch:
  format: "{type}/{issue}-{description}"
  types: [feat, fix]
commit:
  title_format: "{type}({scope}): {subject}"
  types: [feat, fix]
  scopes: [api, ui]
  max_length: 72
pull_request:
  title_format: "[{issue}] {title}"
  required_sections: [Summary, Test plan]
issue:
  title_format: "[{type}] {title}"
  required_sections: [Problem, Acceptance]
```

이 파일은 선택 사항입니다. 파일이 없으면 기존 Stackcord branch 동작을 유지하며 AI가 commit·PR·issue 규칙을 임의로 만들지 않습니다. 저장소 convention은 안전한 Git ref 검사를 완화하거나 issue·PR 외부 쓰기를 승인하지 않습니다. 기여 문서끼리 충돌하면 AI가 충돌 내용을 알리고 어떤 규칙을 적용할지 묻습니다.

## 저장되는 내용 이해

- `.harness/work/definitions/<work-id>.yaml`은 commit되는 실행 checklist와 의미 범위입니다.
- `.harness/work/mappings/<work-id>.yaml`은 선택한 외부 item과의 안정적인 관계이며 commit됩니다.
- `.harness/local/providers/<provider>/<work-id>.yaml`은 Git에서 제외되는 짧은 수명의 정규화 관찰입니다. Canonical이 아니며 stale 또는 cache 복사본은 변경을 승인할 수 없습니다.
- Remote coordination branch에는 Git compare-and-swap으로 갱신하는 작고 기한이 있는 의미 선점이 들어갑니다. 사용자가 직접 편집하지 않습니다.
- 선택한 외부 issue에는 사람에게 보이는 assignee, workflow status, 팀 대화가 들어갑니다.

Provider 원본 payload, token, 원본 대화, 사용자 말투는 commit하지 않습니다.

## Clone 또는 중단 뒤 복구

Clone은 work definition, mapping, contract, 결정, workspace topology, remote Git 선점을 복구합니다. Git에서 제외한 provider 관찰은 의도적으로 복구하지 않습니다. 따라서 AI는 누가 어떤 branch와 의미를 선점했는지 복구하면서 외부 status는 선택한 connector로 다시 읽기 전까지 unknown이라고 정확히 표시합니다. Cache snapshot이 존재한다는 이유만으로 live truth가 되지 않습니다.

기여자가 token을 모두 쓰거나 컴퓨터를 바꾸거나 나중에 돌아오면 “이 프로젝트 이어서 해”라고 말합니다. AI는 저장소 evidence를 audit하고 provider를 새로 읽으며 confirmed, stale, unknown, blocked, local-only 상태를 설명한 뒤 dependency-ready 다음 행동 하나를 권합니다.

## 상태를 꾸미지 않고 실패 처리

Provider를 사용할 수 없으면 AI는 unknown이라고 보고하고 재연결 또는 단일 provider를 바꾸는 명시적 결정을 제안합니다. Status를 몰래 Git-local로 복사하지 않습니다. 외부 provider 갱신은 성공했지만 Git 선점 compare-and-swap race에서 지면 branch 작업을 시작하지 않고 ownership과 충돌 범위를 새로 읽습니다. 외부 status와 의미 coordination이 다르면 정확한 revision이 맞을 때까지 integration과 release를 차단합니다.

Branch, commit, pull request는 `feature/account-recovery`, `feat(account): add recovery challenge`, “Add account recovery flow” 같은 팀의 저장된 convention을 사용합니다. AI, agent, model, tool 이름을 넣지 않습니다.

## 실시간 GitHub 이슈

`stackcord issues list --repo owner/repository --json`은 실시간 이슈를 읽습니다. `stackcord issues create --repo owner/repository --id work-example --title TITLE --body-file issue.md`로 생성을 미리 보고 `--apply`를 추가해 생성하세요. 안정적인 ID는 이전 이슈가 닫혀 있어도 재시도 중복을 막습니다. UI에서도 같은 미리보기·생성 흐름을 제공합니다.

기존 작업 정의는 `{"work.example": 42}` 같은 JSON 매핑을 `stackcord issues migrate --root . --repo owner/repository --mapping mapping.json`에 전달하고 검토 후 `--apply`로 적용하세요. 정의·의존 관계·인수 기준 참조는 저장소에 유지하고 GitHub가 실시간 상태를 관리합니다. 전환은 연결된 이슈와 의존 관계를 확인한 뒤 로컬 매핑을 바꿉니다. GitHub를 선택한 작업 흐름은 인증된 실시간 이슈 데이터를 새로 읽으며, 복사한 스냅샷을 실시간 원본으로 인정하지 않습니다. 다른 프로젝트는 선택한 제공자를 유지합니다.
