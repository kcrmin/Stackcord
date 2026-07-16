# 프로젝트 생명주기와 단계별 게이트

> 상태: 확정
>
> 마지막 갱신: 2026-07-16
>
> 이 문서는 신규 서비스 시작, 기존 프로젝트 도입, clone 후 작업 재개부터 최종 release와 운영까지의 공통 진행 규칙을 정의한다. 특정 프레임워크, 언어, 데이터베이스, 배포 사업자를 전제로 하지 않는다.

## 1. 설계 목표

이 제품의 생명주기는 사용자가 고정된 명령 목록을 외워서 실행하는 절차가 아니다.

사용자가 `새 프로젝트 시작해줘`, `지금 뭐 해야 해?`, `이 변경을 진행해줘`, `릴리스할 수 있어?`라고 말하면 AI가 실제 프로젝트 상태를 읽고 다음을 판단해야 한다.

- 현재 프로젝트가 어느 단계에 있는가
- 해당 단계가 완료되었다는 근거가 있는가
- 이전 결정의 변경으로 다시 검토해야 하는 결과물이 있는가
- 다른 사람이나 AI가 진행 중인 작업과 충돌하는가
- 지금 안전하게 실행할 수 있는 가장 작은 다음 작업은 무엇인가
- 어떤 작업은 자동으로 수행하고 어떤 결정은 사용자 승인을 받아야 하는가

따라서 생명주기는 문서가 존재하는지만 확인하는 선형 체크리스트가 아니라 **상태, 증거, 의존 관계, 승인 기록을 가진 상태 기계**로 구현한다.

## 2. 기본 원칙

### 2.1 대화가 아니라 프로젝트가 상태를 소유한다

원본 대화와 AI의 기억은 source of truth가 아니다. 대화에서 확정된 내용은 정규화된 프로젝트 지식, 계약, 작업 연결, 검증 결과로 저장한다.

대화가 압축되거나 AI·담당자·컴퓨터가 바뀌어도 저장소를 읽으면 다음 행동을 판단할 수 있어야 한다.

### 2.2 파일 생성은 단계 완료가 아니다

어떤 단계도 `파일이 존재한다`는 이유만으로 통과하지 않는다. 단계 완료에는 최소한 다음 네 가지가 필요하다.

1. 요구되는 결과물
2. 결과물을 검증한 기계적 또는 의미적 증거
3. 기준으로 삼은 Git commit 또는 내용 fingerprint
4. 필요한 경우 사용자나 지정 검토자의 승인

### 2.3 이미 통과한 단계도 다시 열릴 수 있다

제품 정책, UI 흐름, 계약, 보안 요구처럼 후속 결과물의 전제가 바뀌면 영향받은 단계를 `stale`로 표시한다. 변경 전 승인을 그대로 재사용하거나 조용히 다음 단계로 진행하지 않는다.

### 2.4 질문은 의사결정에만 사용한다

경로 처리, 원자적 파일 쓰기, schema validation, 안전한 Git 조회처럼 기술적으로 명백한 항목은 제품 기본값으로 적용하고 알린다.

사용자 의도, 제품 범위, 비용, 외부 공개, 데이터 손실 위험, 되돌리기 어려운 저장소 구조가 달라질 때만 한 번에 한 가지 결정을 요청한다.

### 2.5 병렬 작업은 허용하되 기준선은 공유한다

승인된 제품·아키텍처·UI·계약·구현 경계 기준선 안에서는 서로 독립적인 작업을 병렬로 진행할 수 있다. 각 작업은 담당 범위, 영향 workspace, 의존 계약, 기준 commit을 가져야 한다.

아직 통과하지 못한 상위 게이트를 우회해 production 구현이나 release로 병합할 수는 없다.

### 2.6 불확실한 Git 변경은 자동으로 하지 않는다

dirty tree, divergence, 공유 branch, submodule pointer 불일치가 있으면 자동 pull, rebase, stash, reset, force push로 해결하지 않는다. 먼저 읽기 전용으로 상태를 진단하고 보존할 작업과 선택지를 설명한다.

### 2.7 전체 범위는 먼저 확인하고 세부사항은 점진적으로 깊게 만든다

제품 기준선은 모든 역할·기능·예외·운영 흐름을 빠짐없이 다루되, 모든 내부 구현을 미리 설계한다는 뜻은 아니다. 협업과 후속 설계에 필요한 깊이까지 전체 범위를 먼저 정의하고, 기능 내부의 세부 알고리즘은 승인된 경계 안에서 구현하며 구체화한다.

### 2.8 기술 타당성, 통합, 테스트는 마지막까지 미루지 않는다

고위험 기술은 UI 기준선 전에 격리된 타당성 spike로 검증한다. 기능 구현은 실제 frontend와 backend를 연결하는 수직 단위로 진행하며 단위·계약·통합 검사를 함께 추가한다. 뒤의 통합과 프로덕션 강화 단계는 최초 통합과 최초 테스트가 아니라 전체 시스템 수준의 최종 게이트다.

### 2.9 동작 변경과 버그 수정은 test-first로 진행한다

새 동작, bug fix, contract, migration, 권한, UI interaction, infrastructure behavior는 실패하는 검사 또는 기존 동작을 고정하는 characterization test를 먼저 만든다. 문서, pure design asset, 결정적 generated file, production에 병합하지 않는 spike, 의미 없는 formatting만 명시적 예외로 허용한다.

TDD는 behavior와 regression 충돌을 탐지하지만 Git text conflict를 대신하지 않는다. 작업 전 scope claim과 충돌 검사를 함께 수행한다.

## 3. 상태 모델

### 3.1 단계 상태

각 단계는 다음 상태 중 하나를 갖는다.

| 상태 | 의미 |
|---|---|
| `not_started` | 아직 시작 조건을 충족하지 않았거나 작업을 시작하지 않음 |
| `in_progress` | 결과물을 만들거나 검토하는 중 |
| `blocked` | 명시된 외부 의존성 또는 사용자 결정 때문에 진행할 수 없음 |
| `ready_for_review` | 요구 결과물과 자동 검사는 준비됐고 승인 또는 검토를 기다림 |
| `approved` | 해당 fingerprint의 결과물과 증거가 게이트를 통과함 |
| `stale` | 한때 통과했지만 전제가 바뀌어 재검토가 필요함 |

