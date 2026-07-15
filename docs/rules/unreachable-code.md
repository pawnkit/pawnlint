# unreachable-code

Reports statements that cannot be executed

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | enabled |
| Fixable | no |
| Tags | control-flow, unreachable, correctness |

## Details

Code after an unconditional return, jump, or non-terminating loop cannot execute. The rule skips functions with malformed or uncertain control flow.

## Configuration

```toml
[rules]
unreachable-code = "warning"
```

## Examples

### Bad

```pawn
GetDefaultScore()
{
    return 10;
    return 0;
}
```

### Good

```pawn
main()
{
    new value;
    if (value)
        return;
    value = 1;
}

with_jump()
{
    goto done;
done:
    return;
}

with_break()
{
    while (true)
    {
        break;
    }
    return;
}
```
