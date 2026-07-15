# restricted-syntax

Reports configured language and dependency restrictions

| | |
| --- | --- |
| Category | restriction |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | restriction, policy, project, syntax |

## Details

Project policy can restrict calls to exact functions or natives, include path globs, global variables, direct and mutual recursion, and goto statements. Calls and recursion are reported only when resolution is definite. Inactive and uncertain syntax is skipped.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `functions` | string-list | `[]` | — | Exact function names whose calls are restricted |
| `natives` | string-list | `[]` | — | Exact native names whose calls are restricted |
| `includes` | string-list | `[]` | — | Include path glob patterns to restrict |
| `globals` | boolean | `false` | — | Restrict global variable declarations |
| `recursion` | boolean | `false` | — | Restrict direct and mutual recursion |
| `goto` | boolean | `false` | — | Restrict goto statements |
