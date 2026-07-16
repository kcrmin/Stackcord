# AI 풀스택 오케스트레이션 도구 작업 설계

> 상태: 확정된 결정 요약 — 공개 이름만 보류
>
> 마지막 갱신: 2026-07-16
>
> 이 문서는 대화 원문을 보관하는 문서가 아니다. 합의된 제품 방향, 변경된 결정, 미확정 사항을 중립적인 문장으로 정리하는 기준 문서다. 최종 구현 명세는 설계가 승인된 뒤 별도로 작성한다.

## 1. 제품 목표

이 제품은 특정 프레임워크의 애플리케이션 코드를 생성하는 템플릿이 아니다.

사용자와 AI가 서비스 아이디어를 충분히 구체화하고, 여러 사람과 AI가 동일한 제품 의도·계약·프로젝트 상태를 기준으로 풀스택 개발을 협업하며, 중단되거나 대화 컨텍스트가 압축되어도 현재 상태를 복구해 최종 릴리스까지 도달하게 하는 오케스트레이션 도구다.

사용자는 Git과 내부 명령을 일일이 조작하지 않는다. 기본 상호작용은 다음과 같은 자연어 요청이다.

- 새 프로젝트 시작해줘.
- 지금 뭐 해야 해?
- 내가 맡은 작업을 시작해줘.
- 다른 작업과 충돌하는지 확인해줘.
- 프로젝트 문맥을 다시 읽어.
- 이제 릴리스할 수 있어?

AI는 내부적으로 프로젝트 문서, Git, 원격 저장소, 서브모듈, 계약, 작업 상태를 확인하고 안전한 다음 행동을 설명하거나 실행한다.

### 출시 원칙

첫 공개 버전부터 이 문서가 약속한 전체 사용자 흐름과 production gate를 충족한다.

첫 공개 릴리스는 이 문서에서 약속한 전체 사용자 흐름을 실제 신규 프로젝트에서 끝까지 수행할 수 있는 프로덕션 제품이어야 한다.

- 서비스 발견부터 프로젝트 구성, 협업, 계약 관리, 통합, 릴리스 검사까지 끊기지 않아야 한다.
- macOS와 Windows에서 설치, 실행, 업데이트, 제거가 검증되어야 한다.
- 여러 사람과 AI가 동시에 작업하는 충돌·복구 시나리오가 검증되어야 한다.
- 예제용 mock, 수동 우회 절차, 특정 개발자 환경에만 맞는 경로를 릴리스 결과에 남기지 않는다.
- 오류가 발생하면 원인, 보존된 작업, 복구 방법을 사용자에게 설명해야 한다.
- 문서, 보안, 권한, 호환성, 마이그레이션, 자동 테스트를 제품 기능과 같은 출시 요건으로 본다.

버전 번호는 호환성과 업데이트를 관리하기 위한 기술적 표기일 뿐, 미완성 제품을 정당화하는 범위 축소 용어로 사용하지 않는다.

## 2. 제품 경계

### 포함하는 것

- 서비스 발견 질문과 정리
- 확정·가설·미정·후속 고려사항 구분
- AI가 놓칠 수 있는 운영·보안·확장 관점 제안
- 외부 UI 목업·디자인·코드 입력의 안전한 import와 source 추적
- 오케스트레이션 하네스 생성
- 필요한 workspace 저장소와 Git submodule 구성
- 공동 프로젝트 컨텍스트 유지와 복구
- 작업 담당, 충돌 탐지, context sync, checkpoint, 선택적 handoff
- 계약 변경 관리
- OpenAPI·DBML 등 계약 검사
- dbdiagram CLI를 이용한 DBML 검토 흐름
- 동작 변경과 bug fix의 TDD 증거와 품질 gate
- 통합 상태와 submodule pointer 검사
- 기술 검증과 사용자 승인을 분리한 릴리스 흐름
- macOS와 Windows 지원
- Plugin이 없는 AI를 위한 Markdown fallback

### 강제하지 않는 것

