# duplicate-global-definition

Reports global variables defined more than once in one include graph

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | variables, project, includes |

## Details

A translation unit cannot contain multiple global variables with the same name. Separate entry-point files are checked independently.
