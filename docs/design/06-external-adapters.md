# 외부 도구와 provider adapter 설계

> 상태: 확정
>
> 마지막 갱신: 2026-07-16
>
> 외부 도구는 제품의 핵심 상태를 대신 소유하지 않는다. 사용자가 이미 쓰는 도구를 연결하고, 없을 때도 local fallback으로 같은 개발 흐름을 유지한다.

## 1. 기본 결정

- Git 자체는 협업·release에 강하게 권장하지만 Git host는 고정하지 않는다.
- task provider는 프로젝트마다 하나의 live status source를 선택한다. 기본 GitHub 프로젝트에는 GitHub Issues/Projects를 추천하되 강제하지 않는다.
- Superpowers와 BMAD는 작업 상태 저장소가 아니라 workflow/방법론이다. 감지하고 함께 쓸 수 있지만 core dependency로 넣지 않는다.
- Beads는 local/offline task graph가 필요한 팀의 선택 adapter다.
- DBML file in Git이 database schema의 canonical source다. dbdiagram은 협의·시각화·양방향 초안 도구다.
- 외부 UI mockup은 import할 수 있으며 `reference`, `seed`, `canonical` 권위를 명시한다.
- 모든 adapter는 capability를 먼저 확인하고 없는 기능을 있는 것처럼 흉내 내지 않는다.

## 2. 공통 adapter protocol

각 adapter는 동일한 lifecycle을 가진다.

```text
discover → capability → authenticate → plan → execute → normalize → receipt
```

필수 interface:

| 기능 | 책임 |
|---|---|
| `descriptor` | provider ID, adapter version, supported schema를 제공 |
| `discover` | 실행 파일, config, project link를 read-only로 찾음 |
| `capabilities` | read/write, dependencies, hierarchy, comment, attachment 등 지원 기능 반환 |
| `health` | auth, connectivity, version compatibility 확인 |
| `plan` | 실행 전 local/remote mutation과 approval 등급 표시 |
| `execute` | idempotency key를 포함해 승인된 작업만 수행 |
| `normalize` | provider 결과를 공통 result schema로 변환 |
| `receipt` | remote ID, revision, URL, checksum, redaction 정보를 기록 |

adapter config는 `.harness/integrations/`에 두고 credential은 environment 또는 OS credential store에 둔다. 모든 external write는 operation ID로 중복 여부를 확인한다.

## 3. Task management

### 공통 작업 모델

```text
work item
├── objective / acceptance criteria
├── status / owner
├── dependencies / blockers
├── related spec·policy·scenario·contract IDs
├── affected workspaces
├── claim / branch / PR links
└── evidence / release target
```

하네스는 이 모델을 제공하지만 live status를 중복 저장하지 않는다. `.harness/work/links.yaml`에는 provider ID와 repository 개체 연결만 둔다.

내장 Git fallback에서는 active work/claim을 feature branch의 첫 commit으로 remote에 공개하고, CLI가 remote refs의 machine file을 checkout 없이 읽어 충돌을 검사한다. 완료된 work는 merge 전에 compact evidence로 바꾼다. remote에 공개하지 않은 local claim은 한 컴퓨터 밖의 협업 상태 원본이 될 수 없으며 진단 결과를 `unknown`으로 유지한다.

### 추천 순서

| 상황 | 추천 | 이유 |
|---|---|---|
| GitHub에서 일반 팀 협업 | GitHub Issues + Projects | PR/commit 연결, sub-issue, dependency, 공개 지원 흐름이 자연스러움 |
| 이미 Jira/Linear를 운영 | 기존 Jira/Linear | 조직 상태 원본을 하나로 유지 |
| local/offline·agent-heavy graph | Beads | repository 근처에서 dependency graph를 관리 가능 |
| provider를 원하지 않음 | 내장 Git fallback | YAML work item과 claim을 branch/PR 없이도 사용 가능 |

GitHub Issue는 단순 bug 게시판이 아니다. capability·journey·기능·bug·release 준비를 acceptance criteria와 dependency가 있는 work item으로 만들고, PR이 이를 닫는다. 큰 항목은 sub-issue로 나누되 제품 전체 정의 자체를 수백 개의 임시 Issue에 분산하지 않는다. approved spec이 의미 원본이고 Issue는 실행 상태 원본이다.