- 프론트엔드·백엔드 프레임워크
- 프로그래밍 언어
- 데이터베이스 제품
- 클라우드·배포 사업자
- 하나의 작업관리 서비스
- 모든 프로젝트에 동일한 Git Flow
- AI 정체성이 드러나는 branch, commit, PR 이름

## 3. 핵심 사용자 경험

### 새 프로젝트

1. 사용자가 새 서비스를 시작한다고 말한다.
2. AI가 한 번에 하나의 질문을 한다.
3. 객관식 질문에는 기본적으로 `기타`와 `잘 모르겠음`을 제공한다.
4. 원본 답변을 그대로 누적하지 않고 제품 지식으로 정규화한다.
5. 확정된 내용, 현재 가설, 모순, 추가 확인 사항을 계속 갱신한다.
6. 핵심 사용자, 문제, 가치, 주요 흐름이 잡히고 서비스명 또는 안정적인 프로젝트·저장소 이름이 확정되면 오케스트레이션 저장소와 workspace 구성을 제안한다.
7. 필요한 workspace는 필요성이 확정되는 즉시 사용자 승인 후 생성한다.
8. 전체 서비스 의도와 기능이 정의되면 기능·운영 요구를 근거로 아키텍처와 기술 스택을 비교해 확정한다.
9. realtime, offline, media처럼 실패 비용이 큰 기술 가정은 격리된 spike로 UI 전에 검증한다.
10. 선택한 production 기술로 mock data 기반 전체 UI 기준선을 만든다.
11. UI에서 확인된 요구를 기준으로 API·event·DBML 계약을 확정한다.
12. canonical contract에서 공유 type과 client·server stub을 생성하고 module 경계와 최소 기능 골격을 통합한다.
13. 같은 기준선에서 backend 기능 구현과 frontend 연결을 수직 기능 단위로 병렬 진행한다.
14. 전체 통합·강화 검사·릴리스 후보·사용자 검증을 거쳐 릴리스한다.

### 질문 정책

- AI는 이미 합의된 내용이나 기존 결정에서 자연스럽게 따라오는 전제를 다시 질문하지 않는다.
- 이름이 없는 저장소를 임시 이름으로 생성하지 않는다. 서비스명 또는 안정적인 내부 프로젝트 이름을 먼저 확정한다.
- 명백한 후속 결정은 AI가 `이 기준으로 확정했다`고 알리고 문서에 기록한 뒤 넘어간다.
- 사용자 의도, 비용, 위험, 되돌리기 어려운 구조가 달라지는 선택만 질문한다.
- 질문이 필요하면 한 번에 하나씩 A/B/C/D와 `기타`를 제공하고 권장안을 표시한다.

### 기존 프로젝트

사용자가 프로젝트를 열고 `지금 뭐 해야 해?`라고 묻는다.

AI는 다음을 먼저 확인한다.

- 현재 branch와 upstream
- 로컬 수정
- push하지 않은 commit
- 원격보다 앞섰거나 뒤처진 상태
- branch divergence
- workspace와 submodule 상태
- root pointer와 실제 checkout 차이
- 현재 개발 단계
- 담당자가 있는 작업
- 변경 중인 계약
- 시작 가능한 다음 작업

로컬이 깨끗하고 fast-forward만 필요한 상태처럼 안전한 경우에는 상황을 알리고 진행한다. dirty tree, divergence, 공유 branch, 계약 충돌처럼 작업 손실 가능성이 있으면 자동으로 pull, rebase, stash, reset하지 않고 이유와 권장 행동을 설명한 뒤 사용자에게 한 가지 선택만 요청한다.

기존 제품 프로젝트에 도입할 때는 현재 저장소 구조와 Git 이력을 보존하고 오케스트레이션 하네스를 먼저 추가한다. 별도 저장소나 submodule로 전환했을 때 담당 경계, 배포 주기, 권한, 충돌 관리에 실질적인 이점이 있는 경우에만 변경안과 비용을 제시하고 사용자 승인 후 전환한다. 승인 없는 repository 분리, history rewrite, submodule 변환은 하지 않는다.

## 4. 구성 요소

### 표준 Agent Skills

