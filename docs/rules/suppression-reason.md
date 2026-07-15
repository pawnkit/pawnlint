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

## Configuration

```toml
[rules]
suppression-reason = "warning"
```

Set options under `[rules.suppression-reason]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `minimum-length` | integer | `1` | minimum 1; maximum 1000 | Minimum number of characters in a reason |
| `pattern` | string | `` | — | Regular expression required in each reason |

## Examples

### Bad

```pawn
// pawnlint-disable-next-line discarded-expression
1;

// pawnlint-disable-next-line discarded-expression -- short
2;

// pawnlint-disable-next-line discarded-expression -- legacy compatibility behavior
3;

main()
{
}
```

### Good

```pawn
// pawnlint-disable-next-line discarded-expression -- CORE-123 legacy behavior
1;

// pawnlint-disable unused-parameter -- retained because the callback signature is fixed
stock Process(value)
{
	return 1;
}
// pawnlint-enable unused-parameter

main()
{
}
```
