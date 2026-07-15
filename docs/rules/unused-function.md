# unused-function

Reports internal functions unused by any translation unit

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | unused, functions, project |

## Details

An unreferenced internal function may be dead code. Main, externally callable functions, resolved timer targets, state-qualified functions, operators, and underscore-prefixed functions are skipped. Translation units containing parser errors are skipped.

## Configuration

```toml
[rules]
unused-function = "warning"
```

## Examples

### Bad

```pawn
OrphanedHelper() {}

main() {}
```

### Good

```pawn
Used() {}
public Exported() {}
stock LibraryFunction() {}
CMD:ExternalCommand(playerid, params[]) {}
task ScheduledTask[1000]() {}

main()
{
    Used();
}
```
