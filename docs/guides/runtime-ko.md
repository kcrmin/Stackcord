# 실행 서비스와 작업 연속성

## 시작과 연결

현재 소스의 실행 서비스는 Git과 선택형 프로젝트 하네스에 의존하지 않습니다.
루트 README에 따라 소스를 빌드하세요. 이전 `v1.0.0` 배포에는 포함되지 않습니다.
소스 저장소 밖의 비공개 디렉터리를 사용하세요.

macOS/Linux에서는 다음과 같이 준비할 수 있습니다.

```sh
mkdir -p "$HOME/.config/stackcord"
chmod 700 "$HOME/.config/stackcord"
(umask 077; openssl rand -hex 32 > "$HOME/.config/stackcord/token")
stackcord serve --db "$HOME/.config/stackcord/state.db" --token-file "$HOME/.config/stackcord/token" --ui
```

인증 파일은 처음 한 번만 생성하며, 기존 팀의 인증 파일을 덮어쓰지 마세요.
Windows에서는 현재 사용자만 읽을 수 있는 파일에 무작위 인증값을 생성하고 같은
`--token-file` 옵션으로 지정합니다. 서버는 32자 이상을 요구합니다. 기억하기 쉬운
암호 대신 암호학적 난수 생성기를 사용하세요.

서버는 `127.0.0.1:7331`에서만 수신합니다. `--port 0`은 사용 가능한 포트를 선택합니다.
선택형 UI는 출력된 주소에서 열고 인증값을 입력합니다. CLI는 `STACKCORD_TOKEN`도
지원하며 인증 파일이 우선합니다. 인증값 자체를 명령 인자로 받거나 서버 URL에 출력하지 않습니다.

아래 CLI 요청에는 `--token-file /private/path/token`을 지정합니다. 또는 호출 프로세스에
`STACKCORD_TOKEN`을 안전하게 설정하세요. `--endpoint` 기본값은
`http://127.0.0.1:7331`입니다. 실행 서비스 결과는 기존 `--json` 옵션과 관계없이 JSON입니다.

## A가 멈추고 B가 인수하기

각 JSON을 파일에 저장하고 `task apply`로 전달합니다. 사용자 지정 모델 어댑터도
같은 API를 사용하며 특정 모델이나 프롬프트 형식은 필요하지 않습니다.
작업마다 새 요청 ID를 사용하고 정확히 같은 요청을 재전송할 때만 ID를 재사용합니다.

`create.json`으로 작업을 만듭니다.

```json
{"id":"create-api","project":"shop","task":"api","kind":"create","definition":{"goal":"Implement the approved checkout endpoint","constraints":["Preserve the payment contract"]}}
```

```sh
stackcord task apply --input create.json --token-file /private/path/token
```

A가 맡습니다 (`claim-a.json`).

```json
{"id":"claim-a","project":"shop","task":"api","kind":"claim","owner":"alice","session":"alice-session-1","lease_seconds":300}
```

결과에는 `epoch: 1`이 포함됩니다. 반환된 owner/session/epoch를 저장하세요.
실행 중에는 만료 전에 해당 값과 `lease_seconds`를 담은 `checkpoint`를 보냅니다.
제공자가 사용량 제한을 알려오면 중단 상태를 저장합니다 (`pause-a.json`).

```json
{"id":"pause-a","project":"shop","task":"api","kind":"pause","owner":"alice","session":"alice-session-1","epoch":1,"reason":"quota","checkpoint":{"summary":"Request schema implemented","next":"Add validation tests","artifacts":[{"uri":"git:COMMIT:path/to/schema","digest":"sha256:CONTENT_HASH"}]},"usage":{"input_tokens":1200,"output_tokens":300}}
```

B가 인수합니다 (`claim-b.json`).

```json
{"id":"claim-b","project":"shop","task":"api","kind":"claim","owner":"bob","session":"bob-session-1","lease_seconds":300}
```

같은 명령으로 파일들을 순서대로 적용합니다. B는 새 세대 번호와 A의 checkpoint를
받습니다. 필요한 작업 문맥은 다음 명령으로 읽습니다.

```sh
stackcord resume --project shop --task api --max-bytes 16384 --token-file /private/path/token
```

B는 자신이 받은 값으로 완료를 보고합니다. A는 이전 값으로 완료 처리할 수 없습니다.
나중에 돌아온 A는 현재 문맥을 읽고 인수 가능한 작업만 맡습니다. 저장 전에 종료되면
마지막으로 보존한 진행 상황까지만 복구할 수 있습니다. 권한 만료는 인수를 허용하지만
이전 프로세스를 종료하지는 않습니다.

`approval_required` 사유로 멈춘 작업은 자동으로 인수할 수 없습니다. 중단·실패한
작업을 명시적으로 `requeue`할 때는 현재 `expected_revision`이 필요합니다.

