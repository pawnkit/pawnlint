# non-public-callback

Reports functions named exactly like a callback but missing the public qualifier

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | callbacks, openmp, samp, api |

## Details

The server dispatches callbacks by looking up a `public` function with the
exact callback name; a same-named function without `public` compiles cleanly
but is never called:

```pawn
OnPlayerConnect(playerid)
{
    // never runs; the server calls the public symbol, which does not exist
}
```

The rule reports functions whose name is an exact, case-sensitive match for a
known callback but that lack the `public` qualifier. No fix is offered
because a same-named private helper may be intentional.
