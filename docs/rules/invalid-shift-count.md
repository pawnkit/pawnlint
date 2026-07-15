# invalid-shift-count

Reports constant shift counts outside the 32-bit cell width

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | constants, bitwise, control-flow |

## Details

Pawn cells are 32 bits wide. A constant shift count must be between 0 and 31.

## Configuration

```toml
[rules]
invalid-shift-count = "error"
```

## Examples

### Bad

```pawn
main()
{
    new first = 1 << 32;
    new second = 1 >> -1;
    first >>>= (16 + 16);

    new count = 32;
    new third = 1 << count;
}
```

### Good

```pawn
main()
{
    new count;
    new first = 1 << 0;
    new second = 1 >>> 31;
    first >>= count;

    new changed = 32;
    Change(changed);
    first >>= changed;
}
```
