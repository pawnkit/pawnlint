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
new values[4];

main()
{
    new index;
    values[0] = 1;
    values[3] = 2;
    values[index] = 3;

    new changed = 4;
    Change(changed);
    values[changed] = 4;
}
```
