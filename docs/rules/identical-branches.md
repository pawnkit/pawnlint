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
GetTeamColour(bool:isRedTeam)
{
    new colour;
    if (isRedTeam)
    {
        colour = 0xFF0000FF;
    }
    else
    {
        colour = 0xFF0000FF;
    }
    return colour;
}
```

### Good

```pawn
GetTeamColour(bool:isRedTeam)
{
    if (isRedTeam)
    {
        return 0xFF0000FF;
    }
    return 0x0000FFFF;
}
```
