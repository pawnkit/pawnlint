# out-of-bounds-constant-index

Reports constant indexes outside a known array dimension

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | constants, arrays, control-flow |

## Details

A constant index must be between zero and the array size minus one. The rule checks direct indexing when both the symbol and first dimension are known.

## Configuration

```toml
[rules]
out-of-bounds-constant-index = "error"
```

## Examples

### Bad

```pawn
const SIZE = 4;
new values[SIZE];

main()
{
    values[-1] = 1;
    values[4] = 2;
    values[2 + 3] = 3;

    new index = 4;
    values[index] = 4;
}
```

### Good

```pawn
main()
{
    new teamScores[4];
    teamScores[3] = 10;
    return teamScores[3];
}
```
