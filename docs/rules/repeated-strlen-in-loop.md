# repeated-strlen-in-loop

Reports loop conditions that repeatedly scan an unchanged local string

| | |
| --- | --- |
| Category | performance |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | strings, loops, calls, performance |

## Details

A loop condition is evaluated on every iteration. Calling strlen there repeatedly scans the same local string when the loop neither writes it nor passes it to another call.

## Configuration

```toml
[rules]
repeated-strlen-in-loop = "warning"
```

## Examples

### Bad

```pawn
main()
{
	new text[32];
	new index;
	while (index < strlen(text))
	{
		index++;
	}
	for (index = 0; index < strlen(text); index++)
	{
		text[index];
	}
}
```

### Good

```pawn
main()
{
	new text[32];
	while (strlen(text))
	{
		text[0] = EOS;
	}
}
```
