# todo-policy

Reports task comments that violate configured metadata policy

| | |
| --- | --- |
| Category | restriction |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | comments, documentation, policy, todo |

## Details

Configured tags identify task comments at the start of comment lines. Metadata uses the form TAG(owner, YYYY-MM-DD, ISSUE-123): description. Policies can allow owners, require owners, dates, or issue references, validate issue formats, and limit task age.

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `tags` | string-list | `[TODO FIXME]` | — | Task comment tags |
| `allowed-owners` | string-list | `[]` | — | Owners permitted in task metadata |
| `require-owner` | boolean | `false` | — | Require an owner |
| `require-date` | boolean | `false` | — | Require an ISO date |
| `require-issue` | boolean | `false` | — | Require an issue reference |
| `issue-pattern` | string | `[A-Z][A-Z0-9]+-[0-9]+` | — | Issue reference regular expression |
| `maximum-age-days` | integer | `0` | minimum 0; maximum 36500 | Maximum task age; zero disables age checks |