`blocked`는 단순히 다음 작업이 어렵거나 오래 걸린다는 의미로 사용하지 않는다. 해제 조건과 책임자가 명확할 때만 기록한다.

### 3.2 검사 결과

각 검사는 다음 결과 중 하나를 돌려준다.

| 결과 | 의미 |
|---|---|
| `pass` | 요구조건을 만족함 |
| `warn` | 진행은 가능하지만 알려진 위험 또는 권장 개선이 있음 |
| `fail` | 게이트를 통과할 수 없음 |
| `unknown` | 도구, 권한, 정보 부족으로 판단할 수 없음 |

`unknown`을 `pass`로 간주하지 않는다. 필수 검사라면 게이트를 막고, 선택 검사라면 제한사항과 대체 증거를 기록한다.

### 3.3 게이트 증거

게이트 통과 기록은 최소한 다음 정보를 포함한다.

- 단계 식별자
- 검사 대상 결과물과 version 또는 fingerprint
- root와 관련 workspace의 commit SHA
- 실행한 검사와 결과
- 알려진 warning과 수용 근거
- 의미적 승인자와 승인 시점
- 통과 후 무효화 조건

정확한 저장 파일과 machine-readable schema는 공동 context model 설계에서 확정한다.

## 4. 진입 경로

### 4.1 신규 프로젝트 시작

빈 폴더나 아직 제품 저장소가 없는 상태에서 시작한다. 서비스 발견 결과로 안정적인 프로젝트 이름과 저장소 경계가 정해지기 전에는 임시 Git 저장소를 만들지 않는다.

### 4.2 기존 프로젝트 도입

이미 코드와 Git 이력이 있는 프로젝트에 오케스트레이션 하네스를 추가한다. 첫 실행은 읽기 전용 진단이며, 기존 구조를 재배치하거나 submodule로 바꾸지 않는다.

진단 결과와 도입 계획을 사용자가 승인한 뒤 하네스를 추가한다. 기존 증거가 충분한 단계는 다시 만들지 않고 인정하지만, 증거가 부족한 가장 이른 단계부터 보완한다.

### 4.3 clone 후 재개

다른 사람이 기존 오케스트레이션 프로젝트를 clone하고 작업을 이어간다. AI는 저장된 문맥만 읽는 것이 아니라 로컬 checkout과 원격, workspace, submodule, 진행 중인 작업, 제품·아키텍처·UI·계약·구현 경계 기준선을 비교한다.

로컬 상태가 저장된 기준과 다르면 차이를 먼저 설명하고 안전한 복구 또는 동기화 작업을 제안한다.

## 5. 전체 단계

```text
진입 진단
  → 서비스 발견
  → 프로젝트 초기화
  → 제품 기준선
  → 아키텍처·기술 스택 기준선
  → 전체 UI 기준선
  → 계약 기준선
  → 구현 경계·골격 기준선
  → 기능 구현과 연결
  → 통합
  → 프로덕션 강화
  → 릴리스 후보
  → 사용자 검증
  → 최종 릴리스
  → 운영과 다음 변경
```

| 단계 | 한 줄 요약 |
|---|---|
| 진입 진단 | 신규·기존·clone 재개 여부와 Git·workspace·로컬 변경 상태를 안전하게 파악한다. |
| 서비스 발견 | 핵심 사용자, 문제, 가치, 주요 흐름과 서비스의 중심 방향을 대화로 찾아낸다. |
| 프로젝트 초기화 | 이름이 확정되면 orchestration root와 하네스를 만들고 확정된 workspace를 구성한다. |
| 제품 기준선 | 모든 역할, 기능, 정책, 예외, 권한, 운영·복구 흐름과 비기능 요구를 정의한다. |
| 아키텍처·기술 스택 기준선 | 제품 요구를 근거로 구조와 기술 대안을 비교하고 고위험 가정을 검증한 뒤 선택한다. |
| 전체 UI 기준선 | 모든 역할과 정상·오류·권한 상태를 실제로 탐색 가능한 UI로 검증한다. |
| 계약 기준선 | UI에 필요한 API, event, 인증, 오류, 데이터 구조와 DBML을 확정한다. |
| 구현 경계·골격 기준선 | 병렬 개발 전 공유 type, module 경계, 생성 stub과 최소 기능 골격을 통합한다. |
| 기능 구현과 연결 | 같은 기준선에서 기능별 backend 구현과 frontend 연결을 수직 단위로 진행한다. |
| 통합 | 모든 workspace, 계약, submodule pointer와 전체 동작을 재현 가능한 상태로 고정한다. |
| 프로덕션 강화 | 보안, 성능, 접근성, 관측성, migration, backup·restore와 운영 준비를 검증한다. |
| 릴리스 후보 | 정확한 commit 집합과 artifact를 변경 불가능한 검증 대상으로 고정한다. |
| 사용자 검증 | 사용자가 AI가 검사한 것과 동일한 release candidate로 실제 흐름을 확인한다. |
| 최종 릴리스 | 승인된 release candidate를 변경 없이 tag, artifact, 문서와 함께 공개한다. |
| 운영과 다음 변경 | issue와 incident를 관리하고 변경 영향에 따라 필요한 이전 단계부터 다시 시작한다. |

기존 프로젝트는 별도의 축약 코스를 사용하지 않는다. 현재 증거로 통과 가능한 단계를 순서대로 판정한 뒤, 가장 이른 미충족 또는 `stale` 단계에서 위 흐름에 합류한다.

이 순서는 의존 게이트의 순서이지 한 번 지나가면 돌아갈 수 없는 waterfall 일정이 아니다. 각 단계 안에서는 작은 변경과 병렬 작업을 지속적으로 통합하며, 새 사실이 발견되면 영향받는 이전 단계만 다시 연다.

## 6. 단계별 정의

### P00. 진입 진단

#### 목적

신규 시작, 기존 프로젝트 도입, clone 후 재개 중 어떤 경로인지 판단하고 안전하게 읽을 수 있는 범위를 확정한다.

#### 주요 동작

