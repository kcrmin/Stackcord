# Production dogfood / 프로덕션 dogfood

This fixture uses the product against an actual orchestration repository, frontend and backend Git submodules, bare remotes, concurrent owners, and a clean recursive clone. It does not modify the product repository or require a hosted provider account.

이 fixture는 실제 오케스트레이션 저장소와 프론트엔드·백엔드 Git submodule, bare remote, 동시 작업자, 재귀 clone을 사용합니다. 제품 저장소를 수정하지 않으며 외부 계정도 요구하지 않습니다.

## Run / 실행

macOS or Linux:

```sh
bash dogfood/run.sh
```

Windows PowerShell:

```powershell
./dogfood/run.ps1
```

To retain the machine result and temporary repositories, pass explicit paths:

```sh
bash dogfood/run.sh \
  --output /tmp/stackcord-result.json \
  --workspace /tmp/stackcord-fixture
```

PowerShell accepts the equivalent `-Output`, `-Workspace`, and optional `-Binary` parameters. The default wrappers build the native CLI from `cli/` and place all generated state under the operating system temporary directory.

결과와 임시 저장소를 보존하려면 위와 같이 출력 경로를 지정합니다. 기본 실행은 네이티브 CLI를 빌드하고 모든 생성물을 운영체제 임시 디렉터리에 둡니다.

## What it proves / 검증 범위

The executable scenario in `scenario.yaml` and `expected-results.json` verifies:

- an orchestration root with actual frontend and backend submodules;
- approved product, business-rule, and failure-behavior contracts;
- one winner under a concurrent Git-local reservation race;
- an external task assignment reconciled to a Git CAS semantic reservation, including lifecycle synchronization and stale-read rejection;
- semantic conflict detection even when file paths do not overlap;
- rejection of evidence from the wrong workspace;
- red-then-green backend and frontend tests with commit-bound evidence;
- provider-before-consumer integration and exact root gitlinks;
- a conventional feature branch in an isolated worktree;
- technical and user verification bound to the same candidate digest;
- tamper rejection, clean-clone context recovery, one safe next action, pointer drift, and unpublished local state.
- non-destructive adoption of this product source plus a real Git-local release-guidance reservation.

한국어로 요약하면, 파일 충돌뿐 아니라 비즈니스 규약 충돌·작업 선점·잘못된 저장소 작업·서브모듈 포인터·새 clone 복구·동일 RC 검증까지 실제 Git 상태로 확인합니다.

## Boundaries / 한계

The fixture intentionally uses local bare remotes behind public-looking placeholder URLs. Its external-provider scenario executes the normalized connector boundary but does not certify hosted GitHub or Jira writes, network reliability, production load, code signing, or marketplace publication. `report.md` compares raw deterministic scenario coverage only; it does not claim that the harness makes people a particular percentage faster.

이 fixture는 외부 GitHub/Jira 쓰기, 네트워크 신뢰성, 부하 성능, 코드 서명, marketplace 공개를 인증하지 않습니다. `report.md`의 수치는 결정적 검증 범위 비교이며 사람의 생산성 향상 수치가 아닙니다.
