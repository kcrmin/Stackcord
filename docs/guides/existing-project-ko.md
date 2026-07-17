# 기존 프로젝트 도입

## 변경 전 검사

AI에게 저장소를 이어서 하거나 도입하라고 말합니다. AI는 가장 가까운 trusted instruction을 읽고 언어와 기존 도구를 감지하며 Git·submodule을 확인하고 제품 문서·contract·schema·test·CI·배포·task tracking을 inventory합니다. 파일이나 Git에서 알 수 있는 사실은 사용자에게 다시 묻지 않습니다.

## 비파괴 계획 미리 보기

도입은 최소 harness와 managed section만 추가합니다. 기존 파일·설정·source code·branch·task system·history가 계속 authoritative합니다. Target file에 사용자 내용이 있으면 경계가 있는 managed section을 합칠 수 있는지 plan에 보여주며 안전하지 않은 충돌은 덮어쓰지 않고 중단합니다.

## 제품 의미 복원

AI는 저장소가 증명하는 내용을 요약하고 사실과 가정을 분리한 뒤 중요한 미해결 질문만 묻습니다. 기존 정책·scenario·contract·DB entity·migration·UI flow에 stable ID를 부여하고 fingerprint를 기록합니다. 제품이나 운영 근거가 바꾸라고 하지 않는 한 기존 기술을 유지합니다.

## Work status 원본 하나 선택

Git-local이 안전한 기본값입니다. 저장소가 이미 GitHub·Jira·Linear·Beads·다른 시스템을 사용하면 실제 connector나 사용 가능한 local command가 있는지 확인한 뒤 유지하도록 추천할 수 있습니다. 선택한 도구 하나만 live task-status authority가 되며 지원하지 않는 adapter가 있다고 가장하지 않습니다.

## 첫 변경 시작

Context와 Git audit을 실행하고 stale 또는 divergent 상태를 사용자와 해결하며 dependency-ready 작업을 고른 뒤 code를 쓰기 전에 의미 충돌을 검사합니다. 모호하거나 위험하지 않으면 기존 branch·commit convention을 유지합니다. TDD와 저장소의 기존 test/build interface를 사용합니다.
