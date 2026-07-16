# 테스트, RC, Release, 운영 준비 기준

> 상태: 확정
>
> 마지막 갱신: 2026-07-16
>
> 공개 첫 release는 실험판 표기를 붙인 미완성 배포가 아니라 production 기준을 충족한 `1.0.0`이다. release 전에는 기술 검증과 사용자 검증을 같은 source SHA와 artifact에 대해 수행한다.

## 1. TDD 정책

다음 변경에는 test-first가 필수다.

- 새로운 관찰 가능한 기능과 bug fix
- contract, compatibility, migration, rollback
- security·privacy·approval·path/process behavior
- UI interaction, state, accessibility behavior
- Git/submodule/worktree/provider mutation
- lifecycle, stale propagation, conflict detection, release gate

순서:

```text
acceptance scenario
→ 실패하는 가장 작은 automated test
→ 최소 구현
→ 통과
→ 구조 개선
→ 관련 integration/end-to-end 회귀 검사
```

허용 예외:

- 설명만 바뀌는 문서
- 순수 asset 교체
- 결정적 generator의 output 자체
- 버리는 탐색 spike
- 의미 없는 formatting

예외도 PR에 이유와 대체 verification을 남긴다. “테스트하기 어렵다”는 예외 사유가 아니다. test harness를 먼저 개선한다.

## 2. Test portfolio

| 층 | 검증 대상 |
|---|---|
| unit | parser, normalizer, policy, fingerprint, path, redaction |
| property/model | lifecycle transition, stale graph, merge ordering, idempotency invariant |
| fuzz | YAML/JSON/DBML/import/archive/provider output parser |
| golden | generated project, migration, human/JSON output, localized docs |
| integration | real Git repo, submodule, worktree, provider contract, filesystem |
| end-to-end | 신규 프로젝트와 기존 clone이 discovery부터 RC까지 가는 흐름 |
| security | path escape, command injection, secret leak, untrusted Hook/config, malicious archive |
| install/upgrade | clean install, update, migration, rollback, uninstall, reinstall |
| compatibility | Plugin × CLI × harness schema × adapter version matrix |
| performance | large repository scan, graph update, bounded memory, cancellation |

line coverage 숫자 하나를 release 기준으로 사용하지 않는다. 대신 critical mutation과 safety invariant의 scenario coverage를 필수로 한다. coverage report는 사각지대를 찾는 보조 지표로 유지한다.

## 3. 반드시 test하는 Git topology

- Git이 없는 빈 directory
- 기존 single repository
- root + 하나의 submodule
- root + 여러 submodule과 directory workspace 혼합
- nested path와 space/Unicode가 있는 path
- detached submodule HEAD
- dirty tracked/untracked file
- local ahead/behind/diverged branch
- missing/deleted/renamed remote branch
- shallow clone과 missing submodule
- worktree 병렬 branch
- cross-repo contract change bundle
- tag/maintenance hotfix와 forward-port
- Git 없는 fallback에서 release gate 차단

fake Git만으로 끝내지 않고 temporary real repositories와 bare remotes를 사용한다.

## 4. OS 지원 matrix

### First-class release target

- macOS arm64
- macOS x86_64
- Windows x86_64
- Windows arm64

각 대상에서 clean machine install, path/line ending, PowerShell, Git credential interaction, archive signature, update/uninstall E2E를 실행한다.

Linux x86_64/arm64 artifact도 제공할 수 있지만 실제 end-to-end suite가 안정적으로 통과하기 전에는 `standard-compatible`로 표시한다. 지원 문구는 테스트 증거보다 앞서지 않는다.

## 5. AI client conformance

동일 fixture repository에서 다음 행동을 평가한다.

- correct root와 current work 복구
- 이미 승인된 결정을 반복 질문하지 않음
- unknown을 추측하지 않음
- conflict preflight와 TDD 순서 준수
- destructive/production approval 준수
- 압축 후 context audit로 복귀
- natural-language request를 stable CLI command로 변환
- Plugin이 없을 때 repo-local Skill fallback

Codex는 first-class verified target이다. 다른 client는 위 suite를 실제 통과한 version만 verified로 표시한다.

## 6. 생성 프로젝트 품질 검사

- framework/database/cloud 중립 template에 기술 전제가 숨어 있지 않음
- existing repository에서 user file을 덮어쓰지 않음
- generated `AGENTS.md`가 짧고 canonical entry만 가리킴
- JSON Schema와 sample file이 일치함
- 한국어·영어 문서가 같은 stable section과 의미를 가짐
- generated summary에 source fingerprint가 있음
- `.gitignore`가 secret/cache만 제외하고 shared state를 숨기지 않음
- `.gitattributes`가 OS 간 fingerprint를 안정화함
- uninstall 시 project-owned spec/contract를 삭제하지 않음

## 7. Production readiness gate

### 제품과 문서

- 신규/기존 프로젝트 주요 journey가 끝까지 동작
- 모든 command, error, recovery, security, migration, provider setup 문서화
- example project와 실제 multi-repo fixture 제공
- public roadmap와 known limitations 정확히 공개
- English canonical과 Korean parity 검증

### Engineering

