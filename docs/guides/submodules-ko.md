# Workspace와 submodule

독립적인 구현·검증·소유권·contract 경계가 확인되면 바로 workspace를 만듭니다. 그 경계가 별도 저장소와 정확한 버전 통합까지 필요하면 submodule을 사용하고 아니면 root/directory/external을 사용합니다.

Root orchestration 저장소가 contracts와 조정 상태를 소유하므로 반드시 함께 clone해야 합니다. 검토한 각 submodule은 `git submodule update --init -- <path>`로 root가 가리키는 commit을 받고, nested submodule은 따로 검사한 뒤 초기화합니다. `update --remote`를 통합 정책으로 사용하지 않습니다.

작업 전 root와 각 workspace의 dirty, ahead, behind, diverged, detached, missing, unsafe URL, pointer mismatch, nested module 상태를 확인합니다. 병렬 branch는 별도 worktree를 사용합니다. Claim은 의미 범위를 다루고 worktree는 파일만 격리합니다.

여러 저장소 변경은 additive/versioned contract, provider, consumer, frontend 연결, root pointer 순으로 합칩니다. Pointer PR에는 정확한 workspace commit, 검증, deploy 순서, rollback을 기록합니다.
