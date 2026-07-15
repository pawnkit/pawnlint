# unused-global

Reports global variables unused by any translation unit

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | project |
| Default | disabled |
| Fixable | no |
| Tags | unused, variables, project |

## Details

An unreferenced global variable may be dead code. Public and underscore-prefixed globals are skipped. Initializers are not removed automatically because they may have side effects.

## Configuration

```toml
[rules]
unused-global = "warning"
```

## Examples

### Bad

```pawn
new orphaned_value;

main() {}
```

### Good

```pawn
new active_value;
new stock library_value;

main()
{
    active_value = 1;
}
```
