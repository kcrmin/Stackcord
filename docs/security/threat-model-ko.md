# Threat model

## 보호 대상

제품은 source와 history, 제품 의도, contract, database와 migration 의미, credential, 외부 UI 출처, 작업 ownership, submodule identity, test evidence, release identity를 보호합니다. 원본 대화는 저장하지 않으므로 repository asset이 아닙니다. 보호해야 할 coordination 불변조건은 외부 live status, Git 작업 선점, 현재 서비스 의미, workspace commit, exact candidate를 서로 몰래 대체할 수 없다는 것입니다.

## Trust boundary

Repository instruction, Git remote, child 저장소, submodule URL, archive, DBML, diagram, 선택 task provider, Memory 도구, AI output, hook, CI, publication system은 서로 다른 trust boundary를 넘습니다. 악성 issue·comment·mockup·repository file에는 prompt injection이 있을 수 있으며 그 text는 input일 뿐 command 실행이나 정책 변경 권한이 아닙니다. Connector는 provider output을 identity·status·owner·dependency·revision·timestamp·capability·source·raw hash만 가진 제한된 normalized observation으로 줄입니다. CLI는 이 observation을 검증하지만 원본 payload를 실행하지 않습니다.

Cache summary보다 actual local state를 먼저 검사합니다. Credential은 environment 또는 OS store에 두고 plan·command argument·tracked evidence·diagnostic에 넣지 않습니다. Remote URL과 안전하지 않은 submodule URL 변경은 review가 필요합니다. 인증된 provider에서 왔다는 이유만으로 외부 내용이 canonical이 되지 않습니다.

## 기본 control

가장 가까운 trusted root 발견, strict schema와 stable ID, canonical fingerprint, read-only 진단, 보이는 mutation plan을 사용합니다. Git은 shell 없이 command allowlist, 축소한 environment, protocol 제한, output limit 안에서 실행합니다. Resolved root 기준 path containment로 symlink escape, path traversal, 비정상 provider file, 정규화 뒤 중복 archive name, 과도한 archive entry 수와 archive size를 차단한 뒤 quarantine 내용을 승격할 수 있습니다.

Root 저장소는 정확한 child pin을 기록합니다. Issue assignee는 advisory이며 coordination branch는 compare-and-swap으로 의미 범위를 배타적으로 선점하고 stale revision이나 race 패배를 거부합니다. 파일 path가 달라도 policy·scenario·contract·DB entity·migration·UI flow·dependency·workspace·pointer 의미를 검사합니다. Normalized observation에는 짧은 freshness window가 있으며 cache는 live provider 상태를 증명하지 못합니다. TDD·integration evidence는 현재 commit에, 기술·사용자 validation은 하나의 exact candidate digest에 묶입니다. 파괴적인 Git 복구나 외부 write를 숨기지 않습니다.

## Strict-release control

조직은 SBOM·provenance·signature·supply-chain receipt·보호된 publication 검증을 켤 수 있습니다. 이 control은 약속한 release 보장을 강화하지만 기본 repository·충돌·test·exact-candidate 검증을 대체하지 않습니다.

## 잔여 위험

AI 판단은 불완전할 수 있고 외부 도구는 바뀌며 compromised repository에는 오해를 부르는 instruction이 있을 수 있고 의미 선점은 제품 의미가 옳다는 것을 증명하지 못합니다. Local test는 hosted provider write, account permission, network reliability, rate limit, production load, marketplace review, signing infrastructure를 인증하지 않습니다. Trusted instruction을 review 가능하게 유지하고 최신 외부 도구 문서를 확인하며 import 내용을 검토하고 data 변경 backup을 보존하며 실제 환경에서 exact candidate를 사람이 검증해야 합니다.
