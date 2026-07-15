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
public OnPluginInput(playerid, const text[])
{
    new query[128];
    format(query, sizeof query, "SELECT '%s'", text);
    SQL_Query(query);
    ForwardInput(text);
    new command[64];
    Plugin_Read(command, sizeof command);
    ExecuteCommand(command);
    return playerid;
}

ForwardInput(const value[])
{
    OpenPath(value);
}
```

### Good

```pawn
public OnPluginInput(playerid, const text[])
{
    SQL_Query("SELECT 1");
    new query[128];
    query = text;
    query = "SELECT 1";
    SQL_Query(query);
    new command[64];
    Plugin_Read(command, sizeof command);
    Plugin_Clean(command, sizeof command);
    ExecuteCommand(command);
    new unknown[64];
    unknown = text;
    UnknownSanitizer(unknown);
    SQL_Query(unknown);
    new rewritten[64];
    rewritten = text;
    Rewrite(rewritten);
    SQL_Query(rewritten);
    return playerid;
}

Rewrite(value[])
{
    value[0] = EOS;
}
// …
```