- 모든 required test와 static/security/license/secret scan 통과
- supported OS clean install/upgrade/uninstall 통과
- critical operation fault-injection과 idempotent recovery 통과
- schema migration과 이전 stable version rollback 검증
- 성능 budget과 cancellation/timeout 검증
- dependency pinning과 reproducible build 검증

### Security와 supply chain

- threat model review
- 외부 security review 또는 독립 review 완료
- SBOM 생성
- source provenance와 reproducible build evidence
- artifact code signing과 checksum
- vulnerability reporting·severity·patch SLA 공개
- untrusted repository test suite 통과

### 운영과 지원

- install/update outage가 core local workflow를 막지 않음
- release rollback runbook
- provider outage/degraded behavior runbook
- issue template, reproduction bundle, privacy-safe diagnostic export
- supported version와 deprecation policy
- release owner와 emergency key/process

## 8. Release candidate

`rc create`는 다음을 immutable manifest로 고정한다.

```text
root commit
all workspace commits and submodule pointers
Plugin source commit and package digest
CLI source commit and binary digests
harness/schema/adapter versions
dependency lock and build environment
SBOM, provenance, signatures
test and gate receipt IDs
documentation fingerprint
```

RC 뒤에 source, dependency, document, artifact가 하나라도 바뀌면 기존 사용자 검증은 무효다. 새 RC를 만든다.

## 9. 두 단계 승인

### 1차: AI 기술 승인

AI가 full matrix, security, install/update, migration/rollback, docs, supply chain을 엄격히 검사하고 blocker가 0임을 evidence로 증명한다. warning은 risk owner와 이유 없이 무시할 수 없다.

### 2차: 사용자 검증

사용자는 같은 RC SHA와 artifact로 macOS/Windows 대표 환경에서 실제 주요 journey를 실행한다.

권장 사용자 확인:

- 새 프로젝트 시작과 긴 발견 대화 저장
- 기존 multi-repo clone 후 “이어서 해” 복구
- submodule workspace 추가와 병렬 작업 충돌 경고
- 외부 UI import
- DBML 수정과 dbdiagram 시각 확인·pull diff
- TDD 기능 개발과 PR-ready 결과
- RC 생성과 publish 직전 화면

사용자 승인은 기술 검사를 대체하지 않으며, AI 기술 승인은 실제 사용자 경험 검증을 대체하지 않는다.

## 10. Publish

final approval 뒤 자동화된 release workflow가 수행한다.

1. protected source tag 생성
2. clean ephemeral runner에서 reproducible build
3. test matrix 재확인
4. SBOM/provenance/signature/checksum 생성
5. GitHub Release와 Plugin marketplace metadata 게시
6. Homebrew/WinGet channel update PR 또는 publish
7. 설치 smoke test
8. release notes, migration, known limitations, rollback, support link 공개

`latest` pointer보다 semantic version과 digest가 원본이다. release artifact를 나중에 같은 version으로 교체하지 않는다.

## 11. Versioning과 support

- 첫 public production release: `1.0.0`
- SemVer를 사용한다.
- harness schema breaking change는 major version과 migration guide가 필요하다.
- security fix는 영향받는 supported line에 backport한다.
- 최소 두 개의 최신 minor line 또는 공개한 기간 정책 중 더 명확한 범위를 지원한다.
- deprecation은 warning, migration path, removal version을 미리 공개한다.

## 12. Rollback

### CLI/Plugin release

- 이전 signed artifact와 marketplace entry를 유지한다.
- project schema를 새 version으로 migration했다면 compatibility 또는 restore 절차를 먼저 확인한다.
- binary만 downgrade해서 새 schema를 쓰지 못하는 상태를 만들지 않는다.

### 생성 프로젝트 release

- source commit, submodule pointer, deploy artifact, database migration이 함께 rollback 가능한지 검사한다.
- destructive migration은 backward-compatible expand/contract를 우선한다.
- rollback 불가능한 단계는 backup·restore rehearsal과 user impact 승인 없이는 release하지 않는다.

## 13. Issue와 진단 bundle

공개 release 후 GitHub Issues를 기본 support channel로 제공한다. bug template는 다음을 수집한다.

- CLI/Plugin/schema version
- OS/architecture와 Git version
- command와 stable error code
- redacted `doctor --json` 및 operation receipt
- expected/actual behavior와 minimal fixture

diagnostic export는 secret, source content, absolute user path를 기본적으로 제외하고 사용자가 포함 범위를 확인한 뒤 첨부한다.

## 14. Release 차단 조건

- flakey required test
- manual-only critical verification
- unsigned 또는 출처 불명 artifact
- schema migration rollback 미검증
- untrusted repository에서 command auto-execution 가능
- Windows/macOS 중 하나의 first-class journey 실패
- Plugin 없이 clone continuation 불가
- known data loss/history rewrite risk 미해결
- user validation과 artifact SHA 불일치

## 15. 수용 기준

- first public release가 production checklist와 두 단계 승인을 통과한 1.0.0이다.
- macOS와 Windows 설치·업데이트·복구가 실제 machine matrix에서 검증된다.
- 같은 RC만 기술·사용자 검증과 publish에 사용된다.
- release 실패 시 source, binary, schema, generated project의 복구 경로가 문서화된다.
- issue로 받은 문제를 privacy-safe evidence로 재현할 수 있다.
