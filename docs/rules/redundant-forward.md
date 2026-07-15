# redundant-forward

Reports forward declarations that are not needed before a definition

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | functions, forward, declarations |

## Details

A forward declaration is redundant when the same file defines a non-public function without an earlier call that needs the declaration. Includes, macro invocations, unresolved calls, state functions, and declarations with storage effects are ignored.