### Provider 기능 차이

- hierarchy가 없으면 parent link를 local metadata로만 표시하고 remote 구조를 위조하지 않는다.
- dependency가 없으면 blocker 관계를 설명/label로 동기화하되 capability warning을 낸다.
- provider가 unavailable이면 last-known status를 완료 사실로 사용하지 않고 `unknown`으로 표시한다.
- provider 변경은 open item ID mapping, status mapping, backlink, duplicate 방지 검사를 거친다.

## 4. Superpowers, BMAD, Beads의 위치

| 도구 | 강점 | 이 제품이 맡지기는 영역 | 함께 쓰는 방식 |
|---|---|---|---|
| Superpowers | brainstorming, TDD, debugging, review 등 agent workflow | multi-repo 제품 원본, workspace/submodule, cross-provider 상태, RC 고정 | 설치돼 있으면 해당 workflow를 호출하되 하네스 원본과 gate를 유지 |
| BMAD | 분석·기획·역할별 방법론과 산출물 | repository actual state, Git conflict reservation, contract/pointer integration | 발견 산출물을 import해 canonical spec으로 정규화 |
| Beads | local task dependency graph | 제품 정책·contract·release evidence | task provider adapter로 선택 가능 |

세 도구를 단순히 묶는 것만으로는 같은 source of truth를 읽으며 full-stack multi-repo 제품을 release까지 일관되게 이어가는 문제가 해결되지 않는다. 이 제품의 차별점은 **제품 의미, workspace 실제 상태, contract/DB/UI baseline, 작업 dependency, 충돌 위험, RC를 하나의 검증 가능한 graph로 연결하는 것**이다.

## 5. Git host adapter

지원 계층:

1. GitHub: first-class 검증 대상
2. GitLab: standard adapter
3. Bitbucket: standard adapter
4. generic Git remote: branch/push/fetch/tag 기본 기능

Git host adapter는 다음 capability를 선언한다.

- protected branch와 required checks
- PR/MR, draft, review, CODEOWNERS 유사 기능
- Issue/project 연결
- release와 artifact
- commit/tag signature와 provenance

generic Git에서는 host API 기능 없이 local checks와 Markdown PR template를 제공한다. host-specific file은 해당 provider를 선택했을 때만 생성한다.

## 6. DBML과 dbdiagram

### 정식 흐름

```text
요구사항·정책·data lifecycle 합의
→ Git의 contracts/data/*.dbml 수정
→ DBML syntax/semantic 검사
→ migration·compatibility 영향 분석
→ dbdiagram project에 push하여 시각 검토
→ 사용자/팀 피드백
→ 필요한 수정은 다시 Git DBML change로 반영
```

2026-07-06 공개된 dbdiagram CLI의 `init`, `push`, `pull`을 adapter에서 지원한다. `DBDIAGRAM_TOKEN`은 secret으로만 주입한다. DBML CLI는 syntax check와 SQL/DBML 변환에 사용할 수 있다.

### dbdiagram에서 직접 바꾼 경우

`pull`을 canonical file 위에 바로 실행하지 않는다.

1. `.harness/local/dbdiagram/<operation-id>/`로 pull한다.
2. canonical DBML과 semantic three-way diff를 계산한다.
3. table/column/relation/index/note 변경과 관련 policy·contract·migration 영향을 보여준다.
4. AI가 수정 의도를 추론할 수 없거나 의미가 달라지면 “왜 이렇게 바꿨는지” 질문한다.
5. 승인된 change로 canonical DBML을 수정하고 test/migration을 갱신한다.
6. 다시 push해 remote와 Git fingerprint 일치를 확인한다.

### CI

- PR에서는 DBML validate와 semantic diff만 수행한다.
- protected integration branch에서만 승인된 project로 push할 수 있다.
- fork PR이나 untrusted code에는 token을 노출하지 않는다.
- remote project ID, last synchronized fingerprint, URL만 tracked config에 저장한다.

## 7. UI source adapter

지원 source:

- local image/PDF/document
- Figma/Penpot 등 design provider
- 접근 가능한 URL 또는 prototype
- 기존 HTML/component code
- AI나 사용자가 만든 mockup bundle

import record:

