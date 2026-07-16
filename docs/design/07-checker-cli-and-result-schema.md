# Cross-platform CLI, 검사기, 결과 schema 설계

> 상태: 확정
>
> 마지막 갱신: 2026-07-16
>
> 아래의 `<cli>`는 최종 제품명에 맞는 실행 파일 이름으로 치환한다. 사용자는 대개 직접 명령을 외우지 않고 AI에게 자연어로 요청하며, CLI는 AI·CI·사람이 공유하는 결정적 실행 계층이다.

## 1. 구현 기술 결정

CLI는 Go 단일 binary로 구현한다. 구현 기준 toolchain은 2026-07-16 현재 security patch가 반영된 Go 1.26.5이며, release build마다 공식 지원·보안 patch 상태를 다시 확인한다.

| 대안 | 장점 | 제외 이유 또는 위치 |
|---|---|---|
| Go | macOS·Windows cross-compile, 단일 binary, 빠른 startup, 표준 library, 배포·서명 용이 | 채택 |
| Node.js/TypeScript | Plugin·JSON tooling 친숙, ecosystem 풍부 | runtime과 package manager가 core 설치 전제에 들어가므로 제외; optional adapter에 사용 |
| Rust | 안전성과 성능, 단일 binary | 초기 개발·기여 난도가 더 높고 이 제품의 병목은 CPU가 아님 |

core는 daemon이나 중앙 server 없이 동작한다. Git도 subprocess adapter로 사용하며, 발견 단계는 Git 없이 가능하다. dbdiagram/DBML 등 optional 기능은 해당 CLI가 있을 때만 활성화한다.

초기 dependency는 Cobra v1.10.2, stable YAML v3.0.4, JSON Schema v6.0.2, Testify v1.11.1로 pin한다. YAML v4는 현재 release candidate이므로 production core에 사용하지 않고 stable release와 migration test가 준비된 뒤 재검토한다.

## 2. 사용 계층

```text
사용자의 자연어
→ Plugin/Skill 또는 Markdown fallback
→ <cli> plan/check/apply
→ project files, Git, provider adapters
→ stable JSON + 사람이 읽는 요약
```

Plugin이 정책을 재구현하지 않는다. 모든 mutation·검사 규칙은 CLI와 versioned schema에 있고 Skill은 올바른 command와 질문 흐름을 선택한다.

## 3. 명령군

### 시작과 진단

| 명령 | 기능 |
|---|---|
| `<cli> init` | 빈 공간에서 발견 draft를 만들고 승인 후 새 root·harness를 생성 |
| `<cli> adopt` | 기존 repository를 비파괴 진단하고 필요한 최소 파일만 제안 |
| `<cli> status` | 현재 lifecycle, workspace, work, stale, conflict, gate를 짧게 표시 |
| `<cli> next` | dependency와 위험을 고려한 다음 안전 작업을 추천 |
| `<cli> doctor` | OS, Git, executable, auth, path, line ending, schema 호환성 검사 |

### Context

| 명령 | 기능 |
|---|---|
| `<cli> context refresh` | 실제 상태를 읽고 index/impact/current 생성 계획을 계산 |
| `<cli> context audit` | 압축·새 작업자·불일치 때 전체 원본과 fingerprint 재검증 |
| `<cli> context pack` | 현재 작업에 필요한 최소 근거를 stable ID와 함께 출력 |

### Workspace와 Git

| 명령 | 기능 |
|---|---|
| `<cli> workspace add` | directory/root/submodule/external workspace를 계획·등록 |
| `<cli> workspace sync` | root pointer 기준 clone/update 계획과 local divergence를 진단 |
| `<cli> workspace check` | remote, branch, dirty, pointer, CI, dependency 상태 검사 |

### 작업과 협업

| 명령 | 기능 |
|---|---|
| `<cli> work plan` | 요구를 spec/contract/workspace/acceptance/dependency로 정규화 |
| `<cli> work start` | conflict preflight, claim, branch/worktree 계획, baseline 고정 |
| `<cli> work checkpoint` | 현재 진행, test, unresolved decision, actual state를 저장 |
| `<cli> work finish` | acceptance, evidence, PR/merge readiness, claim release 검사 |
| `<cli> work handoff` | 실제 담당 변경이 있을 때만 별도 책임·다음 행동을 기록 |
| `<cli> conflict check` | path와 semantic overlap, migration, pointer, merge order 검사 |
| `<cli> conflict claim` | bounded lease 생성 또는 provider에 등록 |
| `<cli> conflict release` | 완료·취소·만료 확인 후 claim 종료 |

