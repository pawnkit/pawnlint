# self-assignment

Reports assignments that store a symbol back into itself

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | enabled |
| Fixable | yes |
| Tags | assignment, semantic |

## Details

An assignment such as `value = value` does not change the value and is
usually a typo. The rule reports only direct identifiers that resolve to the
same symbol. The safe fix removes the redundant assignment while preserving its
value when it is part of a larger expression.