- 운영체제, shell, AI client, locale 확인
- 현재 디렉터리와 상위 디렉터리의 Git 경계 확인
- 기존 하네스와 제품 코드 존재 여부 확인
- 기존 Git의 branch, upstream, dirty, ahead, behind, divergence 확인
- workspace와 submodule 구성 및 초기화 상태 확인
- 외부 task provider와 필요한 인증의 존재 여부만 확인
- secret이나 untrusted repository 명령을 실행하지 않고 신뢰 경계 확인

#### 결과물

- 진입 유형
- 읽기 전용 상태 보고서
- 즉시 보존해야 할 로컬 변경 또는 충돌 위험
- 다음 단계 권장안

#### 종료 게이트

- 작업 손실 없이 읽기 가능한 상태가 확인됨
- 신규, 도입, 재개 경로가 결정됨
- 쓰기 작업 전에 필요한 사용자 승인 항목이 식별됨

### P10. 서비스 발견

#### 목적

만들려는 서비스의 사용자, 문제, 가치, 전체 흐름과 제약을 구현 가능한 제품 지식으로 정규화한다.

#### 주요 동작

- 한 번에 하나의 적응형 질문
- 객관식에는 권장안, 기타, 잘 모르겠음 제공
- 확정, 가설, 미정, 제외, 후속 고려사항 분리
- 사용자 역할, 주요 흐름, 예외 흐름, 운영 흐름 탐색
- 보안, 개인정보, 권한, 규제, 접근성, 국제화, 실패 복구, 확장성 관점 제안
- 앞뒤 답변의 모순과 빠진 의사결정 탐지
- 원본 답변이 아닌 중립적이고 검증 가능한 제품 지식으로 정리

#### 결과물

- 핵심 사용자와 해결할 문제
- 제품 가치와 성공 기준
- 주요 사용자 여정과 역할
- 알려진 제약, 가설, 미결정 목록
- 안정적인 프로젝트·저장소 이름 후보

#### 종료 게이트

- 핵심 사용자, 문제, 가치, 주요 여정이 서로 모순 없이 설명됨
- 프로젝트를 식별할 안정적인 이름이 승인됨
- 지금 저장소를 만들 만큼 제품 경계가 안정적이라는 AI 검토를 통과함
- 사용자가 발견 결과를 승인함

서비스의 모든 세부 기능이 이 단계에서 완성될 필요는 없다. 다만 프로젝트 구조를 만들기 위한 중심축이 흔들리지 않아야 한다.

### P20. 프로젝트 초기화

#### 목적

합의된 제품 경계에 맞춰 오케스트레이션 root, 하네스, Git, workspace 전략, 협업 기준을 안전하게 만든다.

#### 주요 동작

- 신규 프로젝트는 사용자 승인 후 root 저장소 초기화
- 기존 프로젝트는 구조와 history를 보존하고 하네스만 추가
- Git 미사용 시 제한 기능과 협업 위험을 명확히 경고
- frontend, backend, infrastructure 같은 이름을 고정하지 않고 실제 책임 경계 분석
- 독립 배포 주기, 권한, 소유권, 재사용 경계가 이미 명확한 workspace만 별도 repository와 submodule로 제안
- 기술 구조 검토가 필요한 workspace는 후보만 기록하고 P35에서 경계를 확정하는 즉시 승인 후 생성
- task provider 선택 또는 내장 Git 기반 fallback 설정
- canonical language와 생성 문서 언어 설정

#### 결과물

- 오케스트레이션 하네스
- root와 현재 확정된 workspace topology 및 추가 검토가 필요한 후보
- Git remote와 보호 정책 계획
- 선택된 task source of truth
- 프로젝트별 기본 정책과 검사 설정

#### 종료 게이트

- 생성 결과가 같은 입력에서 결정적으로 재현됨
- 기존 파일과 Git history가 보존됨
- root와 현재 구성된 workspace 책임 경계가 설명 가능함
- 현재 구성된 repository와 submodule remote 접근이 검증됨
- 다른 사용자가 clone하여 같은 구조를 복원할 수 있음
- 사용자에게 구조와 운영 비용이 설명되고 승인됨

### P30. 제품 기준선

#### 목적

UI와 production 구현의 입력이 될 전체 서비스 의도와 정책을 빠짐없이 정의한다.

#### 주요 동작

- 모든 사용자, 관리자, 운영자 역할 정의
- 전체 기능 목록과 기능 간 의존 관계 정의
- 정상, 예외, 취소, 재시도, 복구, 운영 흐름 정의
- 인증, 권한, 데이터 보존·삭제, 감사, 알림 정책 정의
- 핵심 domain 용어, entity lifecycle과 상태 전이를 물리 database schema와 분리해 정의
- 성능, 가용성, 보안, 접근성, 개인정보, 관측성 요구 정의
- 초기 release에 포함할 것과 명시적으로 제외할 것 구분
- AI가 놓치기 쉬운 관점을 제안하고 완전성·모순 검토 수행

#### 결과물

- 정규화된 제품 명세
- 역할과 권한 모델
- 전체 기능·흐름 inventory
- business rule과 비기능 요구
- traceability 가능한 acceptance criteria

#### 종료 게이트

- 모든 역할의 시작부터 종료까지 흐름이 정의됨
- 정상 상태뿐 아니라 실패·운영·권한 상태가 정의됨
- `미정` 항목이 UI 기준선 제작을 방해하지 않거나 명시적으로 격리됨
- 기능과 정책 사이의 모순 검사가 통과함
- AI 완전성 검토와 사용자 승인을 모두 받음

이 게이트 전에는 production UI 구현으로 넘어가지 않는다. 아이디어 검증용 throwaway spike는 가능하지만 제품 결과물이나 기준선으로 병합하지 않는다.

### P35. 아키텍처·기술 스택 기준선

#### 목적

제품 기능과 비기능 요구를 실제로 만족할 수 있는 구조와 기술을 근거 기반으로 선택하고, 전체 UI와 각 workspace가 사용할 production 방향을 확정한다.

#### 주요 동작

