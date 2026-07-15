# possibly-uninitialized

Reports local variables read before an explicit assignment on every path

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | control-flow, initialization, data-flow |

## Details

Pawn zero-fills local cells, but the compiler still tracks whether a local received an explicit value. This rule reports reads that can occur before an initializer or assignment. Resolved function effects distinguish read-only and mutating arguments. Unknown reference arguments stop tracking, while API parameters marked as outputs establish assignment.

## Configuration

```toml
[rules]
possibly-uninitialized = "warning"
```

## Examples

### Bad

```pawn
GetDefaultWorld()
{
    new world;
    return world;
}
```

### Good

```pawn
GetDefaultWorld()
{
    new world = 0;
    return world;
}
```
