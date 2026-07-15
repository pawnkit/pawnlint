# self-assignment

Reports assignments that store a symbol back into itself

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | enabled |
| Fixable | yes |
| Tags | assignment, semantic |

## Details

An assignment such as `value = value` does not change the value and is
usually a typo. The rule reports only direct identifiers that resolve to the
same symbol. The safe fix removes the redundant assignment while preserving its
value when it is part of a larger expression.

## Configuration

```toml
[rules]
self-assignment = "warning"
```

## Examples

### Bad

```pawn
new global_value;

main()
{
    new value;
    value = value;
    global_value = global_value;
    if (true) value = value;
    if ((value = value))
    {
        return;
    }
}
```

### Good

```pawn
new global_value;

main()
{
    new left = 1;
    new right = 2;
    left = right;
    global_value = left;
}
```
