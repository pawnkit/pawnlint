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

## Options

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `units` | object-list | `[]` | — | Unit objects with name and equivalent tags |