AI가 어떤 질문을 하고 어떤 판단 절차를 따라야 하는지 정의한다. 가능한 한 개방형 Agent Skills 형식을 사용해 Codex에 종속되지 않게 한다.

예상 역할:

- 프로젝트 시작
- 지금 할 일 판단
- 컨텍스트 복구
- 작업 조정
- 계약 변경 관리
- 데이터베이스 설계
- 릴리스 준비 검사

### 프로젝트 하네스

대화와 AI 모델이 바뀌어도 남아야 하는 프로젝트 지식의 원본이다.

제품 의도, 결정, 계약, workspace 구성, 현재 단계, 진행 중인 변경, 검증 결과를 저장한다. 원본 대화나 사용자의 말투는 저장하지 않는다.

프로젝트 하네스는 `.harness/` directory 하나를 뜻하지 않는다. 규범적 서비스 의미를 저장하는 `specs/`, 구성 요소 간 약속을 저장하는 `contracts/`, 오케스트레이션 제어 상태인 `.harness/`, 설명·운영 자료인 `docs/`를 합친 전체 구조다.

### 프로젝트 검사 도구

이전 설계에서 `runtime`이라고 부르던 구성이다. 사용자에게는 `프로젝트 검사 도구`라고 설명한다.

AI의 기억이나 추측이 아니라 실제 Git·파일·계약 상태를 기계적으로 확인하고 구조화된 결과를 돌려준다.

예상 검사:

- Git clean/dirty, ahead/behind/diverged
- 원격 commit 도달 가능 여부
- workspace 상태
- submodule pointer와 checkout 차이
- 계약 호환성
- DBML 유효성
- 릴리스 구성 commit 일치 여부

### Codex Plugin

표준 Skill, 프로젝트 검사 도구 연결, 템플릿, 선택적 Hook을 Codex에 설치하기 쉽게 묶는 배포 방식이다. 다른 AI client는 Agent Skills와 Markdown fallback을 사용하며 제품의 핵심 자체는 Plugin에 종속되지 않는다.

공개 배포는 GitHub repository에 Plugin manifest와 marketplace catalog를 두는 방식을 우선한다.

### Hook

세션 시작, 컨텍스트 압축 전후 같은 lifecycle event에서 작은 검사를 자동 실행하는 보조 장치다.

권장 용도:

- 세션 시작 시 프로젝트 문맥 읽기 안내
- 컨텍스트 압축 후 핵심 상태 복구

Hook은 강제 보안장치가 아니다. 자동 pull·rebase·push나 모든 명령 차단에는 사용하지 않는다. 강제 검사는 프로젝트 검사 도구와 CI가 담당한다. Plugin은 Hook을 신뢰하거나 사용할 수 없는 환경에서도 동작해야 한다.

## 5. 협업 개념

### Context sync

모든 작업자가 알아야 하는 현재 사실을 갱신한다.

예:

- 제품 정책 확정
- API·DB 계약 변경
- 기능 제외
- PR 병합
- 통합 기준 commit 변경

### Checkpoint

작업 시작, 계약 변경, PR, 통합, 릴리스처럼 중요한 경계에서 실제 상태와 기록이 일치하는지 검사한다.

### Handoff

일상적인 작업 완료를 의미하지 않는다. 담당자가 실제로 바뀔 때만 사용한다.

예:

- 휴가·퇴사·역할 변경
- 중단된 작업을 다른 사람이 재개
- 큰 작업을 분리해 책임 이전
- 기존 담당자가 더 이상 해당 branch를 수정하지 않는 명시적 전환

Handoff에는 기존 담당자, 새 담당자, branch와 원격 상태, 완료 범위, 남은 작업, 검증 결과, 미확정 정책을 포함한다. 새 담당자가 상태를 확인하고 수락해야 완료된다.

## 6. Git과 submodule 방향

Git convention, branch model, commit·PR·merge policy, submodule 운영 정책은 기존 프로젝트의 규칙을 복사하지 않고 이 제품에서 처음부터 독립적으로 설계한다. 생성 대상 프로젝트를 분석해 적합한 정책을 권장할 수 있지만, 제품 자체의 기본 정책과 검사 규칙은 별도의 명세와 test로 관리한다.

