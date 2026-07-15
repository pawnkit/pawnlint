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

```pawn
#include "unused.inc"

main() {}
```

### Good

```pawn
#include "used.inc"

main() {
    Used();
}
```
