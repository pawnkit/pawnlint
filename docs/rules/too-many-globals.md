# too-many-globals

Reports files with too many global variables

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | size, globals, state, maintainability |

## Details

Each global variable declarator counts separately. Constants, enum entries, locals, parameters, inactive declarations, and uncertain declarations are excluded by default. Constants can be included through configuration.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `50` | minimum 1; maximum 1000000 | Maximum global variables per file |
| `include-constants` | boolean | `false` | — | Include constant globals |
