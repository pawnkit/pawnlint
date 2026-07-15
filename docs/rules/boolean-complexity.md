# boolean-complexity

Reports boolean expressions with too many logical operators

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, boolean, maintainability |

## Details

Each maximal expression chain counts its && and || operators. Parentheses, boolean negation, and tag wrappers remain part of the same chain, while nested ternary branches and comparisons are checked independently. Inactive and uncertain syntax is ignored.

## Configuration

```toml
[rules]
boolean-complexity = "warning"
```

Set options under `[rules.boolean-complexity]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `3` | minimum 1; maximum 1000 | Maximum logical operators per expression |

## Examples

### Bad

```pawn
LongChain(first, second, third, fourth)
{
    if (first && second || third && fourth)
    {
        return 1;
    }
    return 0;
}

Wrapped(first, second, third, fourth)
{
    if (first && !(second || (third && fourth)))
    {
        return 1;
    }
    return 0;
}

TernaryBranches(first, second, third, fourth, fifth, sixth, seventh, eighth)
{
    return first
        ? second && third || fourth && fifth
        : sixth || seventh && eighth || first;
}
```

### Good

```pawn
Allowed(first, second, third)
{
    if (first && second || third)
    {
        return 1;
    }
    if (first & second | third ^ 1)
    {
        return 1;
    }
    return first && second;
}

Separate(first, second, third, fourth)
{
    return first && second ? third && fourth : 0;
}
```
