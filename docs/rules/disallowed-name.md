# disallowed-name

Reports declarations denied by configured name policies

| | |
| --- | --- |
| Category | restriction |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | naming, restriction, policy, identifiers |

## Details

Configured policies deny exact names or regular-expression matches for selected symbol kinds, scopes, storage classes, and tags. Exclusions override a policy. Callbacks and natives require explicit opt-in.

## Configuration

```toml
[rules]
disallowed-name = "warning"
```

Set options under `[rules.disallowed-name]`.

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `policies` | object-list | `[]` | — | Name deny-list policy objects |

`policies` entry fields:

| Name | Type | Default | Constraint | Description |
| --- | --- | --- | --- | --- |
| `kinds` | string-list | — | function, global, local, parameter, enum, enum-entry, label | Symbol kinds this entry applies to; matches any kind when empty |
| `scopes` | string-list | — | global, local | Symbol scopes this entry applies to; matches any scope when empty |
| `storage` | string-list | — | automatic, const, static, public, stock, native, forward, default | Storage classes this entry applies to; matches any class when empty |
| `tags` | string-list | — | — | Definite tags this entry applies to; matches any tag when empty |
| `include-callbacks` | boolean | `false` | — | Also apply to public callback functions |
| `include-natives` | boolean | `false` | — | Also apply to native function declarations |
| `names` | string-list | — | — | Exact names this policy denies |
| `patterns` | string-list | — | — | Regular expressions this policy denies |
| `exclude` | string-list | — | — | Regular expressions that exempt a matching name |
| `reason` | string | — | — | Message appended to the diagnostic explaining the policy |

### Example

```toml
[rules.disallowed-name]
severity = "warning"
policies = [
  { kinds = ["local", "parameter"], names = ["foo", "bar"] },
  { patterns = ["^temp_"], exclude = ["^temporaryAllowed$"] }
]
```

## Examples

### Bad

```pawn
native bad_api();

new const DEBUG = 1;

enum Status
{
	UNKNOWN_STATE
}

stock ProcessData(Float:value, foo)
{
	new temp_buffer;
	new bar;
	return temp_buffer + bar + foo + _:value;
}

main()
{
	ProcessData(Float:1, 0);
}
```

### Good

```pawn
native GoodNative();
native ReservedFunction();

new const RELEASE_BUILD = 1;

enum Status
{
	STATUS_OK
}

public ProcessData()
{
	return 1;
}

stock CalculateSpeed(Float:speed, temporaryAllowed)
{
	new result = _:speed;
	return result + temporaryAllowed;
}

main()
{
	CalculateSpeed(Float:1, 0);
}
```
