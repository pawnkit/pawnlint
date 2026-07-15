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
GetDefaultWorld()
{
    new world = 1;
    return world;
}
```

### Good

```pawn
GetDefaultWorld()
{
    const world = 1;
    return world;
}
```
