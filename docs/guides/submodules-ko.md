# Submodule·worktree·협업

## Workspace 경계 결정

Ownership·permission·dependency·배포·release lifecycle이 의미 있게 독립적일 때 child 저장소를 사용합니다. 단순 프로젝트를 frontend/backend 모양 때문에 나누지 않습니다. 각 child를 `.harness/workspaces.yaml`과 Git `.gitmodules`에 등록하고 root 저장소는 coordination과 contract 경계로 둡니다.

## Clone과 진단

Clone 뒤 AI에게 프로젝트를 이어서 하라고 말합니다. AI는 `.gitmodules`, root index pointer, 초기화된 module path, child HEAD, dirty, detached, remote reachability를 비교합니다. 안전한 sync plan은 root가 기록한 정확한 commit을 사용합니다. 누락·dirty·불일치·공개되지 않은 child 상태를 조용히 교체하지 않고 설명합니다.

## 병렬 작업 격리

보통 contributor 한 명이 변경 하나의 일반 branch를 소유합니다. 같은 clone에 여러 branch가 필요하면 Git worktree를 사용합니다. 구현 전에 영향받는 path·정책·scenario·contract·DB entity·migration·UI flow·dependency major·pointer 의도를 semantic claim으로 기록합니다. Claim은 ownership을 조정하지만 대화를 대체하지 않습니다.

## 의존 순서로 통합

공유 경계가 바뀌면 먼저 additive contract를 합의합니다. 가능하면 provider를 consumer보다 먼저 merge하고 호환 동작이 생긴 뒤 frontend나 다른 consumer를 연결합니다. Child 작업은 child 저장소에서 commit·review합니다. 선택한 child commit이 reachable하고 coordinated integration 준비가 되었을 때만 root pointer를 갱신하고 review합니다.

## 충돌 상황 처리

- 같은 파일·다른 의미: 파일을 나누거나 수정 순서를 직렬화합니다.
- 같은 contract 또는 정책: 한 owner가 경계를 발전시키고 다른 작업은 기다리거나 호환 version을 사용합니다.
- 같은 DB entity 또는 migration slot: 구현 전에 migration 순서와 rollback을 합의합니다.
- 같은 UI flow: state ownership과 acceptance behavior를 합의합니다.
- Dependency major 겹침: upgrade 기준선을 하나 먼저 merge합니다.
- Root pointer 겹침: child merge 뒤 integration owner 한 명을 정합니다.
- Dirty/diverged branch: 멈추고 정확한 commit과 변경을 보여준 뒤 사용자가 pull·rebase·merge·cleanup을 고릅니다.

## Handoff를 의도적으로 사용

Handoff는 실제 ownership 변경, 중단, 담당자 부재에 사용합니다. 현재 의도·evidence·blocker·정확한 repository identity를 기록하여 다음 owner가 chat에서 작업을 복원하지 않게 합니다. 일반 병렬 contributor는 자기 범위를 유지하고 canonical spec·contract·Git으로 공통 context를 공유합니다.
