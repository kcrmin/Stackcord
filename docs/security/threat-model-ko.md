# 위협 모델

보호 대상은 source code, 제품 policy, contract, Git history, credential, provider data, 외부 import, release identity, 사용자 승인입니다. Trust boundary는 local machine, 신뢰하지 않은 repository, Git remote, task provider, dbdiagram, archive, CI, package registry, production target입니다.

주요 위협은 repository instruction injection, path/symlink/junction escape, 악성 archive와 decompression bomb, command/argument injection, secret 노출, credential이 든 remote URL, 숨은 Git 변경, stale·조작된 provider 상태, 외부 write 중복, contract 의미 충돌, submodule pointer 바꿔치기, 위험한 Hook, dependency compromise, 사용자 검증 뒤 RC 교체입니다.

대응은 신뢰된 가장 가까운 root 탐색, strict schema와 duplicate key 차단, canonical fingerprint, read-only 기본 진단, A–D 승인, shell을 쓰지 않는 allowlisted Git read, operation journal·idempotency receipt, import 격리·용량 제한, 환경 변수 secret·redaction, 정확한 submodule pin, semantic claim, capability negotiation, immutable RC digest, signed artifact, SBOM/provenance, 같은 RC 사용자 검증입니다.

남은 위험은 담당자와 근거가 있는 warning으로 기록합니다. Production 공개는 항상 정확한 승인이 필요하며 조직 정책으로 비활성화할 수 있습니다.
