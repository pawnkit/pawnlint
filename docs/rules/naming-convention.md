# naming-convention

Reports declarations that violate configured naming policies

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, style, policy, identifiers |

## Details

Ordered conventions select symbols by kind, scope, storage, and tag. The first matching convention checks case, prefix, suffix, and an optional regular expression. Exclusion expressions suppress matching names. Callbacks and natives require explicit opt-in.

## Configuration

```toml
[rules]
naming-convention = "warning"
```

Set options under `[rules.naming-convention]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `conventions` | object-list | `[]` | — | Ordered naming selector and policy objects |

`conventions` entry fields:

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `kinds` | string-list | — | function, global, local, parameter, enum, enum-entry, label | Symbol kinds this entry applies to; matches any kind when empty |
| `scopes` | string-list | — | global, local | Symbol scopes this entry applies to; matches any scope when empty |
| `storage` | string-list | — | automatic, const, static, public, stock, native, forward, default | Storage classes this entry applies to; matches any class when empty |
| `tags` | string-list | — | — | Definite tags this entry applies to; matches any tag when empty |
| `include-callbacks` | boolean | `false` | — | Also apply to public callback functions |
| `include-natives` | boolean | `false` | — | Also apply to native function declarations |
| `case` | string | — | camelCase, PascalCase, snake_case, UPPER_SNAKE_CASE, lowercase, UPPERCASE | Required identifier case |
| `prefix` | string | — | — | Required literal prefix |
| `suffix` | string | — | — | Required literal suffix |
| `pattern` | string | — | — | Required regular expression |
| `exclude` | string-list | — | — | Regular expressions that exempt a matching name |

### Example

```toml
[rules.naming-convention]
severity = "warning"
conventions = [
  { kinds = ["function"], case = "PascalCase", exclude = ["^main$"] },
  { kinds = ["global"], storage = ["const"], case = "UPPER_SNAKE_CASE" },
  { kinds = ["local", "parameter"], case = "camelCase" }
]
```

## Examples

### Bad

```pawn
native bad_native();

new const max_clients = 32;
new globalScore;
new Timer:g_round_Timer;

enum player_state
{
	player_none
}

stock load_player(Float:speed, BadParameter)
{
	new BadLocal;
BadLabel:
	return BadLocal + BadParameter + _:speed;
}

main()
{
	load_player(Float:1, 0);
}
```

### Good

```pawn
native N_Load();

new const MAX_CLIENTS = 32;
new g_score;
new Timer:g_roundTimer;

enum PlayerState
{
	PLAYER_STATE_NONE,
	PLAYER_STATE_ACTIVE
}

public bad_callback()
{
	return 1;
}

stock LoadPlayer(Float:f_speed, ignored_parameter)
{
	new playerCount = _:f_speed;
good_label:
	return playerCount + ignored_parameter;
}

main()
{
	LoadPlayer(Float:1, 0);
}
```
