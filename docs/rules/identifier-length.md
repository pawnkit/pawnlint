# identifier-length

Reports declarations outside configured name-length limits

| | |
| --- | --- |
| Category | style |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, style, policy, identifiers |

## Details

Ordered limits select declarations by kind, scope, storage, and tag. The first matching limit checks minimum and maximum ASCII identifier lengths. Callbacks and natives require explicit opt-in. One-character loop indices can be allowed when their role is definite.

## Configuration

```toml
[rules]
identifier-length = "warning"
```

Set options under `[rules.identifier-length]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `limits` | object-list | `[]` | — | Ordered identifier-length selector and limit objects |

`limits` entry fields:

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `kinds` | string-list | — | function, global, local, parameter, enum, enum-entry, label | Symbol kinds this entry applies to; matches any kind when empty |
| `scopes` | string-list | — | global, local | Symbol scopes this entry applies to; matches any scope when empty |
| `storage` | string-list | — | automatic, const, static, public, stock, native, forward, default | Storage classes this entry applies to; matches any class when empty |
| `tags` | string-list | — | — | Definite tags this entry applies to; matches any tag when empty |
| `include-callbacks` | boolean | `false` | — | Also apply to public callback functions |
| `include-natives` | boolean | `false` | — | Also apply to native function declarations |
| `minimum` | integer | — | minimum 1; maximum 1024 | Minimum allowed identifier length |
| `maximum` | integer | — | minimum 1; maximum 1024 | Maximum allowed identifier length |
| `exclude` | string-list | — | — | Regular expressions that exempt a matching name |
| `allow-loop-indices` | boolean | `true` | — | Allow single-character for-loop indices |

### Example

```toml
[rules.identifier-length]
severity = "warning"
limits = [
  { kinds = ["function", "global"], minimum = 3, maximum = 40 },
  { kinds = ["local", "parameter"], minimum = 2, maximum = 30, exclude = ["^[xyz]$"] }
]
```

## Examples

### Bad

```pawn
new id;

enum PlayerState
{
	PLAYER_STATE_NAME_TOO_LONG
}

stock Do(id)
{
	new x = id;
	new excessivelyLong;
	for (new q = 0; q < 2;)
	{
		excessivelyLong += q++;
	}
	return x + excessivelyLong;
}

main()
{
	Do(1);
}
```

### Good

```pawn
new globalScore;

enum PlayerState
{
	PLAYER_STATE_NONE,
	PLAYER_STATE_ACTIVE
}

stock LoadPlayer(playerId)
{
	new score = playerId;
	for (new i = 0; i < 2; i++)
	{
		score += i;
	}
	return score;
}

main()
{
	LoadPlayer(1);
}
```