- web, mobile, desktop, embedded 등 제공 대상과 배포 형태 확인
- realtime, offline, search, payment, media, geospatial, AI 등 기능별 기술 능력 도출
- 트래픽, 데이터 규모, latency, availability, consistency, recovery 목표 확인
- 보안, 개인정보, 규제, data residency, 감사 요구 확인
- 팀 경험, 채용 가능성, 유지보수 기간, 운영 인력 확인
- 예산, hosting 제약, vendor lock-in, 기존 시스템 연동 조건 확인
- 공식 문서, release·security 상태, 라이선스를 조사해 2~3개 대안 비교
- 기능 요구 → 기술 능력 → 후보 → 선택 근거를 traceability로 연결
- 실패 비용이 큰 기술은 time-boxed, 격리된 feasibility spike로 검증
- major 기술과 workspace topology를 architecture decision record로 확정
- architecture에서 추가로 확인된 workspace repository와 submodule을 사용자 승인 후 구성

AI는 사용자에게 기술 이름을 나열해 취향만 묻지 않는다. 제품 비용·위험·운영 방식이 달라지는 질문만 한 번에 하나씩 묻고, 나머지는 근거와 함께 권장안을 제시한다.

#### 결과물

- 전체 solution architecture와 workspace 책임 경계
- 확정된 root·repository·submodule topology
- frontend, backend, data, messaging, deployment, observability 기술 기준선
- 선택·기각 대안과 근거를 담은 architecture decision record
- 기능별로 필요한 주요 dependency와 도입 이유
- feasibility spike 결과와 폐기 여부
- 지원 version, upgrade, 교체 가능성, 알려진 운영 비용

#### 종료 게이트

- 모든 핵심 기능과 비기능 요구에 구현 경로가 있고 위험도에 맞는 타당성 증거가 있음
- 고위험 가정이 증거 없이 남아 있지 않음
- major dependency의 보안, 라이선스, 유지보수 상태가 검토됨
- workspace·repository·submodule 경계가 기술 구조와 일치함
- 사용자가 비용·lock-in·운영 부담을 포함한 주요 선택을 승인함
- 새로 구성된 workspace와 submodule을 다른 사용자가 clone하여 복원할 수 있음
- spike 코드는 production 기준선에 섞이지 않고 폐기 또는 정식 재구현 대상으로 표시됨

새 dependency는 이후에도 추가할 수 있지만, 어떤 기능 때문에 필요한지, 기존 기술로 해결할 수 없는지, 보안·라이선스·운영 부담과 제거 전략이 무엇인지 기록해야 한다. 제품 의미나 주요 운영 비용이 달라지지 않는 보조 dependency는 AI가 검사 후 선택하고 사용자에게 결과만 알린다.

### P40. 전체 UI 기준선

#### 목적

제품 명세 전체를 실행 가능한 UI로 표현해 사용자 흐름과 데이터 요구를 먼저 검증한다.

#### 주요 동작

- 모든 사용자·관리자·운영자 흐름 구현
- 외부 디자인·목업·코드 입력의 source, version, 권한, license와 reference·seed·canonical 지위 등록
- 외부 UI를 격리 검사하고 제품 policy·journey·필수 UI state와 coverage 비교
- 사용자 시나리오와 데이터 요구를 표현하는 mock data 또는 fixture 사용
- loading, empty, error, permission denied, auth expiry, retry 상태 구현
- 파괴적 작업 확인과 실행 취소·복구 가능성 표현
- 반응형, 키보드, screen reader, focus, contrast 검증
- 필요한 경우 offline, realtime reconnect, 다국어, timezone 상태 구현
- 화면 목록만이 아니라 화면 간 전체 탐색과 작업 완료 흐름 검증
- 역할·domain별 작은 변경으로 나눠 만들고 지속적으로 하나의 UI 기준선에 통합

#### 결과물

- 실행 가능한 전체 UI 기준선
- 화면·상태·행동 inventory
- 각 행동에 필요한 데이터와 command/query 요구
- UI acceptance와 접근성 검사 결과

#### 종료 게이트

- P30의 모든 역할·기능·예외가 UI 또는 명시적 non-UI 흐름에 연결됨
- 사용자 여정이 backend 없이도 mock data로 끝까지 실행됨
- 임시 UI가 아니라 이후 production frontend의 기준으로 사용할 품질을 가짐
- 접근성·반응형 필수 검사가 통과함
- 사용자가 전체 흐름과 정책 표현을 승인함

P40 승인은 production backend 기능 구현과 실제 frontend 연결의 필수조건이지만 충분조건은 아니다. P50 계약과 P55 구현 경계·골격까지 승인된 뒤 P60을 시작한다.

P40의 fixture는 아직 확정되지 않은 API 형태를 최종 계약처럼 고정하지 않는다. P50에서 계약이 승인되면 canonical schema로 typed mock, client, server stub을 다시 생성하고 UI 행동이 그대로 유지되는지 검사한다.

전체 UI를 먼저 만든다는 것은 하나의 대형 branch에서 모든 화면을 한 번에 만든다는 뜻이 아니다. 역할·domain별로 병렬화할 수 있지만 P30 전체 범위를 하나의 실행 가능한 기준선으로 통합·승인하기 전에는 P60 production backend 구현을 시작하지 않는다. 이후 의미 있는 UI 변경은 P40과 영향받는 후속 단계를 다시 연다.

### P50. 계약 기준선

#### 목적

승인된 UI와 제품 규칙을 frontend, backend, database, 외부 시스템이 함께 구현할 명시적 계약으로 바꾼다.

#### 주요 동작

- API, event, job, file, authentication, authorization 계약 정의
- 오류 code와 재시도·idempotency 규칙 정의
- DBML로 데이터 구조, 제약, 관계, index 의도 정의
- migration, seed, retention, deletion, audit 전략 정의
- dbdiagram CLI로 검증 가능한 diagram 제공
- 계약 versioning과 breaking change 기준 정의
- 제품 명세 → UI 행동 → 계약 항목 traceability 검사
- canonical schema에서 생성할 client·server type과 mock 경계 정의

#### 결과물

- versioned API·event·data 계약
- canonical DBML과 diagram 검토 결과
- migration 및 호환성 계획
- contract baseline fingerprint
- workspace별 구현 책임과 의존 계약

#### 종료 게이트

