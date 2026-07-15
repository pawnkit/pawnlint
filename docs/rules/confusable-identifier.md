# confusable-identifier

Reports visible declarations with visually confusable names

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, suspicious, identifiers |

## Details

Pawn identifiers are ASCII. The rule reports declarations whose names differ but become identical after normalizing the visually ambiguous groups 0/O/o and 1/I/l. Only definite declarations visible in the same lexical context are compared.

## Configuration

```toml
[rules]
confusable-identifier = "warning"
```

## Examples

### Bad

```pawn
new PlayerO;
new Player0;

stock LoadI()
{
	return 1;
}

stock Load1()
{
	return 1;
}

stock Process(Iength)
{
	new length = Iength;
	new slot1;
	new slotl;
	return length + slot1 + slotl;
}

main()
{
	Process(1);
}
```

### Good

```pawn
new PlayerScore;
new PlayerScore2;

stock ReadSlot(slot)
{
	if (slot)
	{
		new Value0;
		return Value0;
	}
	else
	{
		new ValueO;
		return ValueO;
	}
}

main()
{
	ReadSlot(1);
}
```
