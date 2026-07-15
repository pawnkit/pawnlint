# too-many-globals

Reports files with too many global variables

| | |
| --- | --- |
| Category | maintainability |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | size, globals, state, maintainability |

## Details

Each global variable declarator counts separately. Constants, enum entries, locals, parameters, inactive declarations, and uncertain declarations are excluded by default. Constants can be included through configuration.

## Configuration

```toml
[rules]
too-many-globals = "warning"
```

Set options under `[rules.too-many-globals]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `maximum` | integer | `50` | minimum 1; maximum 1000000 | Maximum global variables per file |
| `include-constants` | boolean | `false` | — | Include constant globals |

## Examples

### Bad

```pawn
new first, second;
static third;
new fourth;

main()
{
    return first + second + third + fourth;
}
```

### Good

```pawn
new first;
static second;
new third;

const FIRST_CONSTANT = 1;
const SECOND_CONSTANT = 2;
const THIRD_CONSTANT = 3;
const FOURTH_CONSTANT = 4;

main()
{
    new firstLocal;
    static secondLocal;
    return firstLocal + secondLocal;
}
```
