# multiple-declarations

Reports statements that declare multiple variables

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | declarations, style, variables |

## Details

Configured global and local declarations must contain one variable declarator. Multi-variable for-loop initializers can be allowed separately. Inactive, uncertain, and malformed declarations are ignored.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `scopes` | string-list | `[global local]` | global, local | Declaration scopes to check |
| `allow-for-loop` | boolean | `true` | — | Allow multiple variables in for-loop initializers |