```yaml
id: source.ui.checkout-2026-07
kind: figma
authority: seed
version: "..."
license: internal
content_hash: "sha256:..."
imported_at: "..."
coverage_refs:
  - journey.customer.checkout
```

권위 의미:

- `reference`: 영감과 비교 자료. 기존 canonical UI를 자동 변경하지 않는다.
- `seed`: 초기 구현 시작점. 제품 spec과 accessibility 기준에 맞춰 수정 가능하다.
- `canonical`: 승인된 UI baseline. 변경 시 impact와 approval을 요구한다.

외부 UI는 격리 import에서 malware, path, license, secret, font/asset dependency를 검사한다. 픽셀만 가져오지 않고 role, journey, state, responsive behavior, error/empty/loading/accessibility coverage와 연결한다.

Figma 연결은 유용하지만 core에 포함하지 않는다. Plugin 설치 시 별도 connector가 있으면 사용자에게 선택지로 추천한다.

## 8. AI client adapter

### 지원 등급

- `verified`: 실제 end-to-end conformance suite를 통과
- `standard-compatible`: Agent Skills/Markdown/command 호출 규칙으로 설계되었으나 전체 suite 미검증
- `fallback`: AI가 파일과 명령을 수동으로 따라야 함

초기 release에서 Codex는 verified 목표다. Claude Code와 GitHub Copilot은 표준 호환을 목표로 하고, 실제 suite가 통과한 버전만 verified로 승격한다.

공통 repo-local entry는 `AGENTS.md`, `.harness/entry.md`, `.agents/skills/<product-skill>/SKILL.md`다. client별 파일은 이 원본을 가리키는 작은 adapter만 둔다.

## 9. Dependency와 설치 정책

- core CLI는 외부 runtime 없이 동작한다.
- Git이 없으면 발견·local spec은 가능하지만 협업/release capability를 제한한다.
- dbdiagram/DBML 기능을 선택했을 때만 Node 기반 CLI를 발견하거나 설치 안내한다.
- adapter는 최소/검증 version range와 실제 version을 결과에 포함한다.
- package manager script를 silent install하지 않는다.
- install은 공식 source, pinned version, checksum을 보여주고 승인받는다.
- external tool을 못 쓰면 Markdown/파일 export fallback을 제공한다.

## 10. Offline과 장애

| 장애 | 계속 가능한 일 | 차단되는 일 |
|---|---|---|
| Git host offline | local code/test/spec/contract, commit, conflict preflight 일부 | remote claim 확인, push/PR, required checks |
| task provider offline | linked local scope와 acceptance 기준 작업 | 정확한 live status/assignee 변경 |
| dbdiagram offline | DBML 편집·검사·migration test | remote diagram sync |
| design provider offline | cached approved snapshot 기반 구현 | 새 canonical 변경 확인 |
| AI Plugin 없음 | CLI와 Markdown 절차 | 자동 skill routing/Hook 편의 기능 |

장애를 성공으로 표시하지 않고 `unknown` capability와 재시도 명령을 남긴다.

## 11. 공식 기준 자료

- [Git submodule documentation](https://git-scm.com/docs/git-submodule.html)
- [Git worktree documentation](https://git-scm.com/docs/git-worktree.html)
- [GitHub issue dependencies](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-issue-dependencies)
- [dbdiagram release notes](https://docs.dbdiagram.io/release-notes/)
- [DBML CLI](https://dbml.dbdiagram.io/cli/)
- [Agent Skills specification](https://agentskills.io/specification)
- [Superpowers](https://github.com/obra/superpowers)
- [BMAD documentation](https://docs.bmad-method.org/reference/modules/)
- [Beads releases](https://github.com/steveyegge/beads/releases)

## 12. 수용 기준

- provider가 없어도 local Git fallback으로 작업을 계획·추적할 수 있다.
- GitHub를 선택하면 Issue, dependency, PR이 spec/contract/workspace와 연결된다.
- dbdiagram에서 직접 바뀐 schema가 Git DBML을 조용히 덮어쓰지 않는다.
- Superpowers/BMAD/Beads를 설치하지 않아도 core lifecycle이 완전하다.
- 외부 tool 장애와 capability 부족을 숨기지 않고 안전하게 degrade한다.
