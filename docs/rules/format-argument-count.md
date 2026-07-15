# format-argument-count

Reports literal format strings whose placeholders and arguments differ

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | format, arguments, native, api |

## Details

Pawn format placeholders consume variadic arguments in order. The rule checks direct calls to known formatted natives when the format is a literal and every specifier is recognized. Dynamic strings and named arguments are skipped.

## Configuration

```toml
[rules]
format-argument-count = "error"
```

## Examples

### Bad

```pawn
main()
{
    new output[64];
    printf("%d %s", 1);
    printf("%d", 1, 2);
    format(output, sizeof (output), "%d");
    SendClientMessage(0, -1, "value %d");
}
```

### Good

```pawn
#define C_ORANGE "{FF9900}"

main()
{
    new output[64];
    printf("value %05d %.2f %% %q", 1, 1.0, "text");
    printf("%d" " %s", 1, "text");
    format(output, sizeof (output), "%d", 1);
    printf(""C_ORANGE"value %d", 1);
}
```
