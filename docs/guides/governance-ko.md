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

## 제품 책임자를 안전하게 변경하기

`--subject`를 반복해 정책 파일을 바꾸지 않고 한 명 이상의 책임자 추가 계획을 먼저 확인합니다.

```sh
stackcord governance authority add --root . --subject user:alex --subject team:platform --json
```

계획에는 정확한 정책 fingerprint와 변경 전·후 책임자 목록이 나옵니다. 내용을 검토한 뒤 그 계획에 표시된 fingerprint로 적용합니다.

```sh
stackcord governance authority add --root . --subject user:alex --subject team:platform --expected-policy sha256:0000000000000000000000000000000000000000000000000000000000000000 --apply --json
```

예시 값이 아니라 계획에서 반환된 fingerprint를 사용하고, 적용할 때도 같은 책임자 목록을 전달해야 합니다. 한 명 이상의 책임자 삭제도 같은 흐름을 사용합니다.

```sh
stackcord governance authority remove --root . --subject user:alex --subject team:platform --json
```

각 계획은 원자적으로 처리합니다. 요청한 책임자 중 하나라도 형식이 잘못됐거나, 요청 안에서 중복됐거나, 추가할 때 이미 등록돼 있거나, 삭제할 때 등록돼 있지 않으면 아무도 변경하지 않습니다. 적용 단계에서는 오래된 fingerprint, 마지막 책임자 삭제, 필요한 최소 승인 수를 충족할 수 없게 만드는 삭제도 거부합니다. 다른 정책 항목과 YAML 주석은 그대로 보존합니다.

적용된 파일도 아직 governance 변경 제안입니다. Commit한 뒤 설정된 provider의 review를 마쳐야 하며, 로컬 파일 변경만으로 계정 신원이나 승인이 증명되지는 않습니다.

## 팀원과 검토자의 흐름

```text
팀원: 예약 취소 위약금을 바꿔줘.

Stackcord: 이 변경은 비즈니스 규칙과 contract를 바꿉니다. 변경안·테스트·구현은
준비할 수 있지만, 제품 책임자가 정확한 변경을 승인해야 합니다.
PR을 만들거나 갱신하고 지정된 검토자에게 승인을 요청할 수 있습니다.
```

AI는 보호된 의미를 승인된 것으로 다루기 전에 `stackcord governance check --json`을 실행합니다. 현재 계정이 책임자가 아니면 문서 상태를 제안으로 유지하고, 선택된 issue 도구는 논의와 작업 상태에만 사용합니다. 실제 변경 승인은 PR 또는 선택한 provider의 동등한 review가 담당합니다. Issue 담당자 지정이나 완료 상태만으로는 승인되지 않습니다.

검토 후 Stackcord는 provider를 다시 읽어 provider·저장소·commit·보호된 fingerprint·review revision·승인 계정·조회 시점을 확인합니다. 승인 뒤 보호된 내용이 하나라도 바뀌면 기존 승인은 오래된 상태가 됩니다.

## Git과 provider가 각각 하는 일

CODEOWNERS, 필수 reviewer, 보호 branch 또는 선택한 provider의 동등한 설정이 실제 merge 권한을 제한합니다. Stackcord는 변경이 보호된 서비스 의미에 해당하는지 판단하고 정확한 승인을 확인할 수 없으면 통합과 release를 차단합니다.

Provider 설정 변경은 명시적인 외부 작업입니다. 사용자가 선택하고 연결하기 전에는 GitHub·GitLab 등의 adapter를 만들었다고 가장하지 않습니다. 사용 가능한 인증 connector나 CLI를 사용하고 필요한 저장소 규칙을 설명한 뒤 정규화된 결과를 검증합니다.

## Clone과 provider 장애

책임자와 보호 범위는 Git에 commit되므로 다른 clone이나 AI도 복구합니다. PR review 증거는 Git에서 제외된 로컬 관찰이므로 선택한 provider에서 다시 읽습니다. Provider를 사용할 수 없으면 승인 상태는 unknown입니다. Cache된 review, commit 표시 이름, comment, issue 상태로 승인됐다고 추측하지 않습니다.

기본 mode는 provider 계정 승인을 사용합니다. Provider 없이도 확인 가능한 암호학적 증명이나 여러 조직 승인이 필요한 팀은 선택형 strict release에서 서명 승인 요구를 추가할 수 있습니다.

## 중요한 한계

Stackcord는 로컬 파일시스템을 제어하는 사람이 파일을 편집하는 행위 자체를 막을 수 없습니다. 대신 승인되지 않은 보호 변경이 Stackcord 검사에서 canonical로 인정되거나 통합·release를 통과하지 못하게 합니다. 승인되지 않은 merge를 실제로 막는 책임은 Git provider의 저장소 권한과 branch 규칙에 있습니다.

## GitHub 리뷰 보안 모드와 UI 제안

선택형 관리 UI는 세 가지 명시적 모드를 사용하는 버전 2 정책을 제안합니다. 강함은 등록된 개인 관리자 계정의 승인이 필요하고, 중간은 UTC 유효기간 안에서 허용 범위를 가진 임시 승인자도 허용하며, 약함은 Stackcord의 정책 승인 검사를 끕니다. 모든 모드에서 GitHub 권한·필수 검사·보호 브랜치는 계속 적용됩니다. GitHub는 PR 작성자의 자기 승인을 허용하지 않습니다. 단독 소유자는 약함을 명시적으로 선택하거나 다른 적격 검토자에게 요청할 수 있습니다.

모드·관리자·임시 승인자 변경은 신뢰하는 기본 브랜치의 기존 정책으로 평가합니다. 보안을 낮추는 제안이 스스로를 허가할 수 없습니다. 임시 승인자는 정책 체계 변경, 관리자 임명, 자신의 권한 연장을 할 수 없습니다. 신뢰할 정책 누락·접근 실패, 취소된 리뷰, 바뀐 커밋, 제공자 장애는 승인이 아닙니다. 최초 설정에는 GitHub로 인증된 저장소 관리자가 필요합니다. 기존 버전 1 정책은 명시적 전환 제안이 필요하며, UI 편집은 기존 최소 승인 인원과 보호 범위를 유지합니다.

`stackcord review --repo owner/repository --pr 7 --json`으로 확인된 CI 검사·충돌 상태·실시간 정책 리뷰를 보세요. 코드를 완성하고 테스트와 설명을 준비한 뒤 PR을 만들고, CI와 충돌 검사를 통과한 뒤 사람에게 리뷰를 요청하세요. 이 명령은 GitHub의 최종 보호 규칙 적용을 보완합니다. UI는 PR 리뷰로 바로 연결합니다. 적격 검토자는 `stackcord review --repo owner/repository --pr 7 --head COMMIT --account LOGIN --approve --apply`로 정확한 커밋의 GitHub 리뷰를 제출할 수도 있습니다. 머지는 하지 않습니다.

UI에 보이는 설정은 작업 트리의 제안이며 실제로 적용되는 신뢰 정책의 증거가 아닙니다. 변경을 미리 보고 로컬에 저장한 뒤 기능 브랜치와 PR 절차로 확정하세요. 최초 설정·보안 모드 변경·구매·릴리스는 일반 질문의 권장 답변만으로 승인하지 않습니다.