- UI의 모든 실제 데이터 요구가 계약에 연결됨
- 인증·권한·오류·동시성·재시도 의미가 정의됨
- DBML validation과 diagram 검토가 통과함
- breaking change와 migration 위험이 해결되거나 승인됨
- 관련 workspace 검토자와 사용자가 계약 의미를 승인함

dbdiagram 웹에서 직접 바뀐 내용은 임시 위치로 pull하여 Git 기준과 비교한다. 의미가 달라졌다면 변경 이유를 확인하고 계약 변경 절차를 통과하기 전까지 canonical DBML에 덮어쓰지 않는다.

### P55. 구현 경계·골격 기준선

#### 목적

여러 작업자가 기능 구현을 시작하기 전에 자주 충돌하는 공통 경계와 dependency 방향을 확정하고, 각 기능이 독립적으로 구현될 수 있는 최소한의 컴파일 가능한 골격을 만든다.

#### 주요 동작

- workspace 내부 module·package 책임과 허용 dependency 방향 정의
- canonical API·event schema에서 type, client, handler stub 생성
- 생성 파일과 직접 작성할 extension point를 분리하고 생성 파일 수동 편집 금지
- frontend data gateway·state boundary와 backend handler·domain service·repository port·external adapter 경계 정의
- 인증 context, 권한 검사, 오류, transaction, idempotency 공통 규격 연결
- 기능별 directory·package와 test harness의 최소 골격 생성
- 공유 경계별 소유자와 contract change 절차 연결
- 골격 자체의 build, type, dependency-boundary, smoke test 수행

#### 결과물

- module·package dependency map
- schema에서 생성된 공유 type, client, server stub
- frontend·backend의 안정적인 협업 경계
- 컴파일 또는 이에 준하는 검사를 통과하는 기능 골격
- 경계별 책임자와 변경 영향 연결

#### 종료 게이트

- 서로 다른 작업자가 같은 공통 파일을 반복 수정하지 않고 기능을 시작할 수 있음
- 생성 artifact가 canonical contract에서 결정적으로 재생성됨
- workspace 전체 build 또는 type·schema 검사가 통과함
- 순환 dependency와 금지된 module 접근이 없음
- 각 기능의 시작점, 책임 범위, test 위치가 명확함
- 경계 변경이 필요한 경우 사용할 contract change 절차가 연결됨

모든 class와 method를 interface로 미리 만들지 않는다. 다른 module·workspace가 의존하는 경계, 교체 가능한 외부 시스템 경계, 테스트 격리가 필요한 경계만 먼저 고정한다. 단일 구현 내부의 세부 구조는 기능 작업에서 결정한다. 기준선은 하나의 장기 대형 branch로 유지하지 않고 짧은 foundation 변경으로 통합한 뒤 기능 branch가 같은 fingerprint에서 시작한다.

바로 기능 branch를 시작하는 방식은 빠르지만 공유 type과 공통 구조가 작업 중 계속 바뀌어 충돌이 커지므로 채택하지 않는다. 모든 내부 interface를 먼저 만드는 방식은 사용되지 않는 추상화와 변경 비용을 만들므로 채택하지 않는다. 공유되는 안정적 경계와 생성 가능한 골격만 먼저 통합하는 방식을 기본으로 한다.

### P60. 기능 구현과 연결

#### 목적

승인된 제품, 아키텍처, UI, 계약, 구현 경계 기준선을 실제 backend 동작과 production frontend 연결로 완성한다.

#### 주요 동작

- 기능을 사용자 가치가 끝까지 동작하는 수직 단위로 나눔
- 각 작업에 담당자, 영향 workspace, 기준 계약, 의존 작업 연결
- 구현 전에 실패하는 test 또는 characterization test와 관련 scenario·contract를 기록
- backend 기능과 migration 구현
- UI mock을 실제 계약 호출로 교체
- red, green, refactor 순서로 단위, component, contract, integration test와 구현 진행
- 작은 수직 단위마다 실제 구성 요소를 연결하고 지속적으로 통합
- 작업 중 새 제품 요구가 발견되면 즉흥 구현하지 않고 영향 단계 재개방

#### 결과물

- production 구현
- 실제 frontend-backend 연결
- migration과 rollback 구현
- 기능별 검증 증거와 traceability

#### 종료 게이트

- 구현이 승인된 계약과 일치함
- 해당 기능의 UI mock·임시 분기·placeholder가 제거됨
- 정상·오류·권한·재시도 경로 테스트가 통과함
- 동작 변경에 유효한 TDD evidence가 있고 허용되지 않은 예외가 없음
- migration과 rollback이 격리 환경에서 검증됨
- 관련 작업과 PR의 기준 commit이 최신임

P40, P50, P55 기준선은 먼저 승인하지만, P60 내부 구현은 독립 기능 단위로 병렬 진행할 수 있다. architecture·contract·boundary를 바꾸지 않는 기능 내부 설계는 각 담당자가 결정하며, 공유 기준 변경이 필요하면 영향 단계를 다시 연다.

### P70. 통합

#### 목적

기능 개발 중 지속적으로 연결된 각 repository와 workspace 결과를 하나의 재현 가능한 전체 제품 상태로 고정한다.

#### 주요 동작

- 관련 PR 병합 상태와 commit reachability 확인
- root와 workspace의 계약 fingerprint 일치 확인
- submodule checkout과 root pointer 비교
- 관련 workspace 병합 후 통합 checkpoint에서만 pointer 갱신
- clean clone에서 전체 환경 복원과 build 수행
- 실제 서비스 경계 간 integration과 end-to-end 흐름 검증

#### 결과물

- 통합된 root와 workspace commit 집합
- 갱신된 submodule pointer
- clean-clone 재현 결과
- 통합·E2E 검사 결과

#### 종료 게이트

- 관련 변경이 보호된 통합 branch에서 도달 가능함
- dirty, detached, unpublished commit, pointer mismatch가 없음
- 모든 계약 consumer와 provider가 같은 기준선을 사용함
- clean clone에서 설치·build·test가 재현됨
- 주요 사용자 여정이 실제 구성 요소 사이에서 통과함

이 단계는 최초 integration을 수행하는 단계가 아니다. 기능 단위 integration은 P60에서 계속 수행하며, P70은 모든 기능과 workspace의 full-system integration, clean-clone 재현성, submodule pointer를 최종 확인하는 단계다.