- branch와 commit은 일반적인 Git convention을 사용한다.
- AI 사용 여부를 branch, commit, PR 이름에 넣지 않는다.
- 작업관리 도구 ID는 존재할 때만 선택적으로 연결한다.
- `ticket`, `slug`, `child repo`를 제품의 필수 개념으로 만들지 않는다.
- workspace 또는 submodule repo라는 표현을 사용한다.
- 기본 Git 전략은 protected main과 짧은 feature/fix/chore branch다.
- 장기 stabilization이나 다중 릴리스 유지가 필요할 때만 `release/<major.minor>` 유지 branch를 추가한다. `develop` branch는 기본으로 만들지 않는다.
- submodule pointer는 개별 개발 commit마다 갱신하지 않는다.
- 관련 workspace 변경이 병합되고 계약과 구현이 일치하는 통합 checkpoint에서 pointer를 갱신한다.
- 루트 worktree 안에서 초기화된 submodule을 병렬 작업의 기본 기반으로 삼지 않는다.
- 병렬 작업은 독립 clone 또는 개별 workspace repo worktree를 우선 고려한다.
- `.gitignore`는 캐시, secret, 임시 diff, 로컬 상태에만 사용하고 추적 파일 충돌 해결 수단으로 보지 않는다.

## 7. 계약과 데이터베이스

### 계약 변경 관리

이전 설계의 `evolve-contracts` 명칭은 사용하지 않는다. 사용자에게는 `계약 변경 관리` 또는 `contract change`라고 설명한다.

프론트엔드, 백엔드, 데이터베이스 중 한쪽에서 새로운 요구가 발견되면 임의 구현 전에 다음을 확인한다.

1. 제품 규칙 변경인지 구현 세부사항인지 구분
2. 영향을 받는 workspace와 사용자 흐름 확인
3. 호환성을 깨는지 확인
4. 계약 변경 제안
5. 사용자 또는 지정 검토자의 승인
6. 관련 작업의 계약 기준 갱신

### DBML과 dbdiagram

- Git에 저장된 DBML이 source of truth다.
- DBML은 도메인별 module로 분리할 수 있어야 한다.
- dbdiagram CLI로 push, pull, preview가 가능해야 한다.
- dbdiagram 웹에서 변경된 내용은 Git 원본에 즉시 덮어쓰지 않는다.
- 임시 위치로 pull하고 기준 DBML, Git 변경, 웹 변경을 비교한다.
- 의미가 달라진 경우 변경 이유를 사용자에게 질문한 뒤 계약 변경 절차로 진행한다.
- 공유 다이어그램 push는 검증된 통합 branch에서만 수행한다.

## 8. 작업관리 도구

특정 서비스를 강제하지 않는다.

프로젝트를 시작할 때 규모, 팀 구성, 오프라인 요구, 기존 도구를 확인한 뒤 적합한 작업관리 방식을 추천하고 사용자가 선택하게 한다.

선택 가능한 범주는 다음과 같다.

- 로컬 Git 기반
- GitHub Issues/Projects
- 별도 task database
- Jira·Linear 등 외부 서비스
- 사용자가 이미 사용하는 도구

도구와 관계없이 최소한 담당자, 상태, 영향 workspace, 계약 영향, 의존 관계를 표현할 수 있어야 한다.

## 9. 릴리스

릴리스는 두 개의 gate를 통과한다.

### AI 기술 준비 확인

- 전체 build와 test
- 계약 일치
- DB migration 검증
- 보안·secret 검사
- mock·임시 구현 잔존 검사
- workspace와 root pointer 일치
- 재현 가능한 검증 증거

### 사용자 확인

AI가 확인한 것과 동일한 release candidate commit 집합을 사용자가 직접 실행하고 주요 흐름을 검증한다.

최종 tag와 release는 사용자 승인이 있어야 생성한다.

## 10. 배포와 호환성

