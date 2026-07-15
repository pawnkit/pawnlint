# multiple-declarations

Reports statements that declare multiple variables

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | declarations, style, variables |

## Details

Configured global and local declarations must contain one variable declarator. Multi-variable for-loop initializers can be allowed separately. Inactive, uncertain, and malformed declarations are ignored.

## Configuration

```toml
[rules]
multiple-declarations = "warning"
```

Set options under `[rules.multiple-declarations]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `scopes` | string-list | `[global local]` | global, local | Declaration scopes to check |
| `allow-for-loop` | boolean | `true` | — | Allow multiple variables in for-loop initializers |

## Examples

### Bad

```pawn
new first, second, third;

main()
{
    new localFirst, localSecond;
    for (new row = 0, column = 0; row < 3; row++, column++)
    {
        localFirst += row + column;
    }
    return first + second + third + localFirst + localSecond;
}
```

### Good

```pawn
new first;
new second;

main()
{
    new localFirst;
    static localSecond;
    new values[] = {1, 2, 3};
    return first + second + localFirst + localSecond + values[0];
}
```