### P80. 프로덕션 강화

#### 목적

기능 완성을 실제 운영 가능한 서비스 품질로 강화한다.

#### 주요 동작

- 전체 regression과 cross-platform 검사
- 보안, dependency, license, secret 검사
- 성능·부하·자원 한도 검증
- accessibility와 국제화 검증
- logging, metrics, tracing, alerting, runbook 검증
- backup, restore, migration failure, rollback, disaster recovery 연습
- 설치, 운영, 문제 해결, 개인정보, 사용자 문서 검증
- mock, debug flag, temporary bypass, hard-coded secret 잔존 검사

#### 결과물

- 강화 검사 보고서와 재현 명령
- 보안·성능·접근성 증거
- 운영·복구 runbook
- 알려진 제한과 수용 여부

#### 종료 게이트

- 필수 검사에 `fail`이나 미해결 `unknown`이 없음
- 높은 위험의 취약점과 데이터 손실 가능성이 없음
- 복구·rollback 절차가 실제로 실행 검증됨
- 운영자와 사용자가 필요한 문서를 이용할 수 있음
- 남은 warning마다 소유자, 영향, 수용 근거가 있음

### P90. 릴리스 후보 고정

#### 목적

검증할 정확한 제품 상태를 바뀌지 않는 release candidate로 고정하고 AI 기술 준비 확인을 수행한다.

#### 주요 동작

- root, workspace, submodule, contract commit 집합 고정
- 동일 환경에서 전체 build와 test 재실행
- artifact provenance, checksum, dependency lock 확인
- 설치·업데이트·제거·rollback 검증
- release note와 migration guide 생성
- 미완료 작업, 열린 blocker, 변경된 기준선 재검사

#### 결과물

- 고유한 RC 식별자와 commit 집합
- 변경 불가능한 검증 증거 bundle
- 설치 가능한 release artifact
- release note, upgrade, rollback 문서

#### 종료 게이트

- AI 기술 준비 확인의 모든 필수 항목이 통과함
- 검증 증거가 RC commit과 정확히 연결됨
- RC 생성 후 제품 코드는 변경되지 않음
- 사용자 검증에 전달할 실행 절차가 재현 가능함

RC 이후 코드나 계약이 바뀌면 기존 RC를 수정하지 않고 폐기한 뒤 새 RC를 만든다.

### P100. 사용자 검증

#### 목적

사용자가 AI가 검사한 것과 동일한 RC를 실제 목적과 사용 환경에 맞게 검증한다.

#### 주요 동작

- 사용자 또는 지정 승인자가 동일 RC checksum 확인
- 주요 사용자·관리자·운영자 여정 실행
- 데이터, 권한, 오류, 배포·복구 기대 확인
- 발견 사항을 제품·아키텍처·UI·계약·구현 경계·기능·운영 단계 중 원인 단계에 연결

#### 결과물

- 사용자 검증 기록
- 통과·반려 결정
- 반려 시 원인 단계와 후속 작업

#### 종료 게이트

- 승인자가 동일한 RC를 검증했음
- release를 막는 발견 사항이 없음
- 알려진 제한을 사용자가 명시적으로 수용함
- 최종 release 생성 승인을 받음

반려되면 해당 원인 단계와 그 후속 단계를 다시 열고, 변경 후 P90부터 새 RC를 검증한다.

### P110. 최종 릴리스

#### 목적

승인된 RC를 변경 없이 추적 가능한 공식 release로 공개한다.

#### 주요 동작

- 보호된 release commit 확인
- 서명 또는 보호된 tag 생성
- 검증된 artifact와 checksum 게시
- release note, 설치, 업데이트, 제거, migration 문서 게시
- 지원 범위와 compatibility matrix 갱신
- 공개 후 rollback·hotfix 경로 확인

#### 결과물

- immutable release tag
- 공개 artifact와 문서
- release evidence와 승인 기록
- 지원·issue 접수 경로

#### 종료 게이트

- 공개 artifact가 승인된 RC와 byte 또는 provenance 기준으로 일치함
- tag와 관련 commit이 원격에서 도달 가능하고 보호됨
- 신규 사용자가 문서만으로 설치할 수 있음
- 운영·지원·보안 신고 경로가 공개됨

### P120. 운영과 다음 변경

#### 목적

release 이후 발견 사항과 다음 제품 변경을 현재 문맥과 연결해 안전하게 발전시킨다.

#### 주요 동작

- issue, incident, feedback, compatibility evidence 수집
- hotfix와 일반 변경 구분
- 변경이 영향을 주는 가장 이른 생명주기 단계 판정
- 지원 중인 release와 migration 정책 유지
- 새 clone과 새 담당자가 현재 상태를 복원할 수 있는지 지속 검사

#### 결과물

- 운영 중인 release 목록
- issue와 변경 제안
- incident와 사후 조치
- 다음 작업의 진입 단계와 영향 범위

운영은 생명주기의 끝이 아니라 다음 변경의 시작점이다.

## 7. 기존 프로젝트 도입 판정

기존 프로젝트에는 다음 절차를 적용한다.

1. **읽기 전용 inventory**: repository topology, Git history, branch 보호, 로컬 변경, workspace, build, test, 문서, 계약, task provider를 조사한다.
2. **도입 보고서**: 보존할 구조, 발견된 위험, 하네스가 추가할 파일, 충돌 가능성, 선택적 개선안을 분리한다.
3. **사용자 승인**: 하네스 추가처럼 실제 파일을 바꾸기 전에 변경 범위를 승인받는다.
4. **증거 기반 단계 판정**: P10부터 순서대로 결과물과 검증 증거를 확인한다.
5. **가장 이른 gap에서 재개**: 뒤 단계에 코드가 있어도 앞 단계의 중요한 의미가 없으면 그 gap을 먼저 보완한다.
6. **비파괴 도입**: 승인 없는 repository 분리, submodule 변환, history rewrite, 강제 formatting은 하지 않는다.

