# redundant-initialization

Reports local initial values overwritten before any read

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | control-flow, initialization, assignments, data-flow |

## Details

A pure scalar initializer is redundant when every following path overwrites the local or exits before reading its initial value. Static locals, loop declarations, side effects, uncertain flow, and non-standalone writes are skipped.

## Configuration

```toml
[rules]
redundant-initialization = "warning"
```

## Examples

### Bad

```pawn
native Use(value);

main()
{
	Immediate();
	BothBranches(true);
	EarlyExit(true);
	DoLoop(true);
}

stock Immediate()
{
	new value = 0;
	value = 1;
	Use(value);
}

stock BothBranches(bool:condition)
{
	new value = 10;
	if (condition)
	{
		value = 1;
	}
	else
	{
		value = 2;
	}
	Use(value);
}
// …
```

### Good

```pawn
native Use(value);
native SideEffect();

main()
{
	ReadBeforeWrite();
	PartialWrite(true);
	Effectful();
	ReadWrite();
	NoOverwrite();
	LoopDeclaration();
	StaticLocal();
	SameDeclaration();
}

stock ReadBeforeWrite()
{
	new value = 0;
	Use(value);
	value = 1;
}
// …
```
