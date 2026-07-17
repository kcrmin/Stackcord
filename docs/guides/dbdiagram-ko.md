# DBML과 dbdiagram

## Canonical model 하나 유지

Canonical DBML을 stable entity ID와 함께 Git에 저장하고 code처럼 review합니다. Contract와 제품 정책은 동작, DBML은 물리 data 구조, migration은 순서가 있는 전이를 설명합니다. Rendered diagram을 두 번째 source of truth로 만들지 않습니다.

## AI와 model 상의

AI에게 정책·scenario·retention·privacy·access·실패·scale 요구를 바탕으로 data model을 제안하거나 수정하게 합니다. AI는 DBML을 쓰고 semantic check를 실행합니다. 구현 전에 이름, ownership, cardinality, nullability, uniqueness, lifecycle, deletion, auditability, 민감 data 경계를 검토합니다.

## 격리된 시각화

사용자가 [공식 dbdiagram CLI](https://docs.dbdiagram.io/release-notes/2026-07/)를 선택하면 AI는 `dbdiagram` executable을 감지하고 `.harness/local/dbdiagram/` 아래 operation-scoped local workspace를 만듭니다. Canonical DBML을 `candidate.dbml`로 복사하고 `dbdiagram init --entry candidate.dbml --diagram-id <id>`를 준비한 뒤 외부 작업을 보여주고 나서만 `dbdiagram push` 또는 `dbdiagram pull`을 명시적으로 실행합니다. Credential은 Git 밖에 둡니다. 시각화나 remote 협업이 canonical DBML을 암묵적으로 수정하지 않습니다.

## Remote 변경 조정

`push`는 격리 사본으로 선택한 online diagram을 갱신하여 협업자가 바로 볼 수 있게 합니다. 누군가 외부 diagram을 수정하면 `pull`은 격리 사본만 갱신합니다. Entity·field·index·relation 의미를 비교하고 중요한 변경 이유를 묻습니다. 수용한 차이는 정책·contract·migration impact를 갱신하는 명시적 Git 변경이 됩니다. 거절한 차이는 canonical DBML에 영향을 주지 않습니다.

## Production data 안전하게 발전

파괴적이거나 compatibility에 민감한 차이는 migration 순서, consumer compatibility plan, validation, backup 또는 rollback 전략, TDD evidence가 필요합니다. 병렬 작업 전 migration slot을 예약합니다. Migration이 candidate에 실제 포함될 때만 해당 release evidence를 요구합니다.
