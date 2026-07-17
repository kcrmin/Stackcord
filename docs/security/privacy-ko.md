# Privacy

## 저장되는 프로젝트 지식

Discovery checkpoint에는 정규화 summary·decision·정책·scenario·품질 요구·가정·미해결 질문을 둡니다. 원본 prompt, 말투, private reasoning, 전체 대화는 보존하지 않습니다. 개인 또는 production data는 승인된 제품 spec에 실제 필요할 때만 최소화하여 저장합니다.

## Credential과 외부 도구

Credential은 environment variable 또는 OS credential store에 둡니다. DB 시각화·task provider·Git hosting·publication 도구는 선택 사항이며 감지·trade-off 검토·사용자 선택 뒤에만 연결합니다. 외부 내용은 quarantine하고 승격 전에 출처를 기록합니다.

## Diagnostic과 evidence

작은 fingerprint, stable error code, command, 결과 summary, 통제된 CI evidence link를 사용합니다. Raw log, token, home path, provider payload, 사용자 대화를 저장하지 않습니다. 프로젝트 identifier와 repository name도 민감할 수 있으므로 diagnostic bundle을 공유하기 전에 검토합니다.

## 제거와 retention

Plugin이나 CLI를 제거해도 저장소가 소유한 spec·contract·DBML·Git history는 삭제되지 않습니다. 팀이 local quarantine, stale claim, 생성 candidate, 외부 provider data의 retention을 자체 정책으로 정합니다. 안전한 cleanup은 정확한 path를 보여주고 숨겨진 context-recovery action으로 실행하지 않습니다.
