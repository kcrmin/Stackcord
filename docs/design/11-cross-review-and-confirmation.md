# 전체 설계 교차 검토와 확정 기록

> 상태: 확정
>
> 검토일: 2026-07-16

## 1. 검토 결론

현재 01~10 설계는 서로 함께 구현 가능한 하나의 제품 명세다. 제품명 보류를 제외하면 추가 사용자 결정 없이 source 구현을 시작할 수 있다. 다만 문서가 완성되었다는 사실을 실제 Plugin·CLI·release 완성으로 오해해서는 안 된다.

## 2. 핵심 요구 추적

| 요구 | 확정 설계 | 구현 계획 |
|---|---|---|
| 새 서비스 질문·정규화·장기 context | 01, 02, 03 | Task 3, 8, 9 |
| clone 후 다른 사람·AI가 계속 개발 | 03, 04, 08, 10 | Task 3, 5, 8, 9 |
| root harness와 빠른 submodule 구성 | 01, 02, 04 | Task 5, 8 |
| workspace와 submodule 의미 구분 | 02, 03, 04 | Task 2, 5 |
| 전체 제품→stack→전체 UI→contract/DBML | 01, 02 | Task 7, 8 |
| 안정된 interface/skeleton 후 병렬 full-stack | 01, 04 | Task 6, 7, 8 |
| TDD 고정과 좁은 예외 | 01, 02, 09 | 모든 behavior Task |
| path 밖 의미 충돌 사전 감지 | 03, 04 | Task 3, 6 |
| Git convention에 AI 흔적 없음 | 04 | Task 5 |
| GitHub Issue 강제하지 않음 | 06 | Task 10 |
| Superpowers/BMAD/Beads와 역할 구분 | 06, 10 | Plugin/provider integration |
| 외부 UI mockup import | 01, 02, 06 | Task 7 |
| DBML을 dbdiagram에서 즉시 시각 검토 | 01, 06 | Task 7, 10 |
| dbdiagram 직접 수정 이유 확인 | 03, 06 | Task 7 |
| AI context 망각/압축 복구 Skill | 03, 05, 08 | Task 3, 9 |
| macOS·Windows cross-platform | 07, 09 | Task 1~13 CI matrix |
| Plugin을 GitHub로 공유 | 08, 10 | Task 9, 13 |
| strict AI RC + 같은 SHA 사용자 확인 | 09 | Task 11, 13 |
| production product로 공개 | 09, 10 | Task 13 |

## 3. 충돌 정책 검증

다음 시나리오가 한 가지 통제만 믿지 않고 중첩 방어된다.

| 충돌 | 1차 | 2차 | 최종 |
|---|---|---|---|
| 같은 파일 | worktree/clone 격리 | path claim | Git merge + test |
| 다른 파일·같은 정책 | stable ID claim | impact graph/review | scenario regression |
| contract 변경 | compatibility diff | change bundle·merge order | provider/consumer test |
| DB migration | entity/sequence claim | DBML semantic diff | clean replay/rollback |
| frontend/backend 기준 차이 | contract fingerprint | generated client/mock | integration/E2E |
| submodule pointer | exact SHA check | workspace merge first | root clean-clone gate |
| local/remote 차이 | actual-state refresh | hidden mutation 금지 | 사용자 선택·receipt |
| AI context 차이 | source fingerprint | context audit | gate/CI enforcement |

따라서 `.gitignore`, worktree, TDD 중 하나만으로 충돌을 막는다고 주장하지 않는다. 각각 local file, behavior, semantic coordination의 다른 문제를 해결한다.

## 4. 수정한 모순

- 전통적 `develop` 중심 Git Flow를 기본으로 두지 않고 protected `main`으로 통일했다.
- Handoff를 일상적 “이어받기”가 아니라 실제 책임자 변경으로 한정했다. 평상시 context 유지에는 branch work record와 refresh를 쓴다.
- “child”와 “ticket slug”를 필수 용어에서 제거하고 workspace/submodule/work ID로 통일했다.
- Plugin이 제품 핵심을 모두 포함한다는 표현을 제거했다. CLI와 project repository가 독립적으로 동작한다.
- Codex Plugin을 모든 AI client 공통 배포처럼 표현하지 않고 Agent Skills/Markdown fallback을 분리했다.
- GitHub Issues는 추천 기본값이지 고정 requirement가 아니며, product spec과 task status의 원본을 분리했다.
- dbdiagram remote가 schema 원본이라는 오해를 제거하고 Git DBML을 canonical로 고정했다.
- 전체 UI 선행이 거대한 장기 branch나 waterfall이 되지 않도록 역할·domain별 통합과 stale 재개방을 명시했다.
- 모든 interface를 미리 만드는 과설계를 피하고 공유되는 안정적 경계와 generated skeleton만 먼저 만든다.
- worktree마다 달라지는 `current.json`을 추적 상태에서 Git-ignored local cache로 이동해 일상 refresh 충돌을 제거했다.
- 내장 Git fallback claim을 feature branch의 첫 remote commit/Draft PR로 공개하고 remote refs에서 읽게 해 다른 branch에도 보이도록 했다. remote에 공개되지 않으면 `unknown`이다.

## 5. 공개 전 실제로 증명해야 할 항목

문서상 확정만으로 통과 처리할 수 없는 항목이다.

- macOS arm64/x86_64, Windows arm64/x86_64 실제 설치·upgrade·uninstall
- Codex Plugin 현재 validator와 Hook ingestion 동작
- repo-local Agent Skill의 다른 client 호환성
- GitHub/dbdiagram rate-limit·auth·offline 실제 adapter 동작
- malicious repository/import 보안 test와 독립 review
- signed artifact, SBOM, provenance, Homebrew/WinGet publish
- 신규 프로젝트와 multi-repo clone의 release까지 E2E
- 같은 RC에 대한 사용자 검증

이 항목은 구현 계획 Task 9~13의 release blocker다. issue를 받아서 알아가겠다는 이유로 생략하지 않는다. 공개 뒤 Issue는 미지 환경을 확장하는 수단이다.

## 6. 의도적으로 보류한 한 가지

공개 제품 이름은 사용자의 이전 결정을 존중해 보류했다. 구현은 `fullstack-orchestrator`/`orchestrator` working ID로 시작할 수 있지만 public repository·Plugin·package를 만들기 전에 이름·namespace·혼동 가능성을 확인하고 한 번 고정해야 한다.

## 7. 최종 확인

- 제품 범위: 구현 시작 가능
- lifecycle과 generated project structure: 구현 시작 가능
- collaboration, source of truth, approval, provider, CLI, Plugin, security, release 정책: 구현 시작 가능
- production release: 아직 불가 — source와 검증 artifact가 없음
- 다음 작업: 구현 계획 Task 1부터 TDD로 실행
