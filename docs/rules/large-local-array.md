# large-local-array

Reports large automatic arrays allocated on the Pawn stack

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | arrays, stack, memory, performance |

## Details

Automatic local arrays consume stack cells on every active call. The rule reports arrays whose constant total capacity reaches the configured threshold and skips static or unresolved dimensions.

## Configuration

```toml
[rules]
large-local-array = "warning"
```

Set options under `[rules.large-local-array]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `threshold` | integer | `1024` | minimum 1 | Minimum local array capacity to report |

## Examples

### Bad

```pawn
main()
{
	new buffer[1024];
	new matrix[40][30];
	new packed[4096 char];
}
```

### Good

```pawn
new global_buffer[2048];

main()
{
	new small[128];
	static cached[2048];
	new dynamic[UNKNOWN_SIZE];
	new packed[4000 char];
}
```
