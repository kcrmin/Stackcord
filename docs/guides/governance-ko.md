# 제품 책임자와 보호된 서비스 의미

## 무엇을 보호하나요?

제품 목적·정책·비즈니스 규칙·contract와 책임자 정책 자체를 보호할 수 있습니다. 일반 팀원도 변경 제안, 실패 테스트, 구현, issue, PR을 작성할 수 있습니다. 다만 지정된 제품 책임자가 정확한 commit을 승인하기 전에는 공식 제품 의미가 되지 않습니다.

구현 코드가 모두 자동으로 보호되는 것은 아닙니다. 서비스가 무엇을 약속하고, 허용하고, 거절하고, 요구하는지를 바꾸는 변경에 적용됩니다.

## 제품 책임자 지정

서비스 방향을 결정할 실제 Git 계정이나 팀을 AI에게 말합니다.

```text
제품팀과 Git 계정 ryanmin만 서비스 정책 변경을 승인할 수 있게 해줘.
```

Stackcord는 선택한 Git review provider, 저장소, 허용된 계정, 보호할 의미, 필요한 승인 수, 책임자의 자기 변경 승인 허용 여부를 기록합니다. 실제 provider와 계정을 선택하기 전에는 governance가 꺼져 있으므로 새 개인 프로젝트를 빈 설정으로 막지 않습니다.

책임자 목록 변경도 현재 책임자 목록으로 보호합니다. 일반 팀원이 자신을 책임자로 추가하고 같은 변경을 승인할 수 없습니다. Git `user.name`과 `user.email`은 표시 정보일 뿐 권한을 증명하지 않습니다.

## 팀원과 검토자의 흐름

```text
팀원: 예약 취소 위약금을 바꿔줘.

Stackcord: 이 변경은 비즈니스 규칙과 contract를 바꿉니다. 변경안·테스트·구현은
준비할 수 있지만, 제품 책임자가 정확한 변경을 승인해야 합니다.
PR을 만들거나 갱신하고 지정된 검토자에게 승인을 요청할 수 있습니다.
```

AI는 보호된 의미를 승인된 것으로 다루기 전에 `orchestrator governance check --json`을 실행합니다. 현재 계정이 책임자가 아니면 문서 상태를 제안으로 유지하고, 선택된 issue 도구는 논의와 작업 상태에만 사용합니다. 실제 변경 승인은 PR 또는 선택한 provider의 동등한 review가 담당합니다. Issue 담당자 지정이나 완료 상태만으로는 승인되지 않습니다.

검토 후 Stackcord는 provider를 다시 읽어 provider·저장소·commit·보호된 fingerprint·review revision·승인 계정·조회 시점을 확인합니다. 승인 뒤 보호된 내용이 하나라도 바뀌면 기존 승인은 오래된 상태가 됩니다.

## Git과 provider가 각각 하는 일

CODEOWNERS, 필수 reviewer, 보호 branch 또는 선택한 provider의 동등한 설정이 실제 merge 권한을 제한합니다. Stackcord는 변경이 보호된 서비스 의미에 해당하는지 판단하고 정확한 승인을 확인할 수 없으면 통합과 release를 차단합니다.

Provider 설정 변경은 명시적인 외부 작업입니다. 사용자가 선택하고 연결하기 전에는 GitHub·GitLab 등의 adapter를 만들었다고 가장하지 않습니다. 사용 가능한 인증 connector나 CLI를 사용하고 필요한 저장소 규칙을 설명한 뒤 정규화된 결과를 검증합니다.

## Clone과 provider 장애

책임자와 보호 범위는 Git에 commit되므로 다른 clone이나 AI도 복구합니다. PR review 증거는 Git에서 제외된 로컬 관찰이므로 선택한 provider에서 다시 읽습니다. Provider를 사용할 수 없으면 승인 상태는 unknown입니다. Cache된 review, commit 표시 이름, comment, issue 상태로 승인됐다고 추측하지 않습니다.

기본 mode는 provider 계정 승인을 사용합니다. Provider 없이도 확인 가능한 암호학적 증명이나 여러 조직 승인이 필요한 팀은 선택형 strict release에서 서명 승인 요구를 추가할 수 있습니다.

## 중요한 한계

Stackcord는 로컬 파일시스템을 제어하는 사람이 파일을 편집하는 행위 자체를 막을 수 없습니다. 대신 승인되지 않은 보호 변경이 Stackcord 검사에서 canonical로 인정되거나 통합·release를 통과하지 못하게 합니다. 승인되지 않은 merge를 실제로 막는 책임은 Git provider의 저장소 권한과 branch 규칙에 있습니다.
