# empty-condition-body

Accidental semicolon after an if/while/for condition makes the following block unconditional

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | syntax |
| Default | enabled |
| Fixable | yes |
| Tags | if, while, for |

## Details

A semicolon after a condition creates an empty body. The following block then
runs unconditionally:

```pawn
if (connected);
{
    Kick(playerid);
}
```

The rule reports this only when a block follows the empty body. The safe fix
removes the semicolon.
