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
// pawnlint-disable-next-line bogus-rule
x;

main()
{
    // pawnlint-enable never-disabled
    y;
    // pawnlint-disable-next-line
    z;
    // pawnlint-disable-next-line foo, bar
    a;
}
```

### Good

```pawn
// pawnlint-disable-next-line discarded-expression
playerid + 1;

main()
{
    // pawnlint-disable discarded-expression
    a + 1;
    b + 2;
    // pawnlint-enable discarded-expression
    c + 3;
}

// pawnlint-disable-next-line all
x + 1;
```
