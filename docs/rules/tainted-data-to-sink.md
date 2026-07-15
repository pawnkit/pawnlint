# tainted-data-to-sink

Reports configured input reaching a configured sensitive sink

| | |
| --- | --- |
| Category | security |
| Severity | warning |
| Analysis | project |
| Stability | preview |
| Default | disabled |
| Fixable | no |
| Tags | security, taint, input, sink, project |

## Details

Configured sources are traced through local expressions, known buffer writers, project parameters, return values, and scalar reference outputs. The rule reports flows into configured sinks and stops when resolution or transformation is uncertain.

## Configuration

```toml
[rules]
tainted-data-to-sink = "warning"
```

## Examples

### Bad

```pawn
native SQL_Query(const query[]);

public OnPluginInput(playerid, const text[])
{
    SQL_Query(text);
    return playerid;
}
```

### Good

```pawn
native SQL_Query(const query[]);

public OnPluginInput(playerid, const text[])
{
    SQL_Query("SELECT id FROM players");
    return playerid + text[0];
}
```
