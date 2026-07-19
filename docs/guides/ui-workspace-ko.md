# UI workspace와 외부 목업

`ui/`는 선택형 편집 가능 UI 기준선입니다. 화면·상태·상호작용·token·접근성·승인된 asset과 출처를 소유하고, `frontend/`는 정확한 UI 기준선 커밋을 실제 제품으로 구현합니다.

## 언제 분리하나요?

UI가 별도 소유권·history·permission·review 주기를 가지면 `ui/` submodule을 권장합니다. 작은 팀이나 하나의 저장소가 더 단순하면 일반 directory로 등록합니다. Framework나 실행 가능한 prototype은 필요할 때만 선택합니다.

## A. 아직 UI가 없는 경우

제품 역할·journey·UI coverage를 먼저 정리한 뒤 UI workspace에서 작은 흐름 단위로 만듭니다. Figma, Penpot, MengTo/Skills 같은 UI Skill 또는 직접 편집 중 필요한 방법을 선택할 수 있습니다. 외부 도구의 결과는 후보이며 승인된 `ui/` 커밋이 기준입니다.

## B. 일부 목업이 있는 경우

목업을 `seed`로 검사하고 현재 UI와 비교합니다. 전체를 가져오거나 필요한 파일만 선택하여 가져온 뒤 일반 파일처럼 수정합니다. 기존 파일과 제품 의미가 충돌하면 덮어쓰지 않고 어느 쪽을 유지할지 먼저 결정합니다.

```text
stackcord ui import --root . --archive mockup.zip --id ui.external.checkout --authority seed --license MIT --apply
stackcord ui promote --root . --id ui.external.checkout --workspace workspace.ui --mode selected --path screens/checkout.html --apply
```

## C. 이미 승인된 외부 디자인인 경우

`canonical`로 등록하고 적절한 export를 전체 또는 선택해서 가져옵니다. Canonical은 수정 금지가 아니라 현재 결정 권한을 뜻합니다. 가져온 뒤에도 error·loading·permission·responsive·accessibility처럼 빠진 상태를 보완하고 새 기준선으로 commit할 수 있습니다.

## UI submodule 만들기

원격 UI 저장소는 GitHub 등 선택한 provider에서 먼저 만듭니다. CLI는 기존 remote만 안전하게 추가하며 remote 생성·commit·push를 대신하지 않습니다.

```text
stackcord git submodule add --root . --remote https://example.com/team/product-ui.git --path ui --apply
stackcord workspace register --root . --id workspace.ui --kind submodule --path ui --remote https://example.com/team/product-ui.git --responsibility ui-baseline --consumer workspace.frontend --initialize ui --apply
```

Directory를 사용할 때는 `--kind directory`로 등록합니다.

## 기준선과 frontend 연결

UI 파일을 수정하고 일반 Git convention으로 commit·push한 뒤 기준선을 묶습니다.

```text
stackcord ui baseline bind --root . --id ui.baseline.checkout --workspace workspace.ui --source ui.external.checkout --ref ui.checkout --consumer workspace.frontend --apply
```

Frontend 작업 정의는 이 기준선 fingerprint를 기록합니다. UI 커밋·소스·root pointer 중 하나가 바뀌면 이전 frontend 작업과 evidence가 stale로 보입니다. UI가 submodule이면 기준선 파일과 새 UI gitlink를 같은 검토 가능한 root 변경으로 통합합니다.

## 충돌과 안전

- Import 검사는 path traversal, symlink, executable, secret-like content, size, license를 확인합니다.
- Quarantine은 내부 임시 안전 경계이며 사용자가 관리하지 않습니다.
- Promotion은 수정된 UI 파일을 자동으로 덮어쓰지 않습니다.
- 다른 파일을 편집하더라도 같은 UI flow·state·token·policy를 바꾸면 의미 충돌로 조정합니다.
- UI 기준선은 clean하고 remote에서 복구 가능한 commit이어야 합니다.
- Frontend는 화면 사진뿐 아니라 interaction·failure·accessibility를 TDD evidence로 남깁니다.

## Clone 후 이어가기

“이 프로젝트 이어서 해”라고 말하면 Skill이 UI checkout, source authority, baseline commit, root pointer, frontend fingerprint를 함께 확인합니다. 누락된 submodule은 root가 기록한 정확한 pointer로만 초기화하고, dirty·local-only·diverged 상태는 자동으로 버리지 않습니다.