- 공개 GitHub marketplace repository를 우선 배포 경로로 사용한다.
- Codex 사용자는 Plugin을 한 번 설치한 뒤 자연어로 사용한다.
- Plugin 설치가 불가능한 환경은 표준 Agent Skills와 Markdown fallback을 사용한다.
- Skill 자동 발견과 도구 권한은 AI client마다 다를 수 있다.
- 핵심 프로젝트 상태는 특정 AI의 대화 기록이나 Plugin 내부 저장소에만 보관하지 않는다.

## 11. 현재 확정 수준

제품 목표, 핵심 상호작용, 구성 요소의 책임, 협업 용어, Git·submodule, 계약·DBML, CLI, Plugin, 보안, test, release와 배포 전략을 확정했다.

세부 기술 설계 검토 현황:

- [프로젝트 생명주기와 단계별 게이트](./01-project-lifecycle.md): 확정
- [생성되는 프로젝트 구조와 각 파일의 책임](./02-generated-project-structure.md): 확정
- [컨텍스트, 원본, 압축 복구 설계](./03-context-and-source-of-truth.md): 확정
- [Git, 협업, submodule, 충돌 정책](./04-git-collaboration-and-submodules.md): 확정
- [AI 행동, 승인, 안전 정책](./05-ai-action-and-approval-policy.md): 확정
- [외부 도구와 provider adapter 설계](./06-external-adapters.md): 확정
- [Cross-platform CLI, 검사기, 결과 schema 설계](./07-checker-cli-and-result-schema.md): 확정
- [Plugin, Skill, Hook, 설치, 보안 설계](./08-plugin-skills-installation-security.md): 확정
- [테스트, RC, Release, 운영 준비 기준](./09-test-release-and-production-readiness.md): 확정
- [제품 범위, source repository, 배포 청사진](./10-product-repository-and-distribution.md): 공개 이름을 제외하고 확정
- [전체 설계 교차 검토와 확정 기록](./11-cross-review-and-confirmation.md): 확정
- [실제 사용자 경험과 생성 결과 walkthrough](./12-user-experience-walkthrough.md): 확정

구현 계획도 별도로 확정했다. 다만 Go source, Plugin package, signed artifact, marketplace와 public release는 아직 생성하지 않았으므로 현재 상태를 “출시 완료”라고 부르지 않는다.

## 12. 구현 진행 순서

구현은 [production implementation plan](../superpowers/plans/2026-07-16-fullstack-orchestrator-production.md)의 13개 독립 검토 단위를 TDD로 실행한다. schema·result → context → approval/operation → Git/work → contract/DB/UI → project generation → Plugin/provider → RC → 문서·배포 순서다.

## 13. 확정한 결정

### 결정 1 — 프로덕션 출시 완료의 대표 사용자 여정

프로덕션 출시 전에 다음 두 사용자 여정이 모두 최종 release까지 성공해야 한다.

1. 처음 사용하는 사람이 빈 폴더에서 시작해 AI의 안내만으로 서비스 정의, 저장소와 submodule 구성, 기술 스택 선정, 전체 UI 기준선, 계약과 구현 경계, 백엔드 구현과 프론트엔드 연결, 통합 검증, 사용자 확인, 최종 release까지 완료한다.
2. 다른 사람이 기존 프로젝트를 clone한 뒤 프로젝트 문맥, Git·submodule·계약·작업 상태를 복구하고 미완료 작업을 이어서 최종 release까지 완료한다.

### 결정 2 — 오케스트레이션 저장소 생성 시점

핵심 사용자, 문제, 가치, 주요 흐름이 잡히고 서비스명 또는 안정적인 프로젝트·저장소 이름이 확정되면 AI가 오케스트레이션 Git 저장소 생성을 권장한다. 사용자가 승인한 뒤 생성한다.

### 결정 3 — 기존 프로젝트 구조 도입

이미 제품 코드가 존재하는 프로젝트에는 현재 repository 구조와 Git history를 보존한 채 하네스를 먼저 추가한다. submodule 전환은 이점이 명확한 경우에만 제안하고 사용자 승인 후 수행한다.

