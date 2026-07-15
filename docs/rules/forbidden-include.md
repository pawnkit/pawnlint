# forbidden-include

Reports includes denied by project policy

| | |
| --- | --- |
| Category | restriction |
| Severity | error |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | includes, project, policy |

## Details

Configured glob patterns can prohibit dependencies by their requested include path. Inactive and uncertain directives are skipped.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `patterns` | string-list | `[]` | — | Include path glob patterns to prohibit |