예를 들어 frontend와 backend가 이미 동작해도 권한 정책과 데이터 계약이 문서화되지 않았다면 코드를 폐기하지 않는다. 현재 동작을 증거로 제품·아키텍처·UI·계약·구현 경계 기준선을 역으로 정리하고 승인한 뒤 구현과의 차이를 검사한다.

## 8. clone 후 재개 판정

다른 사용자가 프로젝트를 clone했을 때 AI는 다음 순서로 작업한다.

1. root와 workspace의 필수 원격 접근 확인
2. submodule 등록, 초기화, checkout 상태 확인
3. 로컬 branch와 upstream, ahead·behind·divergence 확인
4. dirty 파일, 미게시 commit, detached HEAD 확인
5. 저장된 제품·아키텍처·UI·계약·구현 경계 기준선과 현재 fingerprint 비교
6. 외부 task provider의 담당·상태와 로컬 연결 정보 확인
7. 다른 작업자의 활성 작업과 겹치는 파일·계약·workspace 확인
8. 가장 이른 미충족 또는 `stale` 게이트와 실행 가능한 다음 작업 제안

안전한 fast-forward가 가능한 경우에도 수행 내용을 먼저 알린다. 작업 손실 또는 의미 충돌 가능성이 있으면 자동 동기화하지 않고 한 가지 결정을 요청한다.

## 9. 변경 영향과 단계 무효화

변경을 감지하면 다음 기본 영향 규칙을 적용한다.

| 변경 | 기본적으로 다시 검토할 단계 |
|---|---|
| 핵심 사용자·문제·가치 변경 | P10 이후 전체 |
| 역할·권한·제품 정책·기능 범위 변경 | P30 이후 전체 |
| 주요 기술 스택·solution architecture 변경 | 영향받는 P35 이후 전체 |
| 사용자 흐름 또는 UI 상태 변경 | P40 이후 전체 |
| API·DB·event·auth 계약 변경 | 영향받는 P50, P55 이후 전체 |
| module 경계·공유 type·dependency 방향 변경 | 영향받는 P55 이후 전체 |
| 계약을 따르는 내부 구현만 변경 | 영향받는 P60, P70, P80, RC 이후 |
| 운영·보안·성능 요구 변경 | 영향받는 P30, P80, RC 이후 |
| build·배포·workspace topology 변경 | P20, P70, P80, RC 이후 |
| 문구나 문서만 의미 변화 없이 수정 | 영향받는 문서 검사만 |

실제 무효화 범위는 traceability graph로 좁힐 수 있지만, 근거 없이 기본 범위보다 축소하지 않는다.

변경 제안은 다음 순서로 처리한다.

1. 제품 의미 변경인지 구현 세부사항인지 분류
2. 영향받는 결과물, workspace, 진행 중 작업, release 식별
3. 변경 전후 차이와 호환성 위험 설명
4. 필요한 의미적 승인 획득
5. 영향 단계만 `stale`로 전환
6. 결과물과 검사를 갱신한 뒤 새 fingerprint로 재승인

## 10. `지금 뭐 해야 해?` 판단 규칙

AI는 사용자가 모든 단계를 직접 확인하게 하지 않는다. 다음 순서로 한 가지 실행 가능한 권장 작업을 만든다.

1. 읽기 전용 project refresh 수행
2. 데이터 손실 가능성이 있는 Git·workspace 이상을 최우선 처리
3. `stale`, `blocked`, `fail`, `unknown` 상태 확인
4. 의존 관계상 가장 이른 미충족 게이트 선택
5. 이미 담당자가 있는 작업과 범위 충돌 제외
6. 현재 사용자가 시작할 수 있는 가장 작은 가치 단위 선택
7. 무엇을, 왜 지금, 어디에, 어떤 완료 조건으로 할지 설명
8. 안전한 작업은 실행하고 제품 결정 또는 위험 작업만 승인 요청

권장 결과는 긴 대시보드가 아니라 다음처럼 행동 중심으로 제공한다.

> 현재 제품 기준선은 승인되었지만 UI 기준선의 운영자 권한 오류 흐름이 비어 있어 계약 설계를 시작할 수 없습니다. 기존 작업과 겹치지 않습니다. 이 흐름을 먼저 정의하고 UI에 추가하겠습니다.

## 11. Checkpoint 정책

checkpoint는 모든 작은 commit마다 만들지 않는다. 다음 의미 있는 경계에서 수행한다.

- 작업 시작 전: 기준 branch·commit·계약과 충돌 확인
- 제품·아키텍처·UI·계약·구현 경계 승인 시: fingerprint와 승인 기록 고정
- 계약 변경 시: consumer·provider 영향과 호환성 확인
- PR 준비 시: 작업 범위, 검사, 문서, task 연결 확인
- workspace 병합 시: commit reachability와 계약 일치 확인
- root 통합 시: submodule pointer 갱신과 clean-clone 검사
- handoff 시: 담당 변경과 미완료 상태 확인
- RC 생성 시: 전체 commit 집합과 evidence 고정

context sync는 공동 사실이 바뀔 때 수행하고, checkpoint는 기록과 실제 상태의 일치를 검사할 때 수행한다.

## 12. 병렬 협업 규칙

- 한 작업에는 한 명의 책임자와 명확한 영향 범위를 둔다.
- 작업 시작 전에 path, module, policy, scenario, contract, migration, UI flow, dependency, workspace 범위를 계산한다.
- active task·claim·branch·PR과 겹치면 구현 전에 위험을 알리고 담당 범위와 merge 순서를 합의한다.
- 공통 contract를 먼저 병합한 뒤 병렬 구현하거나, 여러 backend를 먼저 병합한 뒤 frontend를 연결하거나, frontend를 generated client·mock server로 먼저 격리하는 전략을 상황에 맞게 선택한다.
- 실제 변경 범위가 claim보다 넓어지면 즉시 재검사하고 관련 작업자에게 알린다.
- 병렬 작업은 독립 clone 또는 workspace repository별 worktree를 우선한다.
- 같은 작업공간의 추적 파일을 숨기는 `.gitignore`를 충돌 회피 수단으로 사용하지 않는다.
- 공통 제품·계약 파일을 바꾸는 작업은 변경 의도를 먼저 등록하고 영향 작업에 알린다.
- canonical schema에서 생성된 type·client·stub은 직접 수정하지 않고 생성 원본을 변경한다.
- 구현 경계 기준선은 짧게 통합하고 모든 병렬 기능 작업은 동일한 baseline fingerprint에서 시작한다.
- branch·commit·PR 이름에는 AI 사용 여부를 넣지 않고 일반 Git convention을 따른다.
- submodule pointer는 각 workspace의 개별 commit마다 바꾸지 않고 관련 변경이 병합된 통합 checkpoint에서 갱신한다.
- routine 완료는 handoff가 아니다. 실제 책임자가 바뀔 때만 기존 담당자와 새 담당자가 상태를 확인한다.

