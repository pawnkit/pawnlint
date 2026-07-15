# native-argument-count

Reports calls with an impossible number of arguments for a known native

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | native, arguments, api |

## Details

Known open.mp and SA-MP native signatures define their required, optional, and variadic parameters. The rule reports only calls outside that permitted range and skips locally defined functions. Macro-concatenated argument fragments are grouped using the parser token stream.

## Configuration

```toml
[rules]
native-argument-count = "error"
```

## Examples

### Bad

```pawn
main()
{
    new version[24];
    Kick();
    GetPlayerVersion(0, version, 24, 1);
    SendRconCommand();
}
```

### Good

```pawn
main(playerid)
{
    new version[24];
    Kick(playerid);
    GetPlayerVersion(playerid, version);
    return 1;
}
```
