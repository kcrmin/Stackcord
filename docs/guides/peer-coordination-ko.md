# 컴퓨터 사이의 작업 요청

선택형 통신 채널은 공유 Git 원격 저장소를 통해 등록된 작업자끼리 요청과 결과를 주고받습니다. 서로 다른 사용자·컴퓨터·AI 클라이언트가 참여할 수 있습니다. 각 컴퓨터에서 한 번 등록하고 로컬 실행기를 명시적으로 선택하면, 일반 작업 요청·선수 작업 완료·응답은 사람이 복사해 전달하지 않아도 이어집니다.

## 참여 및 신뢰 등록

전용 공유 Git 원격 저장소, 공통 채널 이름, 컴퓨터별 고유 작업자 이름을 정합니다. 참여자는 해당 원격 저장소의 읽기·쓰기 권한이 필요합니다. 채널은 프로젝트 브랜치나 Git 인덱스 대신 별도 브랜치와 로컬 객체 저장소를 사용합니다. Git 커밋 작성자 이름은 신원 확인 수단이 아닙니다. 메시지에 서명하고 수신자는 로컬에 등록한 공개 키로 확인합니다.

```sh
stackcord channel setup --channel team --peer backend --remote https://example.org/team/coordination.git
stackcord channel setup --channel team --peer backend --remote https://example.org/team/coordination.git --apply
stackcord channel status
stackcord channel trust --peer frontend --public-key PUBLIC_KEY --apply
```

먼저 `--apply` 없이 변경안을 확인하세요. 최초 등록 때 실제 참여자와 공개 키를 교환하고 확인하며, 비공개 설정은 교환하지 않습니다. 채널 기록에 등장하는 작성자는 모든 참여 작업자가 신뢰 등록해야 합니다. 대시보드의 통신 화면에서도 참여와 신뢰 등록을 미리 볼 수 있습니다. 신뢰한 상대의 허용된 요청은 자동 실행 대상이 될 수 있으므로, 참여 등록은 각 컴퓨터에서 명시적으로 결정합니다.

## 선수 작업 요청과 결과 수신

```sh
stackcord channel send --to backend --kind implementation --title "Prepare the API contract" --body-file request.md --scope contracts --apply
stackcord channel send --to frontend --kind implementation --title "Implement the consumer" --depends-on REQUEST_ID --body-file consumer.md --scope frontend --apply
stackcord channel status --sync
stackcord channel wait --request REQUEST_ID --timeout 30m
stackcord channel respond --request REQUEST_ID --status success --body-file result.md --apply
```

의존 작업은 모든 선수 작업에 대해 지정된 수신자가 서명한 유효한 성공 응답이 있어야 실행 준비 상태가 됩니다. 실패하거나 끝나지 않은 선수 작업은 차단 상태로 남습니다. 채널은 요청과 응답을 기록하며, 선택된 작업 상태 원본인 GitHub Issues, 의미 단위 작업 선점, 테스트, 릴리스 증거를 대체하지 않습니다. 상대의 성공 메시지는 정책 승인이나 코드 머지 권한을 증명하지 않습니다.

## 이 컴퓨터에서 자동 작업 켜기

Codex나 Claude가 설치되어 있다면 기본 제공 설정을 선택할 수 있습니다. 모델에 명시적인 JSON 성공·실패 응답을 요구하므로, 해결되지 않은 권한 요청을 선수 작업 성공으로 처리하지 않습니다. Codex 설정은 workspace-write 샌드박스를 유지하고 대화형 권한 승격을 사용하지 않으며, Claude 설정은 기존 로컬 권한과 `dontAsk`를 사용합니다. 두 설정 모두 호스트 권한을 우회하지 않습니다. 선택한 실행 파일을 작업자 PATH에 등록하세요. 필요한 경우 사용자 지정 인자 배열에 실행 파일의 절대 경로를 넣을 수 있습니다.

```sh
stackcord channel runner --host codex --kind implementation --timeout 900 --apply
stackcord channel runner --host claude --kind implementation --timeout 900 --apply
```

작업자마다 하나를 선택합니다. 이 명령은 해당 컴퓨터의 실행기 선택을 교체합니다. 맡길 작업에 적절한 호스트 도구 권한만 미리 설정하세요. 추가 권한이 필요한 요청은 스스로 권한을 부여하지 않고 차단·실패 응답을 반환합니다.

