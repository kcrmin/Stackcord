# Production 출시

Production hardening에는 안정적인 required check, 자동화된 critical verification, macOS/Windows journey, Plugin 없이 clone 이어가기, contract/migration rollback, 보안·license 검토, SBOM, provenance, signature, observability, backup/restore, 운영, support, 담당자가 있는 warning이 필요합니다.

`release prepare`는 정확한 root/workspace commit과 artifact/evidence digest를 하나의 manifest digest로 묶습니다. 사용자가 실제 환경에서 같은 RC를 실행하고 영수증에 그 digest를 기록합니다. Code·contract·docs·artifact·signature·evidence·설정 identity 중 하나라도 바뀌면 새 후보를 만듭니다.

`release publish`는 항상 승인 등급 D이며 signed tag, 재현 build, release artifact, marketplace, Homebrew, WinGet, install smoke test, notes, rollback, support 등 모든 공개 side effect를 먼저 계획합니다. Clean install에서 checksum과 signature를 확인하기 전에는 출시가 끝난 것이 아닙니다.

보호된 tag의 release workflow에는 `release/approved-rc.json`, `release/user-validation-receipt.json`, `release/approval-receipt.json` 세 tracked 입력이 필요합니다. 공개 artifact를 staging하기 전에 RC manifest digest를 다시 계산하고 사용자 검증 receipt의 hash와 exact objective·repository·production target·operation ID·만료·Class D 확인을 대조합니다. 이 파일에는 identity와 compact evidence만 두며 secret이나 raw log를 넣지 않습니다.
