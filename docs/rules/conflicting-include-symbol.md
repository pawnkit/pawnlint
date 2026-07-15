# conflicting-include-symbol

Reports namespace collisions contributed by included files

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | symbols, project, includes, namespaces |

## Details

Functions, globals, enum names, and enum entries share Pawn namespaces in combinations that can collide across files. Duplicate function and global definitions remain owned by their dedicated rules.
