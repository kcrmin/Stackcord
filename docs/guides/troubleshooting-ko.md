# 문제 해결

## AI가 프로젝트를 잊음

“아무것도 하기 전에 이 프로젝트 context를 복구해줘”라고 말합니다. 복구 Skill은 `AGENTS.md`, `.harness/entry.md`, canonical spec·contract를 읽고 context·Git audit을 실행합니다. 이미 답한 질문을 반복하면 context audit과 각 unknown의 정확한 source 또는 fingerprint를 요청합니다. 대화 기억으로 프로젝트를 다시 만들지 않습니다.

## Clone의 submodule 누락 또는 불일치

Git inspect와 submodule sync plan을 요청합니다. Checkout 누락은 pointer mismatch, dirty child, detached child, unreachable commit과 다릅니다. Root가 기록한 commit만 초기화합니다. 정당한 child 변경은 root pointer를 바꾸기 전에 commit하고 공유합니다.

## Branch가 dirty 또는 diverged

자동 mutation을 멈춥니다. AI에게 branch, upstream, ahead/behind 수, 변경 path, 양쪽 고유 commit을 보여달라고 합니다. 영향을 본 뒤 merge·rebase·commit·stash·cleanup을 고릅니다. 제품은 조용히 reset하거나 force-push하지 않습니다.

## 병렬 작업이 충돌로 막힘

충돌 category를 읽습니다. Path overlap은 파일을 나누거나 직렬화합니다. 정책·contract·DB·UI·dependency·pointer overlap은 공유 boundary와 통합 순서를 합의합니다. Ownership이 명확해진 뒤 claim을 갱신하며 blocker를 우회하려고 지우지 않습니다.

## 외부 task 도구를 사용할 수 없음

실제 connector 또는 executable이 생길 때까지 Git-local을 live status로 둡니다. 여러 authority 사이에 status를 복사하지 않습니다. 제품 spec·contract·claim·fingerprint·release identity는 task provider와 무관하게 저장소가 소유합니다.

## DBML 또는 UI input이 stale하거나 위험함

외부 input을 quarantine에 둡니다. 의미와 출처를 비교하고 license와 이유를 확인한 뒤 수용한 변경만 명시적으로 승격합니다. 시각화·archive·remote mockup이 canonical file을 자동 덮어쓰게 하지 않습니다.

## Release 검증이 더 이상 통과하지 않음

보고된 변경 field를 현재 commit, artifact, 제품 docs, contract, test, integration 결과, migration, 사용자 validation과 비교합니다. 중요한 변경은 새 candidate와 새 digest 검증이 필요합니다. 통과시키려고 digest나 validation record를 편집하지 않습니다.

## Plugin을 사용할 수 없음

어떤 coding AI에서든 `.agents/skills/use-project-harness/SKILL.md`와 Markdown fallback을 엽니다. 결정적 검사가 필요하면 Go CLI를 build하거나 찾습니다. Plugin-less 동작은 편의가 줄어들 뿐 source of truth가 달라지지 않습니다.
