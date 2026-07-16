# 기존 프로젝트 도입

“이 저장소를 도입해서 이어서 해”라고 말합니다. 계획 단계는 read-only이며 Git history, dirty file, root/workspace 경계, 기존 지침, 기술, test, CI, 문서, contract, 알 수 없는 제품 동작을 조사합니다.

`project adopt`는 빠진 harness 파일과 README/AGENTS의 명시적 managed section만 추가합니다. 사용자 내용·Git history·topology·dirty file을 보존합니다. 기존 `.editorconfig`나 `.gitattributes` 정책이 충돌하면 덮어쓰지 않고 차단합니다.

첫 baseline은 희망적인 재설계가 아니라 characterization입니다. 관찰 가능한 기존 동작을 stable policy와 scenario에 연결하고 unknown을 표시하며 중요한 동작을 test로 고정한 뒤 제품 변경을 별도로 제안합니다. 원본 관계가 안정될 때까지 도입 후와 모든 변경 전에 `context audit`을 실행합니다.
