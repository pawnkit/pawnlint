# argument-value-range

Reports constant arguments outside API parameter bounds

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | enabled |
| Fixable | no |
| Tags | calls, arguments, api, contracts, range |

## Details

API metadata can define inclusive minimum and maximum values for scalar input parameters. The rule reports only definite constant violations and skips named, macro-structured, unresolved, and locally overridden calls.

## Configuration

```toml
[rules]
argument-value-range = "error"
```

## Examples

### Bad

```pawn
native SetLevel(level);
native SetFloor(value);
native SetCeiling(value);

main() {
    SetLevel(5 + 6);
    SetFloor(-6);
    SetCeiling(101);
}
```

### Good

```pawn
native SetLevel(level);
native SetFloor(value);
native SetCeiling(value);

main() {
    new value;
    SetLevel(10);
    SetFloor(-5);
    SetCeiling(value);
}
```
