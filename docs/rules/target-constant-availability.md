# target-constant-availability

Reports open.mp-only constants when targeting SA-MP

| | |
| --- | --- |
| Category | openmp |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | constant, target, migration, api |

## Details

The selected target determines which official constants are available. The rule reports unresolved value references declared only by open.mp modules and skips names declared by the project.
