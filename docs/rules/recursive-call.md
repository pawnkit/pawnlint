# recursive-call

Reports direct and mutual recursion in the project call graph

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | calls, recursion, project |

## Details

Recursive Pawn calls consume a fixed runtime stack and can overflow. The rule reports statically resolved direct and named-call cycles and skips ambiguous targets.

## Configuration

```toml
[rules]
recursive-call = "warning"
```

## Examples

### Bad

```pawn
main()
{
	Direct(2);
	First();
}

stock Direct(value)
{
	if (value > 0)
	{
		return Direct(value - 1);
	}
	return 0;
}

stock First()
{
	Second();
}

stock Second()
{
	First();
}
```

### Good

```pawn
main()
{
	First();
}

stock First()
{
	Second();
}

stock Second()
{
	return 1;
}
```
