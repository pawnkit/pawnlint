# declaration-order

Reports declarations outside the configured source order

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | declarations, order, style |

## Details

Top-level declaration groups follow the configured order. Omitted groups are ignored. Local variables can optionally be required before executable statements in each block. Inactive, uncertain, and malformed syntax is ignored.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `order` | string-list | `[include define enum constant variable native forward function]` | include, define, enum, constant, variable, native, forward, function | Top-level declaration group order |
| `locals-before-statements` | boolean | `false` | — | Require local declarations before executable statements |
