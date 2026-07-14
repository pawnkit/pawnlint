# identical-branches

Reports if and ternary branches with identical code

| | |
| --- | --- |
| Category | suspicious |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | branches, conditionals, semantic |

## Details

Identical alternatives make the condition ineffective and often indicate a copy-and-paste mistake. Branches must have the same parsed tokens; whitespace and comments are ignored.