```json
{"id":"retry-api","project":"shop","task":"api","kind":"requeue","expected_revision":3}
```

재배정해도 checkpoint는 유지됩니다. 신뢰하는 운영자의 조작이며 제품 승인 증거는
아닙니다. 부분 변경과 필요한 승인을 확인한 뒤 사용하세요. 새 실행 서비스는 직접
프로세스를 실행하거나 제공자의 사용량 한도·초기화 시간을 추측하지 않습니다.

## 변경분과 문맥 예산

```sh
stackcord projects --token-file /private/path/token
stackcord task list --project shop --token-file /private/path/token
stackcord events --project shop --after 0 --limit 100 --token-file /private/path/token
```

페이지를 처리한 뒤에만 반환된 `next` 커서를 저장하세요. `more`가 참이면 이어서 읽습니다.
커서는 프로젝트와 데이터베이스에 종속됩니다. 복원된 백업보다 앞선 커서는 오류가 나며,
복원 시점을 확인한 뒤 커서를 재설정해야 합니다. 처리하지 않은 이벤트를 조용히 버리지 마세요.

이벤트는 목표나 로그를 반복하지 않고 식별자와 상태 변화를 담습니다. 필요한 작업만
`resume`으로 읽습니다. 큰 결과물은 참조 위치에 두며 서비스가 직접 가져오지 않습니다.
필수 제약은 조용히 잘라내지 않습니다. 바이트 예산이 부족하면 필요한 크기와 함께
오류를 반환합니다. 바이트 수는 토크나이저 기준의 토큰 수가 아닙니다. 사용량은 보고된
증분만 요청 ID마다 한 번 합산합니다. 바이트 측정이 토큰 절감 실험을 의미하지 않습니다.

## 원격 작업자와 데이터 보호

여러 컴퓨터를 연결하려면 접근 가능한 한 컴퓨터에 조정 서비스를 실행하고 인증된
SSH 터널의 로컬 HTTP로 접속하거나, 라이브러리 handler를 운영자가 관리하는 HTTPS
서비스 뒤에 배치합니다. 배포 CLI는 LAN 수신 옵션을 제공하지 않습니다. HTTP 클라이언트는
로컬 외부의 평문 HTTP와 redirect를 거부합니다. 요청은 bearer 인증을 요구하며
다른 출처의 브라우저 요청을 거부합니다.

하나의 인증값은 해당 실행 서비스의 모든 프로젝트와 운영 조작에 접근할 수 있습니다.
작업자 이름은 인증 주체가 아닙니다. 신뢰하지 않는 팀은 분리하세요. Stackcord 자체는
데이터베이스를 암호화하지 않으므로 운영체제 권한으로 디렉터리를 보호하세요.
Git이나 네트워크 파일시스템에 두지 마세요. 작업·목표·checkpoint·결과물 참조에는
비공개 프로젝트 정보가 포함될 수 있습니다.

## 백업·복원과 기존 Git 채널

```sh
stackcord backup --output /private/backup/stackcord.db --token-file /private/path/token
```

백업은 기존 파일 덮어쓰기를 거부하며 실패하면 불완전한 출력 파일을 제거합니다.
일관된 데이터베이스 스냅샷을 저장합니다. 복원할 때는 실행 서비스를 종료하고 기존
데이터베이스를 보관한 뒤, 백업을 새로운 비공개 로컬 경로로 복사하고 인증값과 해당
경로를 지정해 `serve --db`로 다시 시작합니다. Git clone만으로 진행 상황을 복원할 수는
없습니다. 자동 이력 삭제나 디스크 압축은 실행하지 않습니다.

기존 `stackcord channel`은 선택형 하네스에 유지합니다. 서명된 Git 이력, 신뢰한
작업자 키, 로컬 실행 기록도 그대로 남습니다. 두 방식의 상태를 자동으로 이전하지
않습니다. 기존 작업은 그곳에서 마치거나 중단하고, 새 실행 서비스에는 관련 Git commit과
결정을 참조하는 새 작업을 만드세요. 원래 명령 경로는 설치된 Skill을 위해 숨겨진
호환 경로로 남습니다. 확장 사용을 명확하게 하려면 `stackcord harness channel`,
`stackcord harness status`, `stackcord harness dashboard`를 사용하세요.

기본 웹 화면은 로컬 주소와 SSH 터널을 지원합니다. HTTPS API reverse proxy는
웹 화면 배포 경로로 지원하지 않습니다. 브라우저 출처는 handler에 지정한 host와
TLS scheme이 일치해야 하며 forwarded header를 자동으로 신뢰하지 않습니다.
정의는 32 KiB, checkpoint는 24 KiB, 복귀 문맥 전체는 1 MiB(기본 16 KiB)까지
지원합니다. 더 큰 입력은 결과물 참조로 전달하세요.
