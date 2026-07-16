# AI 행동, 승인, 안전 정책

> 상태: 확정
>
> 마지막 갱신: 2026-07-16

## 1. 목표

사용자가 Git과 하네스 명령을 일일이 지시하지 않아도 AI가 현재 상태를 진단하고 다음 행동을 수행하게 한다. 동시에 shared state, destructive action, production, secret에 관한 권한을 자연어 요청보다 넓게 추론하지 않는다.

핵심 원칙은 다음과 같다.

- 사용자는 “지금 뭐 해야 해?”, “이 기능 시작해”, “release 준비해”라고 말할 수 있다.
- AI는 필요한 read-only 진단과 안전한 project-local 작업을 묶어 수행한다.
- 같은 작업 범위에서 이미 위임받은 행동을 매 단계마다 다시 묻지 않는다.
- 외부 공유 상태, 구조 변경, destructive action, production에는 분명한 승인 경계를 둔다.
- 자동화 여부와 관계없이 모든 mutation은 계획, 결과, 실패 복구 지점을 남긴다.

## 2. 행동 등급

| 등급 | 예시 | 기본 행동 |
|---|---|---|
| A 관찰 | 파일·Git·CI·task 상태 읽기, schema 검사, diff 계산, 다음 일 추천 | 자동 실행하고 결과만 요약 |
| B 안전한 local 변경 | 사용자가 요청한 기능 코드·test·문서 수정, generated index 갱신, local work record 작성 | 범위와 주요 위험을 알린 뒤 실행; 반복 승인 없음 |
| C shared·구조·외부 변경 | Git init/remote 설정, submodule 추가·변환, dependency major 변경, 외부 Issue 작성, dbdiagram push, plugin 설치, push/PR, RC 고정 | 현재 요청이 해당 행동을 명시적으로 위임하지 않았다면 실행 전 승인 |
| D destructive·production | file/data 삭제, history rewrite, force push, production deploy, final release publish, irreversible migration, secret 외부 전송 | 정확한 target·영향·rollback을 보여주고 항상 직전 승인 |

사용자가 “구현하고 PR까지 올려”라고 명시하면 그 작업의 B·push·PR 범위는 승인된 것이다. 그 말이 production deploy, final release, unrelated issue 정리까지 승인한 것은 아니다.

## 3. Standing consent

긴 작업에서 질문 반복을 막기 위해 하나의 task에 한정된 standing consent를 기록할 수 있다.

```yaml
operation_scope:
  objective: implement account recovery
  repositories: [root, workspace.identity, workspace.web]
  allowed_actions: [edit, test, commit, push, draft_pr]
  excluded_actions: [production_deploy, final_release, history_rewrite]
  expires_on: task_completion
```

- consent는 현재 요청의 목적, repository, action, 만료 조건으로 제한한다.
- 새로운 repository, production, destructive action으로 확장할 수 없다.
- 중요한 전제가 바뀌거나 사용자의 새 요청이 기존 목적을 대체하면 다시 확인한다.
- `.harness`에는 credential이나 원문 대화를 저장하지 않고 승인 범위와 operation ID만 기록한다.

## 4. 자연어 사용 흐름

### “이 프로젝트에서 뭐 해야 해?”

AI는 A등급 진단을 수행하고 다음처럼 답한다.

```text
현재 contract baseline까지 승인되었습니다.
workspace.web가 root pointer와 다르고 local 변경은 없습니다.
다음 작업은 recovery journey의 frontend connection이며 기존 claim과 겹치지 않습니다.
정확한 pointer로 동기화하고 test-first 작업을 시작하겠습니다.
```

fast-forward sync처럼 working tree를 바꾸는 순간에는 변경 대상만 알리고, 현재 요청에 진행 위임이 있다면 이어서 수행한다. local dirty나 history divergence가 있으면 자동 해결하지 않고 선택을 요청한다.

### “로그인 기능 개발해”

AI는 관련 spec·scenario·contract가 충분한지 확인하고 conflict preflight를 수행한다. 빠진 제품 의미가 결과를 크게 바꾸면 구현 전에 질문하고, 명백한 기술 세부는 현재 architecture와 convention에 맞게 결정해 기록한다. TDD evidence를 남기며 branch/commit 이름에는 AI 표식을 넣지 않는다.

### “release 해”

AI는 production gate, RC, 설치·업그레이드·보안·rollback 검증을 먼저 수행한다. 기술 RC는 자동으로 엄격하게 검사하지만 final publish 직전에는 동일 SHA와 artifact, 사용자 검증 결과, rollback을 보여주고 D등급 승인을 받는다.

## 5. Git 안전 규칙

- status, log, diff, fetch, remote 조회는 자동 수행 가능하다.
- pull/rebase/stash/reset/clean/force push를 숨어서 실행하지 않는다.
- dirty file이 있으면 먼저 사용자의 변경과 현재 AI 변경을 구분한다.
- 현재 task와 무관한 사용자 변경은 보존하고 stage/commit하지 않는다.
- shared branch history를 rewrite하지 않는다.
- submodule detached HEAD의 local change를 발견하면 자동 checkout/update하지 않는다.
- commit, push, PR의 실제 대상 repository·branch·remote를 직전에 재검증한다.
- final release tag와 artifact publish는 별도 승인 대상이다.

