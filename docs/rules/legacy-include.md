# legacy-include

Reports official SA-MP wrapper includes when targeting open.mp

| | |
| --- | --- |
| Category | openmp |
| Severity | warning |
| Analysis | syntax |
| Default | disabled |
| Fixable | no |
| Tags | include, migration, compatibility, api |

## Details

The official open.mp compatibility wrappers emit compiler warnings and direct users to include open.mp instead. The rule reports only the exact wrapper names shipped by the pinned API revision.

## Configuration

```toml
[rules]
legacy-include = "warning"
```

## Examples

### Bad

```pawn
#include <a_actor>
#include <a_http.inc>
#include <a_objects>
#include <a_players>
#include <a_samp>
#tryinclude <a_sampdb>
#include <a_vehicles>

main()
{
}
```

### Good

```pawn
#include <open.mp>
#include <a_custom>

main()
{
}
```
