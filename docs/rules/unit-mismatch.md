# unit-mismatch

Reports operations between incompatible configured unit tags

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | units, tags, conversions, semantic |

## Details

Configured unit groups map equivalent Pawn tags to the same unit. Assignments, initializers, returns, addition, subtraction, comparisons, and conditional branches must use the same configured unit. Untagged, unknown, union, macro-derived, and uncertain expressions are ignored.

## Configuration

```toml
[rules]
unit-mismatch = "warning"
```

Set options under `[rules.unit-mismatch]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `units` | object-list | `[]` | — | Unit objects with name and equivalent tags |

`units` entry fields:

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `name` | string | — | — |  |
| `tags` | string-list | — | — |  |

## Examples

### Bad

```pawn
Milliseconds:GetDelay(Seconds:seconds)
{
    new Milliseconds:initialized = seconds;
    new Milliseconds:assigned;
    assigned = seconds;
    assigned += seconds;
    if (assigned < seconds) return seconds;
    assigned = assigned + seconds;
    assigned = true ? assigned : seconds;
    return seconds;
}
```

### Good

```pawn
DurationMs:GetDelay(Milliseconds:left, DurationMs:right, value)
{
    new DurationMs:initialized = left;
    initialized = right;
    initialized += left;
    if (initialized > right) return left;
    initialized = initialized + right;
    initialized = value;
    initialized = Milliseconds:value;
    return right;
}
```