구체적인 branch, PR, merge, conflict resolution 정책은 독립 Git 설계 문서에서 확정한다.

## 13. 실패와 복구 원칙

- 모든 생성·갱신 작업은 가능한 한 idempotent하게 설계한다.
- 여러 파일을 바꾸다 실패하면 완료된 변경과 미완료 변경을 구분해 보고한다.
- 기존 사용자 파일을 덮어쓰기 전에 차이와 보존 방법을 제시한다.
- 검사 실패는 원인, 영향, 보존된 작업, 권장 복구, 재검사 방법을 구조화해 제공한다.
- 자동 복구가 데이터나 Git history를 잃을 수 있으면 실행하지 않는다.
- 외부 서비스 장애 시 로컬 source of truth를 손상하지 않고 재시도 가능한 상태로 남긴다.
- context를 잃은 AI는 구현을 추측해 계속하지 않고 project refresh를 수행한다.

## 14. 생명주기 자체의 출시 검증 시나리오

최종 제품은 최소한 다음 시나리오를 자동·수동으로 검증해야 한다.

1. 빈 폴더에서 시작해 전체 서비스 발견, root와 workspace 구성, 기술 스택 선정, UI, 계약, 구현 경계, 기능 구현, RC, 사용자 승인, release까지 완료
2. 기존 단일 repository에 구조 변경 없이 하네스를 도입하고 부족한 가장 이른 단계부터 재개
3. 여러 repository와 submodule이 있는 기존 프로젝트를 clone한 새 사용자가 문맥과 작업 상태를 복구
4. 제품 기준선 변경으로 UI·계약·구현·RC가 정확히 `stale` 처리되고 새 RC가 생성됨
5. dbdiagram 웹 변경과 Git DBML의 의미 차이를 감지하고 사용자 의도를 확인
6. 두 작업자가 서로 다른 workspace를 병렬 개발한 뒤 계약과 pointer를 안전하게 통합
7. 동일 계약 파일을 동시에 변경하려는 작업을 시작 전에 경고
8. dirty tree, divergence, detached submodule, unpublished commit에서 파괴적 자동 복구를 하지 않음
9. 컨텍스트 압축 또는 AI 교체 후 project refresh만으로 같은 다음 작업을 제안
10. RC 이후 코드 변경을 감지해 기존 사용자 승인을 무효화하고 새 RC 검증 요구
11. macOS에서 만든 프로젝트를 Windows 사용자가 clone하고 동일 단계와 다음 작업을 복구
12. 외부 task provider가 중단되어도 프로젝트 지식과 Git 상태를 잃지 않고 제한 모드로 진단
13. realtime·offline 같은 고위험 요구를 기술 spike로 검증하고 spike 코드를 production에 혼입하지 않음
14. 두 backend 기능이 generated stub과 module boundary를 기준으로 병렬 구현되어 공통 파일 충돌을 피함
15. 주요 dependency 추가 시 기능 근거, 대안, 보안, 라이선스, 운영 비용과 제거 전략을 검증
16. 외부 UI 목업을 reference·seed·canonical로 등록하고 격리 검사와 전체 상태 coverage 후 production UI에 통합
17. behavior change가 실패 test 없이 시작되거나 regression evidence 없이 merge·release되는 것을 차단
18. path는 다르지만 같은 policy·contract·migration을 수정하는 작업을 의미 충돌로 사전 경고하고 merge 순서를 조정

## 15. 이번 설계에서 확정하는 결정

- 생명주기는 신규·기존·재개 경로를 하나의 증거 기반 상태 기계로 통합한다.
- 신규 프로젝트는 발견 단계에서 안정적인 이름이 승인된 뒤 root를 만든다.
- 독립 workspace 필요성이 확정되면 P20 또는 P35에서 지체 없이 repository와 submodule을 만든다.
- 전체 제품 기준선 이후 요구사항을 근거로 아키텍처와 기술 스택을 선택하고, 고위험 기술은 격리된 spike로 먼저 검증한다.
- 전체 UI 기준선은 시나리오 fixture로 제품 행동과 데이터 요구를 검증하고 아직 없는 API 형태를 미리 고정하지 않는다.
- 전체 UI 이후 계약·DBML을 승인하고, schema 기반 생성물과 구현 경계·골격을 통합한 뒤 기능 개발을 시작한다.
- 모든 내부 interface를 미리 만들지 않고 workspace·module 간 안정적인 협업 경계만 먼저 고정한다.
- backend 구현과 frontend 실제 연결은 같은 기준선 아래에서 기능별 수직 단위로 병렬 진행하고 지속적으로 통합·테스트한다.
- 외부 UI는 source와 authority를 기록하고 격리 검사·정책·상태 coverage를 통과한 뒤 역할·domain별 작은 단위로 통합한다.
- 동작 변경과 bug fix는 TDD를 필수로 하며 문서·pure asset·generated file·폐기 spike 등 좁은 예외만 허용한다.
- 병렬 작업은 구현 전에 의미 범위를 claim하고 충돌이 있으면 담당 경계와 merge 순서를 먼저 정한다.
- 기존 프로젝트는 구조와 history를 보존하며, 증거가 부족한 가장 이른 단계부터 보완한다.
- 전제가 바뀌면 영향받는 후속 단계를 `stale`로 만들고 기존 승인과 RC를 재사용하지 않는다.
- AI 기술 준비 확인과 동일 RC에 대한 사용자 검증을 모두 통과해야 최종 release를 만든다.
- 사용자는 생명주기 명령을 외울 필요가 없으며 AI가 실제 상태에서 한 가지 다음 행동을 계산한다.
