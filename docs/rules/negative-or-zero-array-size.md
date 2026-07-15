# negative-or-zero-array-size

Reports array dimensions that evaluate to zero or less

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | constants, arrays, semantic |

## Details

A declared array dimension must be greater than zero. The rule reports only dimensions that can be evaluated with certainty.

## Configuration

```toml
[rules]
negative-or-zero-array-size = "error"
```

## Examples

### Bad

```pawn
new zero[0];
new negative[-1];

main()
{
    new calculated[2 - 2];
}
```

### Good

```pawn
main()
{
    new checkpoints[4];
    return checkpoints[0];
}
```
