# 핵심 개념

하네스는 `.harness/` 폴더 하나가 아니라 지속 가능한 협업 구조 전체입니다. `specs/`는 제품 의미, `contracts/`는 구성 요소 사이의 의무, `.harness/`는 조정 상태와 증거, `docs/`는 가이드와 runbook을 소유합니다.

Workspace는 구현·검증·소유권·contract 경계이며 root, directory, submodule, external 중 하나입니다. child는 중첩 agent나 process를 뜻할 뿐 프로젝트 구조 개념이 아닙니다. 새 경계가 진짜 별도 저장소와 정확한 pinned commit을 필요로 할 때 submodule을 권장합니다.

`policy.account.recovery.rate-limit` 같은 stable ID는 파일이 이동해도 유지됩니다. ticket 번호와 branch 설명은 실행 식별자이지 제품 의미가 아닙니다. Claim은 누가 path·policy·contract·migration·UI flow·dependency·pointer를 바꾸려는지 알리지만 분산 lock은 아닙니다.

Lifecycle 단계는 waterfall 일정이 아니라 의존성 gate입니다. 제품 전체 의도와 UI coverage를 먼저 세우되 role/domain/journey 단위의 작은 변경으로 통합합니다. 공유 interface와 실패 의미를 먼저 정한 뒤 병렬 구현하고, vertical slice는 TDD로 진행합니다.

Plugin은 Skills와 선택 Hook을 묶습니다. repo-local Skill은 Plugin 없이도 이어서 개발하게 합니다. CLI는 schema·Git·operation·충돌·adapter·release를 검사합니다. Hook은 신뢰된 session에 context audit을 알릴 뿐 파일을 쓰거나 외부 시스템을 호출하지 않습니다.
