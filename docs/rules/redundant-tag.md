# redundant-tag

Reports tag overrides that repeat an expression's known tag

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | yes |
| Tags | tags, expressions, semantic |

## Details

A tag override is redundant when its operand already has exactly the same tag. Unknown, union, ambiguous, macro-dependent, and malformed expressions are ignored.

## Configuration

```toml
[rules]
redundant-tag = "warning"
```

## Examples

### Bad

```pawn
enum Colour
{
    Colour_Red
}

Float:KeepFloat(Float:value)
{
    return Float:value;
}

main()
{
    new Float:source = 1.0;
    new Float:copy = Float:source;
    new bool:ready = true;
    new bool:again = bool:ready;
    new Colour:colour = Colour_Red;
    new Colour:sameColour = Colour:colour;
    new Float:result = Float:KeepFloat(source);
    return copy + again + sameColour + result;
}
```

### Good

```pawn
main()
{
    new value = 1;
    new Float:converted = Float:value;
    new Float:literal = Float:1.0;
    new Float:source = 1.0;
    new untagged = _:source;
    new bool:flag = bool:value;
    return converted + literal + untagged + flag;
}
```
