# Production release

Production hardening requires stable required checks, automated critical verification, macOS and Windows journeys, Plugin-less clone continuation, contract/migration rollback, security and license review, SBOM, provenance, signatures, observability, backup/restore, operations, support, and owned warnings.

`release prepare` creates one manifest digest from exact root/workspace commits and artifact/evidence digests. The user runs the same RC in the real environment and the receipt records that digest. Any code, contract, docs, artifact, signature, evidence, or configuration identity change invalidates the candidate.

`release publish` is always approval class D and plans every public side effect before execution: signed tag, reproducible build, release artifacts, marketplace, Homebrew, WinGet, install smoke tests, notes, rollback, and support. Publishing is not complete until clean installs verify checksums and signatures.
