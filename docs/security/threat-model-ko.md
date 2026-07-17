# Threat model

## 보호 대상

제품은 source와 history, 제품 의도, contract, database와 migration 의미, credential, 외부 UI 출처, 작업 ownership, submodule identity, test evidence, release identity를 보호합니다. 원본 대화는 저장하지 않으므로 보호 대상 repository asset이 아닙니다.

## Trust boundary

Repository file, Git remote, child 저장소, archive, diagram, 선택 task provider, AI output, hook, CI, publication system은 서로 다른 trust boundary를 넘습니다. Cache summary보다 actual local state를 먼저 검사합니다. Credential은 environment 또는 OS store에 두고 plan·command argument·tracked evidence·diagnostic에 넣지 않습니다.

## 기본 control

가장 가까운 trusted root 발견, strict parsing과 stable ID, canonical fingerprint, read-only 진단과 보이는 plan, shell 없는 Git inspect, 정확한 submodule pin, semantic conflict claim, import quarantine와 limit, TDD·integration evidence, redaction, exact user validation에 연결된 candidate digest를 사용합니다. 파괴적인 Git 복구나 외부 write를 숨기지 않습니다.

## Strict-release control

조직은 SBOM·provenance·signature·supply-chain receipt·보호된 publication 검증을 켤 수 있습니다. 이 control은 약속한 release 보장을 강화하지만 기본 repository·충돌·test·exact-candidate 검증을 대체하지 않습니다.

## 잔여 위험

AI 판단은 불완전할 수 있고 외부 도구는 바뀌며 compromised repository에는 오해를 부르는 instruction이 있을 수 있고 semantic claim은 제품 의미가 옳다는 것을 증명하지 못합니다. Trusted instruction을 review 가능하게 유지하고 최신 외부 도구 문서를 확인하며 import 내용을 검토하고 data 변경 backup을 보존하며 실제 환경에서 exact candidate를 사람이 검증해야 합니다.
