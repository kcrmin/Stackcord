# Production readiness and release

## Harden continuously

Production readiness is not a final testing phase. Each change carries applicable TDD, contract, accessibility, security, observability, failure, migration, rollback, and integration evidence. Before candidate preparation, the AI reviews the complete product coverage, open risks, Git reachability, clean workspace state, submodule pointers, and reproducible build inputs.

## Prepare one candidate

Core `release prepare` deterministically binds exact root and workspace commits, artifact digests, product/docs/contract fingerprints, TDD evidence, integration evidence, and conditional migration/rollback evidence. Preparation writes a candidate but performs no public side effect. Any missing required evidence blocks preparation.

## Validate the exact candidate

The technical gate first checks the candidate against fresh current inputs. The user then runs that same candidate in the real target environment and confirms its behavior. The compact validation record names the candidate digest and its evidence; it must not contain secrets or raw conversation. `release verify` passes only when technical identity and user validation still point to the same unchanged digest.

## Choose the release profile

Core mode is the default for ordinary projects. Strict release adds SBOM, provenance, signature, supply-chain evidence, protected publication checks, and organization-oriented controls from `profiles/strict-release/`. Enable it only when the team promises or requires those guarantees.

## Publish outside the core verifier

Public repository creation, tag or artifact publication, package channels, signing identities, deployment credentials, and irreversible production actions require explicit user and organization authority. The core CLI deliberately stops at a verified candidate. An organization may connect its chosen CI/CD system or the strict profile after reviewing every external side effect and rollback path.
