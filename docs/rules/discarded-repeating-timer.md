# discarded-repeating-timer

Reports repeating SetTimer/SetTimerEx handles discarded before they can be killed

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | timer, resource, leak |

## Details

A repeating timer runs forever unless its handle is passed to KillTimer.
Discarding the handle immediately, as a standalone statement, means nothing
can ever stop it:

```pawn
SetTimer("Tick", 1000, true);
```

Store the handle so it can be killed later:

```pawn
tickTimer = SetTimer("Tick", 1000, true);
```

The rule reports only direct standalone calls whose `repeating` argument is
the constant `true`. A one-shot timer (`repeating` is `false`) needs no handle,
and is not reported. No fix is offered because a name for the stored handle
cannot be inferred.
