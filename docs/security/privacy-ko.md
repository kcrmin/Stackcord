# 개인정보 보호

Core는 local-first이며 telemetry, daemon, 중앙 service, 필수 account가 없습니다. 원본 대화·prompt·source file 내용·secret·불필요한 개인 정보는 canonical project state가 아닙니다.

Repository에는 정규화된 결정, stable ID, fingerprint, provider link, claim, 재현 가능한 evidence를 저장합니다. Credential은 OS/provider credential store와 환경 변수에 두며 설정에는 환경 변수 이름만 기록합니다.

Diagnostic export는 source 내용과 provider payload를 제외하고 home/project path를 상징 이름으로 바꾸며 URL credential과 secret-like 값을 제거합니다. Version, architecture, stable error code, redacted state, operation receipt ID만 포함합니다. 프로젝트 식별자 자체가 민감할 수 있으므로 공유 전 검토합니다.

외부 adapter는 선택 사항입니다. Provider write는 범위가 맞는 승인과 idempotency receipt가 필요합니다. Plugin을 제거해도 프로젝트가 소유한 spec과 contract는 삭제하지 않습니다.
