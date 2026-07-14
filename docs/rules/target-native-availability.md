# target-native-availability

Reports open.mp-only native calls when targeting SA-MP

| | |
| --- | --- |
| Category | openmp |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | native, target, migration, api |

## Details

The selected target determines which official natives are available. The rule reports direct calls to natives declared only by open.mp modules and skips names declared by the project.
