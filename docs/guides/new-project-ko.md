# 새 프로젝트 흐름

1. “새 서비스를 시작해”라고 말합니다. 발견 결과는 `.harness-drafts/<id>/`에 정규화된 요약·결정·미해결 질문으로 저장하며 원본 대화는 저장하지 않습니다.
2. AI가 추천 답을 먼저 둔 객관식 질문을 한 번에 하나씩 묻고 사용자는 자유 입력도 할 수 있습니다. 역할·가치·전체 journey·성공/실패 정책·품질·보안·운영과 사용자가 미처 떠올리지 못한 가능성까지 살핍니다.
3. 제품 요약과 저장소 이름을 승인합니다. `project init`이 repo-local Skill, harness, specs, contracts, docs가 있는 기술 중립 root를 만듭니다.
4. 필요한 capability와 제약이 확인된 뒤 기술을 고릅니다. 선택 시점에 유지보수·보안·release 상태를 다시 확인합니다.
5. 제품 전체 executable UI coverage를 세웁니다. 외부 mockup은 reference·seed·canonical 중 하나로 가져올 수 있습니다.
6. Contract·실패 동작·DBML·공유 구현 경계를 정한 뒤 작은 vertical slice를 red-green-refactor 증거와 conflict claim으로 구현합니다.
7. 호환성 순서대로 통합하고 production을 강화한 뒤 하나의 RC를 고정합니다. 사용자가 같은 digest를 검증한 뒤에만 공개합니다.

중요한 결정마다 checkpoint를 남기므로 context 압축이나 담당자 변경 뒤에도 발견을 다시 시작하지 않습니다.