새 프로젝트는 오케스트레이션 root를 기준으로 시작하며, 독립적인 담당 경계나 배포 주기가 필요한 workspace는 필요성이 확정되는 즉시 별도 repository와 submodule로 추가한다.

서비스 발견 단계는 전체 제품 의도, 사용자 역할, 주요·예외·운영 흐름, 기능 범위, 핵심 정책을 정리하고 AI 완전성 검토와 사용자 승인을 통과한 뒤 아키텍처·기술 스택 기준선 단계로 전환한다.

### 결정 4 — 아키텍처·기술 스택과 전체 UI 기준선

제품 기준선 이후, 전체 UI를 만들기 전에 기능·비기능 요구를 근거로 solution architecture와 주요 기술 스택을 결정한다.

- AI는 기술 이름에 대한 취향부터 묻지 않고 제공 대상, realtime·offline·search·payment·media·AI 같은 기능, 데이터·성능·가용성, 보안·규제, 팀 경험, 예산, 운영 부담, 기존 시스템과 lock-in 조건을 먼저 정리한다.
- 공식 문서, release·security 상태, 라이선스와 유지보수 가능성을 확인해 대안 2~3개와 권장안을 제시한다.
- 제품 비용·위험·운영 방식이 달라지는 선택만 사용자에게 한 번에 하나씩 묻는다.
- 실패 비용이 큰 기술 가정은 time-boxed spike로 검증하지만 spike 코드를 production 결과물로 간주하지 않는다.
- major dependency는 기능 근거, 대안, 보안·라이선스, 운영 비용과 제거 전략을 기록한다.

백엔드 계약과 구현을 시작하기 전에 프로젝트에 존재하는 모든 사용자, 관리자, 운영자 역할의 전체 UI 기준선을 만든다.

정상 흐름뿐 아니라 로딩, 빈 화면, 오류, 권한 부족, 인증 만료, 재시도, 파괴적 작업 확인, 반응형 화면, 키보드·스크린리더 접근성 상태를 포함한다. 오프라인, 다국어, 실시간 연결 상태처럼 서비스 성격에 따라 필요한 상태는 제품 발견 결과에 따라 포함한다.

UI 기준선은 장식용 mockup이 아니다. 전체 제품 흐름과 데이터 요구를 검증하고 이후 API·event·DBML 계약의 입력으로 사용하는 실행 가능한 제품 기준선이다. 이 단계의 fixture는 아직 없는 API 형태를 최종 계약처럼 고정하지 않는다.

UI 기준선은 외부 디자인·목업·HTML·component code·별도 frontend project를 입력으로 받을 수 있다. source와 immutable version, 권한, license, reference·seed·canonical 지위를 기록하고 격리 검사와 전체 역할·상태 coverage를 통과한 뒤 작은 단위로 통합한다.

계약 승인 뒤 canonical schema에서 공유 type, client, server stub을 생성하고 module·package dependency, frontend data boundary, backend handler·domain·repository port·external adapter 경계를 정의한다. 모든 내부 class와 method를 interface로 만들지 않고 여러 작업자가 함께 의존하는 안정적인 경계만 컴파일 가능한 골격으로 먼저 통합한다.

### 자동 확정한 설계 원칙

- 전체 제품 정의와 완전성 검토를 통과하기 전에 UI 구현 단계로 넘어가지 않는다.
- 제품 요구에 맞는 아키텍처와 기술 스택, 고위험 기술 타당성을 확인한 뒤 전체 UI를 만든다.
- UI 기준선에서 확인된 데이터 요구를 기준으로 API·DB 계약을 만든다.
- 계약과 구현 경계·골격 승인 후 backend 구현과 frontend 연결을 수직 기능 단위로 진행한다.
- 기능 단위 integration과 test는 구현 중 계속 수행하고, 뒤의 통합 단계는 전체 workspace의 최종 통합 gate로 사용한다.
- 새 동작과 bug fix는 TDD를 필수로 적용한다. 문서, pure design asset, generated file, 폐기 spike, 의미 없는 formatting만 좁은 예외로 허용한다.
- 작업 전 path뿐 아니라 policy, contract, migration, UI flow, dependency, workspace의 의미 충돌을 확인하고 담당 범위와 merge 순서를 정한다.
- 공통 contract를 먼저 병합한 뒤 병렬 구현하거나 여러 backend를 먼저 병합한 뒤 frontend를 연결하는 등 상황에 맞는 충돌 회피 전략을 선택한다.
- 안전, 데이터 보존, 결정적 생성, 재현 가능한 검사, CI 강제는 별도 사용자 질문 없이 기본 요구사항으로 적용한다.
- 기술적으로 명확한 파일 분리, schema validation, cross-platform path 처리, idempotent command, 오류 복구, test matrix는 설계자가 최선안을 결정하고 기록한다.