### 의미와 contract

| 명령 | 기능 |
|---|---|
| `<cli> change plan` | 제품·architecture·contract 변경과 stale 영향을 계산 |
| `<cli> contract check` | schema, reference, compatibility, provider/consumer 검증 |
| `<cli> contract impact` | 관련 policy, scenario, workspace, test, deployment를 표시 |
| `<cli> contract evolve` | additive/versioned 전환 계획을 만들며 구현을 직접 숨겨서 바꾸지 않음 |

`evolve-contract`라는 모호한 사용 명령 대신 자연어로 “계약을 변경해”라고 요청하고 Skill이 위 명령군을 사용한다. 여기서 contract는 API 형식뿐 아니라 failure, retry, idempotency, compensation 같은 서비스 의무까지 포함한다.

### Database와 UI

| 명령 | 기능 |
|---|---|
| `<cli> db check` | DBML syntax, naming, reference, policy linkage 검사 |
| `<cli> db diff` | entity-level semantic diff, migration/rollback 영향 계산 |
| `<cli> db pull` | dbdiagram remote를 격리 scratch로 가져와 비교 |
| `<cli> db push` | 승인된 canonical DBML만 remote diagram에 동기화 |
| `<cli> ui import` | 외부 mockup을 격리하고 provenance·authority를 등록 |
| `<cli> ui check` | role/journey/state/responsive/accessibility coverage 검사 |
| `<cli> ui diff` | canonical baseline과 구현·새 source 차이를 표시 |

### 검증과 release

| 명령 | 기능 |
|---|---|
| `<cli> verify stage <id>` | lifecycle stage gate와 stale dependency 검사 |
| `<cli> verify release` | 모든 workspace, install, upgrade, security, rollback gate 실행 |
| `<cli> rc create` | exact root/workspace commit과 artifact checksum을 RC로 고정 |
| `<cli> rc verify` | RC 이후 source 변경 없이 동일 artifact를 재검증 |
| `<cli> release prepare` | tag, notes, SBOM, signature, provenance, support 자료 생성 계획 |
| `<cli> release publish` | final user approval 후 host/package channel에 immutable release 게시 |
| `<cli> migrate plan` | harness schema migration diff와 backup/rollback 표시 |
| `<cli> migrate apply` | dirty/version 검사 후 승인된 idempotent migration 실행 |
| `<cli> completion` | zsh, bash, fish, PowerShell completion 생성 |

## 4. Plan/Apply 규칙

모든 쓰기 명령은 기본적으로 plan을 먼저 반환한다.

```text
<cli> workspace add --kind submodule --path services/identity --url ...
```

기본 결과:

```text
3 files will change, one Git submodule command will run, no remote writes.
Approval class: C (repository topology change)
Operation: 01J...
Run with --apply --operation 01J... after approval.
```

- plan은 시작 state fingerprint와 expiry를 가진다.
- apply 시 state가 바뀌었으면 기존 plan을 거부하고 다시 계산한다.
- `--yes`는 B등급의 이미 승인된 plan에만 사용한다. C/D 승인 경계를 우회하지 못한다.
- AI는 현재 사용자 요청이 이미 해당 action을 명시적으로 위임했는지 approval policy로 판단한다.

## 5. Stable JSON result

모든 명령은 `--json`에서 같은 envelope를 반환한다.

```json
{
  "schema_version": "1.0",
  "tool_version": "1.0.0",
  "command": "workspace.check",
  "operation_id": "01J...",
  "status": "blocked",
  "exit_code": 4,
  "summary": "Submodule has local changes on a detached HEAD.",
  "project": {
    "id": "project.example",
    "root": "<absolute-path-redacted-when-shared>"
  },
  "facts": [],
  "warnings": [],
  "blockers": [],
  "changes": [],
  "evidence": [],
  "next_actions": [],
  "approval": {
    "required": true,
    "class": "C",
    "reason": "Choose how to preserve detached work."
  },
  "timing_ms": 42
}
```

### 상태

```text
passed warning blocked failed unknown partial approval_required
```

### Exit code

