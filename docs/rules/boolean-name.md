# boolean-name

Reports boolean declarations without an allowed prefix

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, style, policy, boolean |

## Details

Ordered policies select declarations with a definite bool tag. The first matching policy requires one configured prefix at a naming boundary. Exclusions override a policy. Callbacks and natives require explicit opt-in.

## Configuration

```toml
[rules]
boolean-name = "warning"
```

Set options under `[rules.boolean-name]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `policies` | object-list | `[]` | — | Ordered boolean-name selector and prefix objects |

`policies` entry fields:

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `kinds` | string-list | — | function, global, local, parameter | Symbol kinds this entry applies to; matches any kind when empty |
| `scopes` | string-list | — | global, local | Symbol scopes this entry applies to; matches any scope when empty |
| `storage` | string-list | — | automatic, const, static, public, stock, native, forward, default | Storage classes this entry applies to; matches any class when empty |
| `tags` | string-list | — | — | Definite tags this entry applies to; matches any tag when empty |
| `include-callbacks` | boolean | `false` | — | Also apply to public callback functions |
| `include-natives` | boolean | `false` | — | Also apply to native function declarations |
| `prefixes` | string-list | — | — | Allowed prefixes; the name must start with one at a naming boundary |
| `exclude` | string-list | — | — | Regular expressions that exempt a matching name |

### Example

```toml
[rules.boolean-name]
severity = "warning"
policies = [
  { kinds = ["function"], prefixes = ["Is", "Has", "Can"] },
  { kinds = ["global", "local", "parameter"], prefixes = ["is", "has", "can", "b_"] }
]
```

## Examples

### Bad

```pawn
new bool:enabled;

bool:Ready()
{
	return true;
}

stock UpdatePlayer(bool:active)
{
	new bool:visible = active;
	new bool:island = true;
	return visible && island;
}

main()
{
	UpdatePlayer(Ready());
}
```

### Good

```pawn
new bool:isEnabled;

bool:IsReady()
{
	return true;
}

stock UpdatePlayer(bool:hasAccess)
{
	new bool:canContinue = hasAccess;
	new bool:b_visible = isEnabled;
	new island = 1;
	return canContinue && b_visible && island;
}

main()
{
	UpdatePlayer(IsReady());
}
```