## 6. 외부 명령과 도구

- 실행 파일은 allowlisted adapter를 통해 argv array로 호출한다. shell string interpolation을 사용하지 않는다.
- tool version, working directory, timeout, expected output, network 필요 여부를 plan에 포함한다.
- 누락된 tool을 자동 설치하지 않는다. 설치 출처, version, 권한을 보여주고 승인받는다.
- 외부 도구가 파일을 직접 수정한다면 격리 directory에서 실행하고 semantic diff 후 적용한다.
- plugin이나 hook이 있다는 이유로 untrusted repository의 command를 자동 실행하지 않는다.

## 7. Secret과 privacy

- token, password, private key는 tracked file, prompt transcript, raw log, evidence에 저장하지 않는다.
- environment variable 또는 OS credential store reference만 저장한다.
- command output은 저장 전에 secret pattern과 provider-specific field를 redact한다.
- 외부 AI·service로 전송할 파일과 data 범위를 사용자에게 설명하고 최소화한다.
- telemetry는 기본 off이며 중앙 server가 없어도 전체 기능이 동작한다.
- 사용자가 opt-in하지 않으면 source code, product spec, command log를 제품 개발자에게 전송하지 않는다.

## 8. Atomic operation과 복구

모든 mutation command는 다음 lifecycle을 따른다.

1. precondition과 권한을 읽는다.
2. 실행 plan과 예상 diff를 만든다.
3. operation ID와 시작 state fingerprint를 기록한다.
4. 임시 파일과 atomic rename으로 project-local file을 변경한다.
5. 각 external step 뒤에 결과를 journal한다.
6. validation 후 완료 receipt를 남긴다.
7. 중간 실패면 성공한 단계, 실패한 단계, 안전한 재시도·수동 복구를 표시한다.

재시도는 같은 operation ID에 대해 idempotent해야 한다. Git push, Issue 생성, dbdiagram push처럼 중복 위험이 있는 외부 작업은 remote receipt를 확인한 뒤 재시도한다.

## 9. AI가 컨텍스트를 잊었을 때

다음 신호를 스스로 감지하거나 사용자가 지적하면 `audit-project-context` 절차를 즉시 수행한다.

- 이미 승인한 결정을 다시 질문한다.
- 현재 branch, work item, workspace pointer를 추측한다.
- 존재하는 spec/contract를 찾지 않고 새로 정의하려 한다.
- repository 파일보다 대화 기억을 우선한다.
- “아마”, “기억상”을 근거로 mutation하려 한다.

audit가 끝날 때까지 B~D 변경을 중단한다. 사실, 불일치, stale, unknown, 안전한 다음 행동을 짧게 보고한 뒤 원래 작업을 계속한다. 이를 위해 repo-local Skill, Plugin Skill, CLI command, Markdown fallback이 같은 규칙을 공유한다.

## 10. Hook 정책

Hook은 event 발생 시 작은 검사나 안내를 자동 실행하는 연결점이다. 예를 들어 SessionStart에서 root를 찾고 PostCompact에서 context refresh를 요구할 수 있다.

- Hook은 opt-in이며 trusted project에서만 활성화한다.
- Hook은 destructive command, package 설치, push, 외부 write를 수행하지 않는다.
- Hook 실패가 project를 사용할 수 없게 해서는 안 된다.
- enforcement는 Hook이 아니라 CLI, CI, branch protection이 담당한다.
- Hook이 지원되지 않는 AI client에서는 동일 Skill을 수동으로 호출한다.

## 11. 승인 표현

승인이 필요할 때는 기술 명령을 묻지 않고 영향 중심으로 짧게 묻는다.

```text
이 변경은 identity repository에 새 remote branch를 push하고 Draft PR을 만듭니다.
production에는 배포하지 않으며 main은 직접 수정하지 않습니다. 진행할까요?
```

D등급은 target을 더 정확히 고정한다.

```text
검증한 RC commit abc123과 artifact checksum xyz를 production release 1.0.0으로 공개합니다.
database migration은 additive이며 이전 artifact로 rollback할 수 있습니다. 최종 공개할까요?
```

객관식 발견 질문에는 기본적으로 추천 선택지를 첫 번째에 두고 `기타` 자유 입력을 항상 제공한다. 그러나 filesystem에서 확인할 수 있는 사실이나 best practice로 정할 수 있는 사소한 선택은 묻지 않는다.

## 12. 수용 기준

- 사용자는 내부 command를 몰라도 자연어만으로 상태 진단과 다음 작업을 진행할 수 있다.
- 하나의 승인된 task 안에서 불필요한 승인 질문이 반복되지 않는다.
- AI가 production, destructive, secret 행동을 암묵적으로 확대하지 않는다.
- 실패한 외부 mutation을 중복 생성 없이 복구할 수 있다.
- Plugin/Hook 없이도 같은 승인 경계가 유지된다.
