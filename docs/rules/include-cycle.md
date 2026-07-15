# include-cycle

Reports cycles in the resolved include graph

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | includes, project, dependencies |

## Details

Include cycles make compilation order and preprocessor state difficult to reason about. Unresolved, inactive, optional-missing, and uncertain includes are skipped.

## Configuration

```toml
[rules]
include-cycle = "error"
```

## Examples

### Bad

```pawn
#include "invalid.pwn"

main()
{
}
```

### Good

```pawn
main()
{
}
```
