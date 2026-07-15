# assignment-in-condition

An assignment used as an if/while condition is often a typo for ==

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | syntax |
| Default | enabled |
| Fixable | no |
| Tags | if, while, assignment |

## Details

A direct assignment in an if or loop condition is often a typo:

```pawn
if (playerid = BAD_PLAYER)
```

Use `==` for comparison. If the assignment is intentional, wrap it in another
pair of parentheses. This rule has no fix because either meaning may be valid.

## Configuration

```toml
[rules]
assignment-in-condition = "warning"
```

## Examples

### Bad

```pawn
main()
{
    if (playerid = GetPlayer())
    {
    }
    while (count = next())
    {
    }
    if (ready = isReady())
    {
    }
}
```

### Good

```pawn
main()
{
    if (IsPlayerConnected(playerid))
    {
    }
    while (count > 0)
    {
        count--;
    }
    do
    {
        x++;
    }
    while (x < 10);
    if ((result = compute()) != 0)
    {
    }
}
```
