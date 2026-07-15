# deprecated-native

Reports calls to natives deprecated by the selected API

| | |
| --- | --- |
| Category | openmp |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | native, deprecated, migration, api |

## Details

The checked-in open.mp API metadata records compiler deprecation messages from the official includes. The rule reports direct calls and includes the upstream replacement guidance.

## Configuration

```toml
[rules]
deprecated-native = "warning"
```

## Examples

### Bad

```pawn
main()
{
    SendRconCommandf("echo %d", 1);
    GetRunningTimers();
}
```

### Good

```pawn
main()
{
    SendRconCommand("echo ready");
}
```
