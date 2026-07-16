# DBML and dbdiagram

Git DBML under `contracts/data/` is canonical. Discuss data through roles, journeys, policies, ownership, privacy, retention, deletion, auditing, concurrency, failure recovery, migration, and rollback—not tables alone.

The dbdiagram adapter reads its token from the configured environment variable. `db diagram` can render or push reviewed DBML. A pull always writes to `.harness/local/dbdiagram/<operation-id>/`, never to canonical files.

After a visual edit, the tool shows semantic table/column/relation/index/note differences and their policy, contract, migration, fixture, and rollback impact. It asks why the meaning changed, proposes the corresponding source change, and updates Git only after approval. Use expand/migrate/contract for destructive evolution.
