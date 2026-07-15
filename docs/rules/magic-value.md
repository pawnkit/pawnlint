# magic-value

Reports unexplained numeric and string literals

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | constants, literals, maintainability, policy |

## Details

Magic values hide the meaning of policy and domain constants. Named constants make reuse and changes safer. Declaration, array, and function-call exemptions keep generated data and fixed interfaces out of scope.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `check-numbers` | boolean | `true` | — | Check numeric literals |
| `check-strings` | boolean | `false` | — | Check string literals |
| `allowed-numbers` | string-list | `[-1 0 1 -1.0 0.0 1.0]` | — | Numeric values to allow |
| `allowed-strings` | string-list | `[]` | — | String contents to allow |
| `ignore-const-declarations` | boolean | `true` | — | Ignore literals in const declarations |
| `ignore-enums` | boolean | `true` | — | Ignore literals in enum declarations |
| `ignore-default-parameters` | boolean | `true` | — | Ignore parameter default values |
| `ignore-array-sizes` | boolean | `true` | — | Ignore array dimensions |
| `ignore-array-indexes` | boolean | `true` | — | Ignore array indexes |
| `ignore-array-literals` | boolean | `true` | — | Ignore values in array literals |
| `ignore-functions` | string-list | `[]` | — | Function name regular expressions whose arguments are ignored |