### 결정 5 — 공개 범위와 라이선스 방향

현재 설계 중인 AI 풀스택 오케스트레이션 제품 전체를 공개 GitHub repository에서 오픈소스로 개발하고 배포한다.

- source, Agent Skills, Codex Plugin package, 프로젝트 검사 도구, templates, tests, documentation을 공개한다.
- 상업적 사용, 수정, 재배포를 허용한다.
- 기본 라이선스는 명시적인 특허 허여와 기여 조건을 제공하는 Apache License 2.0으로 정한다.
- 핵심 기능을 비공개 package로 분리하지 않는다.
- 사용자 프로젝트 내용, prompt, repository 상태를 외부로 전송하는 telemetry는 기본적으로 수집하지 않는다. 진단 정보 공유가 필요하면 사용자가 검토한 결과를 명시적으로 제공하는 방식으로 한다.

### 결정 6 — 제품 공개 이름의 방향

GitHub repository, Plugin marketplace, 명령, 문서에 사용할 제품명은 짧고 기억하기 쉬운 영문 브랜드명으로 정한다. 검색과 설치 화면에는 제품 역할이 바로 드러나는 설명형 부제를 함께 사용한다.

현재 폴더명 `fullstack-orchestrator`는 작업용 임시 이름이다. 최종 후보는 브랜드 방향을 확정한 뒤 GitHub, package registry, domain, 기존 소프트웨어와 상표 충돌 가능성을 조사해 제안한다.

### 결정 7 — 브랜드가 전달할 핵심 이미지

브랜드는 여러 사람, AI, repository가 하나의 제품 의도에 맞춰 정렬되고, 담당자·세션·도구가 바뀌어도 작업이 끊기지 않고 이어지는 이미지를 중심으로 한다.

이름은 단순한 AI 코드 생성기, task manager, monorepo tool, Git wrapper로 오해되지 않아야 한다.

### 보류 8 — 제품 이름과 브랜드 인상

제품 이름, 구체적인 브랜드 인상, 후보 조사와 최종 선정은 기능·배포·운영 설계를 완료한 뒤 진행한다. 그전까지 `fullstack-orchestrator`를 작업용 임시 폴더명으로만 사용한다.

### 방향 9 — AI client 호환 범위

공통 핵심은 특정 AI에 종속되지 않는 Agent Skills, 프로젝트 하네스, 검사 도구로 만든다. Codex, Claude Code, GitHub Copilot을 주요 호환 대상으로 삼고 다른 Agent Skills client도 표준 형식으로 사용할 수 있게 한다.

다만 실제 end-to-end 검증을 수행하지 않은 client를 `공식 검증 완료`로 표시하지 않는다. 자동 conformance test와 실제 client test, community issue evidence를 구분해 compatibility matrix에 공개한다.

### 결정 10 — AI client 지원 등급과 출시 후 Issue 운영

사용자가 모든 AI client를 직접 테스트할 책임을 지지 않는다.

- Codex는 자동 검사와 실제 end-to-end 사용자 여정을 모두 통과한 `검증 완료` 대상으로 출시한다.
- Claude Code와 GitHub Copilot은 공통 Agent Skills conformance와 프로젝트 검사 도구 테스트를 통과한 `표준 호환` 대상으로 시작한다.
- 실제 client별 재현 테스트와 community evidence가 확보되면 검증 완료로 승격한다.
- 다른 Agent Skills client는 community compatibility 대상으로 표시한다.
- 공개 compatibility matrix에 client, OS, 검증 일자, 검증 범위, 알려진 제한을 기록한다.
- GitHub Issue template은 client·version·OS·재현 절차·민감정보 제거 로그를 받도록 구성한다.
- Issue는 발견되지 않은 환경 차이를 수집하고 지원 범위를 넓히는 수단이며, 기본 기능·안전·데이터 보존 검사를 대신하지 않는다.

