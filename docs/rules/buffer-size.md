# buffer-size

Reports native size arguments larger than a declared buffer

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | buffer, arrays, native, api |

## Details

Official native declarations link output arrays to capacity parameters with defaults such as sizeof(buffer). The rule reports only direct array arguments with one known dimension and a definite oversized value.

## Configuration

```toml
[rules]
buffer-size = "error"
```

## Examples

### Bad

```pawn
main()
{
    new name[24];
    new address[16];
    GetPlayerName(0, name, 25);
    GetPlayerIp(0, address, 32);
}
```

### Good

```pawn
main()
{
    new name[24];
    GetPlayerName(0, name, 24);
    GetPlayerName(0, name);
}
```
