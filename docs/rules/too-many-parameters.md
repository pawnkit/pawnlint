# too-many-parameters

Reports functions with too many parameters

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | size, functions, parameters, maintainability |

## Details

Named and variadic parameters count toward the configured maximum. Public functions and known callbacks are skipped by default because their signatures may be externally fixed. Name exclusions support project-specific interfaces.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `7` | minimum 1; maximum 1000 | Maximum parameters per function |
| `include-public` | boolean | `false` | — | Check public function signatures |
| `include-callbacks` | boolean | `false` | — | Check known callback signatures |
| `exclude` | string-list | `[]` | — | Function name regular expressions to exclude |
