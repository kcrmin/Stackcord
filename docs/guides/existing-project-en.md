# Adopt an existing project

Say “Adopt and continue this repository.” Planning is read-only. The tool inventories Git history, dirty files, root/workspace boundaries, existing instructions, technologies, tests, CI, documents, contracts, and unknown product behavior.

`project adopt` only adds missing harness files and explicit managed sections in README/AGENTS. It preserves custom content, Git history, topology, and dirty files. Conflicting `.editorconfig` or `.gitattributes` policies block rather than overwrite.

The first baseline is characterization, not wishful redesign: map observable behavior to stable policies and scenarios, mark unknowns, write tests around critical existing behavior, then propose product changes separately. Run `context audit` after adoption and before every mutation until source relationships are coherent.
