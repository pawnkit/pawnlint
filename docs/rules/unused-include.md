# unused-include

Reports includes with no contribution to a complete build

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | includes, unused, project, dependencies |

## Details

An include is reported only when its declarations are unused and removal has no known macro, directive, unresolved, public, native, forward, or shared dependency effect. A complete configured build context is required.

## Configuration

```toml
[rules]
unused-include = "warning"
```

## Examples

### Bad

`example-invalid.inc`:

```pawn
stock SetExampleMode()
{
    return 1;
}
```

`example-invalid.pwn`:

```pawn
#include "example-invalid.inc"

main()
{
	return 1;
}
```

### Good

`example-valid.inc`:

```pawn
stock SetExampleMode()
{
    return 1;
}
```

`example-valid.pwn`:

```pawn
#include "example-valid.inc"

main()
{
	SetExampleMode();
	return 1;
}
```
