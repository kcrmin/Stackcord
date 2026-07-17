# 신규 프로젝트

## 발견으로 시작

AI에게 만들 서비스를 설명합니다. AI는 먼저 directory와 사용 가능한 도구를 진단한 뒤 답이 제품 동작·architecture·위험·범위를 바꿀 때만 질문을 하나씩 합니다. 질문은 보통 추천 선택지를 먼저 둔 서로 배타적인 2~3개 선택지와 자유 입력을 제공합니다.

중요한 답변 뒤 AI는 대화 원문이 아니라 현재 제품 사실을 담은 정규화 checkpoint를 갱신합니다. Privacy·security·accessibility·실패·운영·observability·retention·악용 상황이 중요하면 사용자가 놓친 항목도 제시합니다.

## 기술 확정 늦추기

Framework나 infrastructure를 고르기 전에 capability, 품질 목표, 팀 제약, 배포 환경, data 민감도, scale, 운영 ownership을 설명합니다. 기술 요구와 기술 선택을 분리해 기록합니다. 선택이 필요해지면 가능한 후보를 비교하고 현재 공식 유지보수·보안·release 상태를 확인합니다.

## Coverage를 세우고 나누기

서비스 전체의 역할, journey, 정책, 실패 결과, UI state를 정의합니다. 외부 mockup을 reference·seed·canonical로 가져올 수도 있습니다. 이 기준선은 고정된 waterfall spec이 아닙니다. 역할·domain·journey별 작은 변경으로 나누어 계속 통합하고 학습에 따라 기준선을 고칩니다.

## Harness 초기화

지속 가능한 root를 만들 만큼 서비스 identity가 생기면 AI에게 프로젝트 초기화를 요청합니다. AI는 정확한 파일을 미리 보여준 뒤 최소 framework-neutral harness를 만듭니다. 협업에는 Git을 일찍 시작합니다. Child 저장소는 frontend/backend라는 이름 때문이 아니라 독립 workspace가 정당화되는 즉시 submodule로 추가합니다.

## 공유 경계로 구현

병렬 작업 전에 여러 변경이 의존하는 interface·contract·DBML·실패 동작을 정의합니다. 의미 범위를 claim하고 일반적인 feature branch 또는 격리 worktree를 만들며 실패 test를 작성한 뒤 최소 동작을 구현하고 자주 통합합니다. 구현 중 실제 제품 결정이 드러나면 제품 checkpoint도 수정합니다.
