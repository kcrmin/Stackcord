# DBML과 dbdiagram

`contracts/data/`의 Git DBML이 원본입니다. Table만 상의하지 않고 role·journey·policy·소유권·privacy·retention·삭제·audit·동시성·실패 복구·migration·rollback을 함께 논의합니다.

dbdiagram adapter는 설정된 환경 변수에서 token을 읽습니다. `db diagram`은 검토된 DBML을 시각화하거나 push할 수 있습니다. Pull은 항상 `.harness/local/dbdiagram/<operation-id>/`에 저장하고 canonical 파일을 직접 바꾸지 않습니다.

시각 편집 후 table/column/relation/index/note의 의미 차이와 관련 policy·contract·migration·fixture·rollback 영향을 보여줍니다. 왜 의미가 바뀌었는지 확인하고 원본 변경안을 제안한 뒤 승인된 경우에만 Git을 갱신합니다. 파괴적 진화는 expand/migrate/contract를 사용합니다.
