# 설계 문서 안내

> 상태: 설계 완료 — 공개 제품 이름만 보류
>
> 마지막 갱신: 2026-07-16

## 먼저 읽을 것

1. [결정 요약](./working-design.md) — 무엇을 만드는지와 이미 확정한 선택
2. [제품 lifecycle](./01-project-lifecycle.md) — 발견부터 운영까지의 순서와 gate
3. [생성 프로젝트 구조](./02-generated-project-structure.md) — target repository에 생기는 모든 영역의 책임
4. [구현 계획](../superpowers/plans/2026-07-16-fullstack-orchestrator-production.md) — 실제 source와 release를 만드는 TDD 실행 순서

## 전체 문서

| 문서 | 한 줄 내용 | 특이 사항 |
|---|---|---|
| [01](./01-project-lifecycle.md) | 진입→발견→제품→stack→UI→계약→구현→RC→운영 | waterfall이 아니라 stale 전파가 있는 dependency gate |
| [02](./02-generated-project-structure.md) | `specs/`, `contracts/`, `.harness/`, `docs/`, workspace 구조 | workspace는 submodule과 다른 개념 |
| [03](./03-context-and-source-of-truth.md) | stable ID, fingerprint, actual-state refresh, 압축 복구 | 대화·AI memory는 원본이 아님 |
| [04](./04-git-collaboration-and-submodules.md) | branch, PR, worktree, submodule pointer, change bundle | protected `main`; `develop`은 기본 아님 |
| [05](./05-ai-action-and-approval-policy.md) | AI 자동 실행 범위와 destructive/production 승인 | 반복 질문을 줄이는 task-scoped consent |
| [06](./06-external-adapters.md) | task/Git host/dbdiagram/UI/AI client adapter | Git DBML이 canonical; 외부 상태 원본은 하나 |
| [07](./07-checker-cli-and-result-schema.md) | Go CLI command, plan/apply, JSON, exit code, OS 규칙 | 사람·AI·CI가 같은 결정적 검사 사용 |
| [08](./08-plugin-skills-installation-security.md) | Plugin/Skill/Hook 차이, GitHub 공유, 설치·migration·보안 | Plugin 없이 repo-local Skill fallback |
| [09](./09-test-release-and-production-readiness.md) | TDD, OS/client matrix, RC, signing, support | 첫 공개 release는 production `1.0.0` |
| [10](./10-product-repository-and-distribution.md) | 제품 범위·차별점·source repository·배포 channel | public package 직전 이름만 확정 필요 |
| [11](./11-cross-review-and-confirmation.md) | 요구 추적, 충돌 방어, 모순 수정, 남은 실제 검증 | 설계 완료와 출시 완료를 구분 |
| [12](./12-user-experience-walkthrough.md) | 실제 질문·답변·생성 파일·Issue·branch·PR·release 예시 | 사용자가 명령 대신 AI에게 말하는 전체 경험 |

## 사용 흐름

```text
설치 후 AI에게 자연어 요청
→ Skill이 현재 상황과 필요한 질문을 선택
→ CLI가 actual state·충돌·gate를 검사
→ project repository가 제품 의미와 진행 상태를 보존
→ PR·통합·RC를 같은 fingerprint로 검증
→ 사용자 최종 확인 후 release
```

사용자는 `.harness` 명령을 외울 필요가 없다. “새 서비스 시작하자”, “이 프로젝트 이어서 해”, “지금 뭐 해야 해?”, “이 기능 개발해”, “release 준비해”라고 요청한다.

## 현재 상태

- 완료: 제품·협업·파일·Git·AI·adapter·CLI·Plugin·보안·test·release 설계와 implementation plan
- 미완료: 실제 Go CLI, Plugin files, signed package, marketplace/public repository, production `1.0.0`
- 의도적 보류: 공개 제품 이름 1개
