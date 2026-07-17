# Production 준비와 release

## 계속 강화

Production 준비는 마지막 testing 단계가 아닙니다. 각 변경은 해당되는 TDD·contract·accessibility·security·observability·실패·migration·rollback·integration evidence를 가집니다. Candidate 준비 전 AI가 제품 전체 coverage, open risk, Git reachability, clean workspace, submodule pointer, 재현 가능한 build input을 검토합니다.

## Candidate 하나 준비

기본 `release prepare`는 정확한 root/workspace commit, artifact digest, 제품/docs/contract fingerprint, TDD evidence, integration evidence, 조건부 migration/rollback evidence를 결정적으로 묶습니다. 준비는 candidate를 쓰지만 공개 side effect를 만들지 않습니다. 필수 evidence가 없으면 중단합니다.

## 정확한 candidate 검증

기술 gate가 candidate를 fresh current input과 먼저 비교합니다. 그 다음 사용자가 실제 target 환경에서 같은 candidate를 실행하고 동작을 확인합니다. 작은 validation record는 candidate digest와 evidence를 지정하며 secret이나 대화 원문을 담지 않습니다. `release verify`는 기술 identity와 사용자 검증이 바뀌지 않은 정확히 같은 digest를 가리킬 때만 통과합니다.

## Release profile 선택

평범한 프로젝트는 기본 mode를 사용합니다. Strict release는 `profiles/strict-release/`에서 SBOM·provenance·signature·supply-chain evidence·보호된 publication check·조직용 control을 추가합니다. 팀이 그 보장을 약속하거나 요구할 때만 켭니다.

## Core verifier 밖에서 공개

공개 저장소 생성, tag/artifact 게시, package channel, signing identity, 배포 credential, 되돌릴 수 없는 production 작업에는 명시적인 사용자·조직 권한이 필요합니다. Core CLI는 의도적으로 검증된 candidate에서 멈춥니다. 조직은 모든 외부 side effect와 rollback을 검토한 뒤 선택한 CI/CD 또는 strict profile을 연결할 수 있습니다.