표준 입력으로 요청 JSON 하나를 읽고, 공유하려는 결과만 표준 출력에 쓰는 로컬 실행 파일을 준비합니다. 프로세스 종료 상태에 따라 성공·실패를 기록합니다. 표준 출력에는 진단 로그나 비밀 정보를 넣지 않습니다. 실행 파일 자체의 샌드박스·도구·권한을 설정하세요. 요청의 `scope`는 작업 의도를 나타내며 운영체제 수준의 샌드박스가 아닙니다. 원격 메시지가 실행 파일 경로나 인자를 선택할 수는 없습니다.

사용자 지정 실행기도 `--result-format json`을 선택하고 정확히 `{"status":"success","body":"result"}` 또는 `{"status":"failed","body":"reason"}` 형태로 응답할 수 있습니다. 형식이 잘못되면 실패 처리합니다. 기본 제공 호스트 설정은 항상 이 구조화된 형식을 사용합니다. 일반 텍스트는 사용자 지정 실행기에만 기본 적용됩니다.

```sh
stackcord channel runner --argv '["/absolute/path/to/local-runner"]' --kind implementation --timeout 300
stackcord channel runner --argv '["/absolute/path/to/local-runner"]' --kind implementation --timeout 300 --apply
stackcord channel worker --apply
stackcord channel worker --once --apply
```

운영체제에 맞는 JSON 인자 배열을 사용합니다. 작업자는 종료할 때까지 전경에서 확인하며, 설치되는 상주 서비스가 아니고 외부에서 접속할 네트워크 포트도 필요하지 않습니다. 명시적으로 허용한 작업 종류만 실행합니다. 오프라인 컴퓨터는 다시 연결하고 작업자를 시작하면 대기 요청을 받습니다. API·모델 사용량은 선택한 로컬 실행기와 계정에 따라 발생합니다.

실행 중인 AI는 같은 CLI로 추가 요청을 보내거나 상대의 결과를 기다릴 수 있습니다. 받은 내용은 작업 데이터이며 로컬 규칙을 무시할 권한이 아닙니다. 서비스 방향이나 보안 정책 변경에는 기존 신뢰 정책의 승인이 계속 적용됩니다. 일반 작업은 이미 부여받은 권한으로 진행하고, 실제로 사람의 권한이 필요한 결정만 요청합니다.

## 재시작 및 진단

실행기를 시작하기 전에 실행 기록을 남깁니다. 완료 결과를 전송 전에 저장하므로 연결이 끊기면 작업을 반복하지 않고 전송만 재시도합니다. 시간 초과·취소·미완료 실행이 발생하면 해당 컴퓨터의 다른 자동 작업도 멈추고 완료 결과를 보내지 않습니다. 실행기가 종료돼도 하위 프로세스가 남을 수 있습니다. 남은 프로세스를 종료하고 일부 반영된 변경을 확인한 뒤 명시적으로 재시도를 허용하세요. 재시도는 이 정리를 확인하는 절차이며, 프로세스를 종료하거나 종료 여부를 검사하는 기능이 아닙니다. 수동 응답으로 이 로컬 중지를 우회할 수는 없습니다.

```sh
stackcord channel retry --request REQUEST_ID --apply
```

통신 화면은 로컬에서 검증한 상태를 보여줍니다. 원격 상태를 읽으려면 화면의 새로고침이나 `stackcord channel status --sync`를 사용합니다. 비공개 키·실행기 인자·실행 기록은 Git에서 제외된 `.harness/local/`에 저장하며, 대시보드는 비공개 키를 내보내거나 임의 명령을 실행하는 기능을 제공하지 않습니다. 기록 재작성, 신뢰하지 않은 작성자, 잘못된 서명은 차단합니다. 서명은 신원을 확인하지만 내용을 암호화하지 않으므로 공유 채널 원격 저장소는 의도한 참여자만 접근하게 유지합니다.

현재 버전은 채널당 이벤트 1,000개, 서명된 개별 데이터 32 KiB로 제한합니다. 기록 한도에 도달하기 전에 새 채널을 준비하세요. 공간 확보를 위해 기존 채널 기록을 삭제하거나 다시 쓰지 않습니다.
