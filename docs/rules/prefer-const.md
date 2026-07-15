# prefer-const

Reports initialized local variables that are never modified

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | const, variables, semantic |

## Details

Initialized local scalar variables should be const when every use is read-only. Unused variables, static declarations, arrays, unresolved call arguments, and uncertain syntax are ignored.

## Configuration

```toml
[rules]
prefer-const = "warning"
```

## Examples

### Bad

```pawn
ReadValue(value)
{
    return value;
}

main()
{
    new literal = 1;
    new expression = 2 + 3;
    new runtime = ReadValue(literal);
    new byValue = 4;
    ReadValue(byValue);
    return literal + expression + runtime + byValue;
}
```

### Good

```pawn
ReadValue(value)
{
    return value;
}

MutateValue(&value)
{
    value++;
}
// …
```
