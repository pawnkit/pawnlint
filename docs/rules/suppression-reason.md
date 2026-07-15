# suppression-reason

Reports suppression directives without an adequate reason

| | |
| --- | --- |
| Category | restriction |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | suppression, policy, documentation |

## Details

Disable directives must include a reason after --. A configurable minimum length prevents empty explanations, and an optional regular expression can require issue or ticket formats. Enable and malformed directives are handled separately.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `minimum-length` | integer | `1` | minimum 1; maximum 1000 | Minimum number of characters in a reason |
| `pattern` | string | `` | — | Regular expression required in each reason |
