# ambiguous-include

Reports includes shadowed by another matching file

| | |
| --- | --- |
| Category | portability |
| Severity | warning |
| Analysis | project |
| Default | enabled |
| Fixable | no |
| Tags | includes, project, configuration, portability |

## Details

The include resolver selects the first matching file. Multiple matches make the selected dependency sensitive to path order and local files.

## Configuration

```toml
[rules]
ambiguous-include = "warning"
```

## Examples

### Bad

```pawn
#include "shared"

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
