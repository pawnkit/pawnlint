# unknown-suppression

Reports unknown, malformed, or unused pawnlint suppression directives

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | enabled |
| Fixable | no |
| Tags | suppression, tooling |

## Details

Reports suppression comments that are malformed, name unknown rules, have no
matching disable, or suppress no finding.

These directives can be removed safely because they do not affect reported
diagnostics. Parser errors cannot be suppressed.

## Configuration

```toml
[rules]
unknown-suppression = "warning"
```

## Examples

### Bad

```pawn
main()
{
    new value = 1;
    // pawnlint-disable-next-line rule-that-does-not-exist
    value + 1;
    return value;
}
```

### Good

```pawn
main()
{
    new value = 1;
    // pawnlint-disable-next-line discarded-expression
    value + 1;
    return value;
}
```
