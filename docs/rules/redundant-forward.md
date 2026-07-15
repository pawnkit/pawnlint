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

## Configuration

```toml
[rules]
redundant-forward = "warning"
```

## Examples

### Bad

```pawn
forward UnusedForward();

UnusedForward()
{
    return 1;
}

forward CalledAfterDefinition(value);

CalledAfterDefinition(value)
{
    return value;
}

main()
{
    return CalledAfterDefinition(1);
}
```

### Good

```pawn
forward RequiredForward(value);

main()
{
    return RequiredForward(1);
}

RequiredForward(value)
{
    return value;
}

forward ExternalCallback();

forward public ExportedByForward();

ExportedByForward()
{
    return 1;
}

forward AcrossInclude();
#include <other>
AcrossInclude()
{
    return 1;
}
// …
```
