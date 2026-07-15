# maximum-nesting

Reports functions with deeply nested control statements

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | complexity, nesting, maintainability |

## Details

Nesting depth increases for if, loop, and switch statements. Else-if chains remain at one level. Inactive and uncertain conditional-compilation branches are ignored.

## Configuration

```toml
[rules]
maximum-nesting = "warning"
```

Set options under `[rules.maximum-nesting]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `4` | minimum 1; maximum 1000 | Maximum permitted nesting depth |

## Examples

### Bad

```pawn
ProcessPlayers(limit)
{
    for (new playerid; playerid < limit; playerid++)
    {
        if (IsPlayerConnected(playerid))
        {
            while (GetPlayerState(playerid) == PLAYER_STATE_WASTED)
            {
                Kick(playerid);
            }
        }
    }
}
```

### Good

```pawn
ProcessPlayer(playerid)
{
    if (!IsPlayerConnected(playerid))
    {
        return 0;
    }
    if (GetPlayerState(playerid) == PLAYER_STATE_WASTED)
    {
        Kick(playerid);
    }
    return 1;
}
```
