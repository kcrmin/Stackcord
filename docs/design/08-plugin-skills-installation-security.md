# Plugin, Skill, Hook, 설치, 보안 설계

> 상태: 확정
>
> 마지막 갱신: 2026-07-16

## 1. 네 구성 요소의 차이

| 구성 | 실제 역할 | 없을 때 |
|---|---|---|
| 프로젝트 하네스 | 각 서비스의 승인된 의미·계약·현재 상태를 repository에 보존 | 이 제품의 연속성이 사라짐 |
| Skill | AI에게 언제 어떤 자료를 읽고 어떤 질문·명령을 사용할지 알려주는 작은 playbook | AI가 Markdown fallback을 직접 따라야 함 |
| Plugin | 여러 Skill, Hook, 설정을 설치·업데이트·공유하는 Codex package | repo-local Skill과 CLI는 계속 동작 |
| CLI | macOS·Windows에서 검사·생성·동기화·release gate를 결정적으로 실행 | Markdown 기반 수동 진행만 가능하고 보장이 약해짐 |
| Hook | session start, context compression 같은 event 때 Skill 실행 필요를 알려주는 선택 trigger | 사용자가 “상태 다시 확인해”라고 요청하면 됨 |

Skill은 Codex에서만 가능한 개념이 아니다. [Agent Skills specification](https://agentskills.io/specification)을 따르는 client에서 사용할 수 있고, 그 밖의 AI는 `SKILL.md`와 `.harness/entry.md`를 일반 Markdown 지침으로 읽는다. Plugin은 Codex에 가장 자연스러운 배포 wrapper다.

## 2. 배포 형태

하나의 공개 GitHub repository에 source, Codex Plugin, marketplace catalog를 함께 둔다. CLI release binary는 GitHub Releases와 package manager로 배포한다.

```text
product-repository/
├── .codex-plugin/
│   └── plugin.json
├── .agents/
│   └── plugins/
│       └── marketplace.json
├── skills/
├── hooks/
│   └── hooks.json
├── references/
├── compatibility.json
├── cli/
├── schemas/
├── templates/
├── docs/
├── scripts/
├── LICENSE
├── SECURITY.md
└── README.md
```

사용자는 GitHub URL을 Codex marketplace source로 추가한 뒤 plugin을 설치할 수 있다. 별도 marketplace repository로 나누는 것은 여러 plugin을 운영하게 될 때만 고려한다.

## 3. Codex Plugin

`.codex-plugin/plugin.json`은 plugin name, version, description, skills path 등 현재 Codex ingestion schema가 허용하는 field만 포함한다. 그 외 파일을 `.codex-plugin/` 안에 넣지 않는다.

```json
{
  "name": "<product-name>",
  "version": "1.0.0",
  "description": "AI-guided full-stack project orchestration from discovery to release.",
  "skills": ["./skills"]
}
```

Hook manifest field는 Codex manual과 local validator version 사이에 지원 시점 차이가 있을 수 있다. 따라서 `hooks/hooks.json`을 표준 위치에 두고, release CI가 목표 Codex version의 validator에서 허용할 때만 manifest field를 추가한다. 검증되지 않은 field를 억지로 넣어 설치 자체를 깨뜨리지 않는다.

## 4. Plugin Skills

각 Skill은 하나의 뚜렷한 trigger만 담당하고, 실제 정책은 중복 복사하지 않고 versioned references와 CLI를 사용한다.

| Skill | 사용자가 하는 말의 예 | 수행 결과 |
|---|---|---|
| `start-project` | “새 서비스 시작하자” | 발견 draft, 질문 흐름, 승인된 root 생성 |
| `resume-project` | “다운받았는데 이어서 해줘” | clone/workspace/context 진단과 다음 안전 작업 |
| `find-next-work` | “지금 뭐 해야 해?” | dependency·stale·claim을 반영한 다음 일 추천 |
| `plan-project-change` | “이 기능을 추가해” | spec/contract/workspace/acceptance/TDD/충돌 계획 |
| `start-project-work` | “작업 시작해” | conflict preflight, branch/worktree, claim, failing test 준비 |
| `manage-contract-change` | “API 정책을 바꾸자” | compatibility, consumer, merge/deploy order 계획 |
| `design-project-database` | “DB 구조 같이 정하자” | 질문→DBML→검사→dbdiagram 시각 검토→변경 반영 |
| `import-project-ui` | “이 mockup으로 UI 만들어” | 격리 import, authority/provenance, journey coverage |
| `integrate-project-work` | “이제 합쳐” | workspace PR, contract checks, root pointer integration |
| `prepare-project-release` | “release 준비해” | production gate, RC, 사용자 검증, publish 승인 |
| `handoff-project-work` | “이 작업 담당을 바꿔” | 범위·actual state·미결정·다음 행동을 새 담당에게 전달 |
| `audit-project-context` | “너 내용 잊은 것 같아” | 전체 source audit, stale/unknown 복구, 원래 작업 재개 |

### Skill 작성 규칙

- directory당 `SKILL.md` 하나가 entry이며 `name`과 `description` frontmatter를 가진다.
- `description`은 trigger가 되므로 “무엇을 한다”와 “언제 사용한다”를 짧고 구체적으로 쓴다.
- 본문은 최소 절차만 두고 긴 schema·정책은 `references/`에서 필요한 것만 읽는다.
- nested reference chain을 만들지 않는다.
- 자연어 질문은 한 번에 하나씩, 추천 선택지를 먼저, `기타` 입력 가능하게 한다.
- filesystem에서 알 수 있거나 안전한 best practice는 묻지 않고 결정·기록한다.
- CLI JSON을 근거로 말하고 성공을 추측하지 않는다.
- Skill에 특정 framework·language·database 선택을 내장하지 않는다.

## 5. 생성 프로젝트의 repo-local Skill

새 프로젝트에는 plugin 전체를 복사하지 않고 작은 project-specific Skill 하나만 생성한다.

```text
.agents/skills/use-project-harness/
├── SKILL.md
└── references/
    └── fallback.md
```

이 Skill의 역할:

1. 가장 가까운 `.harness/manifest.yaml`을 찾는다.
2. `.harness/entry.md`와 현재 context fingerprint를 읽는다.
3. Plugin/CLI가 있으면 `context audit` 또는 해당 workflow로 연결한다.
4. 없으면 `fallback.md`에 있는 read-only 복구 순서를 따른다.
5. project의 실제 spec, contract, work, approval policy를 읽고 변경한다.

이 파일이 있기 때문에 다른 사람이 repository를 clone한 뒤 같은 Plugin을 아직 설치하지 않았어도 AI에게 “이 프로젝트 이어서 해”라고 말할 수 있다.

## 6. Optional Hook

기본 Hook 후보:

| event | 행동 |
|---|---|
| SessionStart | trusted repository인지 확인하고 project root와 stale context 여부만 알림 |
| PostCompact | mutation 전에 `audit-project-context`를 실행해야 함을 알림 |

Hook은 다음을 하지 않는다.

- package 설치
- file 자동 수정
- Git pull/rebase/stash/reset/push
- external write
- secret 읽기/전송
- 긴 test suite 자동 실행

정확성 강제는 CLI와 CI가 담당하고 Hook은 잊기 쉬운 진입 절차를 자동으로 떠올리는 편의 기능만 제공한다.

## 7. 설치 경험

### Codex 사용자

1. 공개 GitHub marketplace repository를 Codex에 추가한다.
2. Plugin을 설치한다.
3. signed CLI가 없으면 Plugin이 공식 설치 선택지를 보여준다.
4. repository를 열고 “이 프로젝트 이어서 해”라고 말한다.
5. AI가 doctor/context audit를 수행하고 필요한 외부 provider 인증만 요청한다.

### 다른 AI 사용자

1. signed CLI를 설치한다.
2. repository의 `.agents/skills/use-project-harness/SKILL.md` 또는 `.harness/entry.md`를 AI에게 읽게 한다.
3. “현재 상태 파악하고 다음 일 해줘”라고 요청한다.

### CLI 배포 channel

- macOS: Homebrew tap + signed universal/architecture-specific archive
- Windows: WinGet + signed MSI 또는 zip
- Linux: signed tarball과 package repository는 conformance가 확보된 후 first-class로 승격
- 모든 channel: GitHub Release checksum, signature, SBOM, provenance 제공

`curl | sh` 또는 관리자 권한 script를 기본 설치 방법으로 제시하지 않는다. manual install은 archive checksum 검증 절차를 포함한다.

## 8. Version compatibility

서로 독립적으로 versioning한다.

```text
Plugin version
CLI version
Harness schema version
Adapter API version
```

Plugin package의 root `compatibility.json`이 지원 CLI/schema/adapter range를 선언한다. `plugin.json`에는 현재 ingestion schema가 허용한 field만 둔다. CLI는 project schema가 더 새로우면 write를 거부하고 read-only diagnostic만 수행한다. 너무 오래된 Plugin은 CLI가 update 안내를 내지만 자동 update하지 않는다.

## 9. Harness migration

1. dirty worktree, unpushed commit, active operation을 검사한다.
2. source schema와 target schema, 영향 파일, 호환성, rollback을 plan으로 표시한다.
3. `.harness/local/backups/<operation-id>/`에 필요한 project-local backup을 만든다.
4. 한 version씩 순차적이고 idempotent한 migration을 적용한다.
5. schema와 semantic invariant를 검증한다.
6. generated index를 재생성하고 diff를 보여준다.
7. 성공 후에만 manifest schema version을 갱신한다.

major migration이나 의미 변경은 자동 적용하지 않는다. migration code는 golden fixture와 downgrade/restore test를 가져야 한다.

## 10. Security threat model

### 신뢰하지 않는 입력

- clone한 repository의 Hook, script, config, submodule URL
- 외부 UI archive와 generated code
- task/PR comment에 포함된 prompt injection
- dbdiagram/외부 provider에서 가져온 text
- binary, package manager lifecycle script, Git config

### 방어

- repository를 명시적으로 trust하기 전에는 read-only parse만 한다.
- Skill과 CLI는 외부 text를 instruction이 아닌 data로 취급한다.
- subprocess는 allowlist, argv, timeout, environment allowlist로 실행한다.
- root 밖 path traversal과 symlink/junction escape를 차단한다.
- plugin package와 CLI artifact에 checksum, signature, SBOM, build provenance를 제공한다.
- release CI는 dependency vulnerability, license, secret scan, static analysis를 수행한다.
- 최소 권한 token과 provider별 scope를 문서화한다.
- vulnerability disclosure는 `SECURITY.md`와 private reporting channel로 받는다.

## 11. License와 privacy

- core source, Plugin, schemas, templates는 Apache-2.0으로 공개한다.
- third-party adapter는 해당 SDK/license를 inventory에 기록한다.
- telemetry는 기본 off다.
- opt-in telemetry를 추후 추가하더라도 source code, spec content, path, prompt, command output을 수집하지 않는다.
- 중앙 account나 server 없이 local-first로 모든 core workflow를 사용할 수 있어야 한다.

## 12. Plugin 업데이트 개발 흐름

개발 중에는 source plugin directory를 수정하고 schema/Skill validator와 integration test를 실행한 뒤 cachebuster reinstall로 실제 Codex에서 검증한다. release artifact가 아닌 plugin cache를 직접 수정하지 않는다.

최종 release CI는 다음을 확인한다.

- plugin manifest schema와 marketplace entry
- 모든 Skill frontmatter, trigger uniqueness, relative reference
- Hook schema와 trusted/untrusted behavior
- missing CLI degraded flow
- old/new Plugin·CLI·harness compatibility matrix
- clean install, update, uninstall, reinstall

## 13. 수용 기준

- GitHub URL 하나로 Codex Plugin marketplace를 공유할 수 있다.
- Plugin 없이 clone해도 repo-local Skill과 Markdown fallback으로 이어서 작업할 수 있다.
- Skill이 정책을 중복 구현하지 않고 CLI/schema가 deterministic core를 제공한다.
- Hook이 실패하거나 비활성화되어도 context correctness가 유지된다.
- macOS와 Windows 사용자가 관리자 shell script 없이 signed CLI를 설치할 수 있다.
- untrusted repository가 Plugin 설치만으로 command를 자동 실행하지 못한다.
