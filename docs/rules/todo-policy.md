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

## Configuration

```toml
[rules]
todo-policy = "warning"
```

Set options under `[rules.todo-policy]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `tags` | string-list | `[TODO FIXME]` | — | Task comment tags |
| `allowed-owners` | string-list | `[]` | — | Owners permitted in task metadata |
| `require-owner` | boolean | `false` | — | Require an owner |
| `require-date` | boolean | `false` | — | Require an ISO date |
| `require-issue` | boolean | `false` | — | Require an issue reference |
| `issue-pattern` | string | `[A-Z][A-Z0-9]+-[0-9]+` | — | Issue reference regular expression |
| `maximum-age-days` | integer | `0` | minimum 0; maximum 36500 | Maximum task age; zero disables age checks |

### Example

```toml
[rules.todo-policy]
severity = "warning"
tags = ["TODO", "FIXME"]
allowed-owners = ["alice", "team-core"]
require-owner = true
require-date = true
require-issue = true
issue-pattern = "[A-Z]+-[0-9]+"
maximum-age-days = 90
```

## Examples

### Bad

```pawn
// TODO: missing metadata
// TODO(charlie, 2999-01-01, CORE-123): owner is not allowed
// TODO(alice, CORE-124): date is missing
// TODO(alice, 2026-99-99, CORE-125): date is invalid
// TODO(alice, 2999-01-01): issue is missing
// FIXME(bob, 2000-01-01, CORE-126): task is stale

main()
{
}
```

### Good

```pawn
// TODO(alice, 2999-01-01, CORE-123): replace the compatibility path
// FIXME(bob, 2999-01-02, UI-42): remove the temporary layout

main()
{
}
```
