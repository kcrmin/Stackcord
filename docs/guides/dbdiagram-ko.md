# DBML과 dbdiagram

## Canonical model 하나 유지

Canonical DBML을 stable entity ID와 함께 Git에 저장하고 code처럼 review합니다. Contract와 제품 정책은 동작, DBML은 물리 data 구조, migration은 순서가 있는 전이를 설명합니다. Rendered diagram을 두 번째 source of truth로 만들지 않습니다.

## AI와 model 상의

AI에게 정책·scenario·retention·privacy·access·실패·scale 요구를 바탕으로 data model을 제안하거나 수정하게 합니다. AI는 DBML을 쓰고 semantic check를 실행합니다. 구현 전에 이름, ownership, cardinality, nullability, uniqueness, lifecycle, deletion, auditability, 민감 data 경계를 검토합니다.

## 격리된 시각화

dbdiagram CLI 또는 지원 renderer를 사용할 수 있으면 AI는 `.harness/local/dbdiagram/` 아래 operation-scoped local workspace를 만듭니다. Credential은 Git 밖에 둡니다. 시각화나 remote 협업이 canonical DBML을 암묵적으로 수정하지 않습니다.

## Remote 변경 조정

누군가 외부 diagram을 수정하면 격리 workspace로 가져와 entity·field·index·relation 의미를 비교하고 중요한 변경 이유를 묻습니다. 수용한 차이는 정책·contract·migration impact를 갱신하는 명시적 Git 변경이 됩니다. 거절한 차이는 canonical DBML에 영향을 주지 않습니다.

## Production data 안전하게 발전

파괴적이거나 compatibility에 민감한 차이는 migration 순서, consumer compatibility plan, validation, backup 또는 rollback 전략, TDD evidence가 필요합니다. 병렬 작업 전 migration slot을 예약합니다. Migration이 candidate에 실제 포함될 때만 해당 release evidence를 요구합니다.
