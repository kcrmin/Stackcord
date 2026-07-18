# 핵심 개념

## 대화와 검증

Skill은 자연어 목표를 해석하고 제품 의미를 발견하며 무엇을 물을지 정하고 trade-off를 설명합니다. CLI는 그 판단을 대신하지 않고 관찰 가능한 상태와 재현 가능한 identity를 검증합니다. 명시적이고 보이는 apply를 요청하지 않으면 read-only 또는 plan-first입니다.

## Source of truth

실제 Git·submodule·worktree·filesystem 상태가 cache summary보다 우선합니다. `specs/`는 제품 의도와 정책, `contracts/`는 component 사이 동작과 실패, Git의 DBML은 data 구조를 소유합니다. `.harness/`는 이 원본을 index하고 작은 조정 상태를 기록합니다. Live task status는 선택한 provider 하나만 소유하며 Git-local은 기본값이지 강제가 아닙니다.

Memory는 canonical 저장소 evidence가 아닙니다. 대화-memory Plugin은 AI가 이전 결정을 찾는 데 도움을 줄 수 있지만 현재 Git identity, 제품 승인, provider revision, release evidence를 증명할 수 없습니다. AI는 중요한 hint를 canonical file 또는 실제 상태로 다시 확인하고 근거가 없으면 unknown으로 표시해야 합니다.

## Identity와 stale

제품 사실에는 stable ID를 부여하여 파일 이동과 문서 재작성 뒤에도 참조를 유지합니다. Fingerprint는 index, change plan, evidence, release candidate를 정확한 내용에 연결합니다. Dependency fingerprint가 바뀌면 downstream은 새로 확인하거나 의도적으로 수용할 때까지 stale이며 timestamp만 믿지 않습니다.

## Workspace와 child

Workspace는 root harness에 표시되는 저장소 또는 별도로 test할 수 있는 component입니다. Child workspace는 흔히 자기 history·branch·test·release 책임을 가진 Git submodule입니다. Root는 수용한 정확한 child commit을 기록합니다. Frontend와 backend라는 이름만으로 분리하지 않고 독립 ownership이나 lifecycle이 실제로 있을 때 분리합니다.

## 충돌 모델

Filesystem overlap만 충돌은 아닙니다. 작업 선점은 정책, scenario, contract, DB entity, migration slot, UI flow, dependency major, stable ID, root pointer도 다룹니다. 선택한 외부 provider가 live status 원본으로 남고 Git compare-and-swap 선점이 여러 저장소의 의미를 배타적으로 보호합니다. 충돌 발견은 조용한 병렬 수정을 막고 ownership·boundary·sequence를 합의하게 하며 자동 reset·stash·rebase를 하지 않습니다.

## 반복 개발과 TDD

팀은 제품 전체 의미와 UI coverage를 세운 뒤 역할·domain·journey slice를 계속 배포합니다. 공유 interface·contract·schema가 병렬 작업의 모호함을 줄일 때는 일찍 합의합니다. 동작·bug·contract 변경·migration·UI interaction·integration rule은 실패하는 test 또는 재현 가능한 실패 검사에서 시작해 통과 evidence로 끝냅니다.

## Context 복구

Clone, session 재시작, context 압축 뒤 AI는 `AGENTS.md`와 `.harness/entry.md`를 읽고 canonical file을 audit하며 실제 Git과 workspace 상태를 확인한 뒤 안전한 다음 행동 하나를 권합니다. 유효한 현재 원본에 답이 있는 질문을 반복하면 안 됩니다. Plugin이 없어도 `.agents/skills/use-project-harness/`가 편의만 줄어든 같은 복구 진입점을 제공합니다.

## Release identity

기본 release 준비는 source commit, artifact digest, 제품/docs/contract fingerprint, TDD evidence, integration evidence, 해당 migration evidence를 하나의 candidate digest로 묶습니다. Candidate가 만들어진 뒤 사용자 검증을 받고 그 검증은 정확히 같은 digest를 지정해야 합니다. Strict release는 supply-chain evidence와 보호된 publication tooling을 선택 profile로 추가합니다.
