# public-documentation

Reports selected functions without complete API documentation

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | documentation, policy, functions |

## Details

Selected functions require an adjacent Doxygen block or consecutive triple-slash comments. Documentation can require descriptions, parameter tags, and return tags. Name patterns and exclusions limit the documented surface.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `storage` | string-list | `[public]` | public, stock, native, forward | Function storage qualifiers to document |
| `include` | string-list | `[]` | — | Function name regular expressions to include |
| `exclude` | string-list | `[]` | — | Function name regular expressions to exclude |
| `minimum-description-length` | integer | `1` | minimum 1; maximum 1000 | Minimum description length |
| `require-parameters` | boolean | `true` | — | Require matching parameter tags |
| `require-return` | boolean | `false` | — | Require a return tag |
