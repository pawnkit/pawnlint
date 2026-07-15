# statement-macro-hazard

Reports statement macros unsafe in unbraced control flow

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | macros, statements, control-flow |

## Details

A function-like macro with multiple unwrapped statements, an embedded terminating semicolon, or an unmatched if can change surrounding control flow. The rule accepts single expressions, blocks, do-while wrappers, and complete if-else expansions. Uncertain, inactive, malformed, and declaration-generating macros are ignored.
