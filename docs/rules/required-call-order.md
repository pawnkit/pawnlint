# required-call-order

Reports API calls missing a required earlier call

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | calls, order, api, contracts, control-flow |

## Details

API metadata can require other natives to be called earlier on every path through the same function. Calls in uncertain control flow, nested calls, and expressions without a definite evaluation order are skipped.

## Configuration

```toml
[rules]
required-call-order = "error"
```

## Examples

### Bad

```pawn
native Connect();
native Authenticate();

Test(flag) {
    Authenticate();
    Connect();
    if (flag) {
        Authenticate();
    }
}

Conditional(flag) {
    if (flag) {
        Connect();
    }
    Authenticate();
}

main() {
    Test(1);
    Conditional(1);
}
```

### Good

```pawn
native Connect();
native Authenticate();
native Query();

main() {
    Connect();
    Authenticate();
    Query();
}

ValidBranches(flag) {
    if (flag) {
        Connect();
    } else {
        Connect();
    }
    Authenticate();
}
```
