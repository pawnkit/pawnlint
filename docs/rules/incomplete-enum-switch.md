# incomplete-enum-switch

Reports enum switches that omit named values

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | switch, enums, coverage, project |

## Details

A switch over a resolved enum should cover every named value or provide a default clause. Enums with custom increments and switches with unknown cases, uncertain branches, ambiguous tags, or malformed syntax are ignored.