### 결정 11 — 중앙 서버와 Git 사용 정책

이 제품은 별도의 계정이나 중앙 서버를 필수로 운영하지 않는 local-first 도구로 만든다. 프로젝트 상태는 프로젝트 하네스와 사용자가 선택한 외부 도구에 저장한다.

Git은 도구를 실행하기 위한 절대 필수조건은 아니지만 협업 프로젝트에는 매우 강하게 권장한다.

- Git 없이도 서비스 발견, 제품 문서화, 하네스 초안, 단일 사용자 로컬 작업을 사용할 수 있다.
- Git이 없으면 원격 최신 상태, 변경 이력, branch 격리, PR, submodule pointer, 책임 이전, 충돌 복구를 신뢰성 있게 제공할 수 없음을 명확히 경고한다.
- 여러 사람이나 AI가 협업하는 프로젝트에서 Git을 사용하지 않겠다고 선택하면 제한되는 기능과 위험을 보여주고 명시적인 확인을 받는다.
- Git을 사용하면 공동 컨텍스트, 계약, 작업 연결, 검증 결과를 versioned source of truth로 관리한다.
- Git hosting provider는 강제하지 않는다. GitHub는 Plugin 배포와 Issue 운영의 기본 provider지만 생성된 프로젝트는 다른 Git hosting도 사용할 수 있다.

### 결정 12 — 작업관리 도구가 없을 때의 기본 동작

Git repository에 함께 저장되는 내장 작업관리 기능을 fallback으로 제공한다.

- 초기 서비스 발견, 개인 작업, offline 환경, 외부 도구 도입 전에도 작업·담당·의존 관계를 관리할 수 있다.
- 여러 사람이 협업하면 팀 규모, 기존 도구, 권한, 비용, offline 요구를 분석해 GitHub Issues/Projects, Jira, Linear, Beads 등 적절한 외부 도구를 적극 권장한다.
- 유료·무료·오픈소스 도구를 배제하지 않고 비용, 개인정보, 자동화, 운영 부담을 비교해 권장안을 제시한다.
- 사용자가 외부 도구를 선택하면 해당 도구를 담당자와 live status의 source of truth로 사용한다.
- 프로젝트 하네스에는 provider 설정, 외부 작업 ID, 관련 contract·workspace·PR 연결만 저장하고 상태를 중복 관리하지 않는다.
- 외부 도구 설치와 인증은 사용자 승인 후 진행한다.

### 결정 13 — 제품 언어와 문서 언어

영어를 canonical language로 사용하고 한국어를 동일한 수준으로 제공한다.

- source identifier, schema key, command, machine-readable error code는 영어로 고정한다.
- README, 설치·운영·문제 해결·기여 문서는 영어와 한국어를 함께 유지한다.
- CLI 메시지와 AI 안내는 사용자 설정 또는 system locale을 감지하고 영어로 fallback한다.
- 두 언어 문서의 구조와 의미가 달라지지 않도록 documentation parity test를 둔다.
- 생성되는 프로젝트 문서의 언어는 프로젝트 시작 시 사용자 대화 언어를 기본으로 하되, machine-readable state와 contract identifier는 영어를 유지한다.

### 남은 사용자 결정

- 공개 repository와 package를 만들기 직전 제품 이름 확정: 1개

### 완료된 기술 설계

공동 context model, Git/submodule 협업, AI 승인, 외부 adapter, CLI/result schema, Plugin/Skill/Hook, schema migration·보안, macOS·Windows test matrix와 release gate까지 설계 문서에 확정했다. 추가 기술 선택은 구현 중 새 사실이 현재 기준을 무효화할 때만 change 절차로 다시 연다.
