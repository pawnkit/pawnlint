# identical-branches

Reports if and ternary branches with identical code

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | branches, conditionals, semantic |

## Details

Identical alternatives make the condition ineffective and often indicate a copy-and-paste mistake. Branches must have the same parsed tokens; whitespace and comments are ignored.

## Configuration

```toml
[rules]
identical-branches = "warning"
```

## Examples

### Bad

```pawn
same_if(bool:condition)
{
    if (condition)
    {
        result = 1;
    }
    else
    {
        result = 1;
    }
}

same_ternary(bool:condition)
{
    new value = condition ? result + 1 : result + 1;
}
```

### Good

```pawn
main(bool:condition)
{
    if (condition)
        result = 1;
    else
        result = 2;

    new value = condition ? 1 : 0;
}
```