| code | 의미 |
|---:|---|
| 0 | 성공; warning은 payload에 포함하지만 command 자체는 완료 |
| 2 | 잘못된 입력 또는 config/schema |
| 3 | 승인 필요; mutation 미실행 |
| 4 | conflict 또는 precondition blocker |
| 5 | verification/test/gate 실패 |
| 6 | 외부 provider unavailable 또는 사실 확인 불가 |
| 7 | 일부 단계 성공 후 복구·재시도 필요 |
| 8 | CLI 내부 오류 |

`1`은 shell wrapper의 일반 실패와 혼동을 피하기 위해 예약한다. adapter가 낸 임의 exit code를 그대로 외부 API로 노출하지 않는다.

## 6. 사람이 읽는 출력

기본 출력은 선택한 locale로 다음 순서를 유지한다.

1. 결과 한 줄
2. 확인한 사실
3. conflict/blocker
4. 변경 예정 또는 실제 변경
5. 다음 안전 행동 하나
6. 필요할 때만 승인 질문

verbose raw logs를 기본으로 보여주지 않는다. `--verbose`도 secret redaction을 유지하며, CI annotation과 JSON에는 stable code를 포함한다.

## 7. 내부 package 구조

```text
cmd/<cli>/
internal/
├── app/              # use case orchestration
├── domain/           # project, workspace, work, gate, release model
├── policy/           # approval, conflict, TDD, source rules
├── schema/           # versioned loaders and validation
├── context/          # index, graph, fingerprint, stale propagation
├── git/              # Git/submodule/worktree adapter
├── providers/        # tasks, hosts, dbdiagram, UI adapters
├── operation/        # plan/apply, journal, recovery
├── output/           # human/JSON renderers and localization
└── security/         # trust, secret redaction, path/process safeguards
schemas/
templates/
locales/
testdata/
```

domain과 policy는 filesystem/process에 의존하지 않게 해 property test와 model test가 가능해야 한다.

## 8. Cross-platform 규칙

- path는 Go `filepath`로 처리하고 저장할 logical path는 `/` separator로 canonicalize한다.
- symlink를 필수 기능으로 사용하지 않는다.
- case-insensitive filesystem에서 충돌하는 이름을 생성하지 않는다.
- Windows reserved names, MAX_PATH 정책, executable extension, PowerShell quoting을 검사한다.
- UTF-8, LF canonicalization은 Windows checkout 설정과 무관하게 fingerprint가 같아야 한다.
- process 호출은 shell을 거치지 않고 argv array, explicit working directory, timeout, environment allowlist를 사용한다.
- file lock은 advisory 보조 수단이며 operation journal과 atomic rename이 correctness를 담당한다.

## 9. Security 경계

- root 밖으로 나가는 `..`, symlink/junction escape, unexpected mount를 차단한다.
- untrusted repository의 Hook, executable, package script를 자동 실행하지 않는다.
- Git config와 submodule URL의 unsafe protocol을 검사한다.
- stdout/stderr, JSON, evidence에서 credential을 redact한다.
- archive/import는 zip-slip, decompression limit, file type, executable bit를 검사한다.
- remote write는 target host/project/repository를 normalized ID로 재확인한다.

## 10. 성능과 cache

- 변경된 fingerprint만 재검사하고 dependency graph로 영향 범위를 줄인다.
- cache는 `.harness/local/cache/`에 두며 삭제해도 결과 정확성이 변하지 않는다.
- generated state는 input fingerprint를 포함하고 stale하면 사용하지 않는다.
- 큰 repository에서는 staged scan을 사용하되 release gate에서는 전체 검증한다.

## 11. 수용 기준

- core binary가 macOS와 Windows에 별도 runtime 없이 설치된다.
- 같은 checkout에서 두 OS의 JSON 결과와 fingerprint가 의미상 동일하다.
- AI, 사람, CI가 같은 command와 schema를 사용한다.
- mutation이 중간 실패해도 operation receipt로 안전한 재시도가 가능하다.
- 외부 tool이 없어도 status, context, spec, contract, local work 기능은 동작한다.
- 모든 자동화가 approval 등급을 우회하지 않는다.

## 12. 구현 기준 자료

- [Go release history](https://go.dev/doc/devel/release)
- [Cobra](https://github.com/spf13/cobra)
- [Go YAML v3](https://pkg.go.dev/go.yaml.in/yaml/v3)
- [JSON Schema v6 for Go](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6)
- [GoReleaser releases](https://github.com/goreleaser/goreleaser/releases)
