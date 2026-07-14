# Suppressions

Use comments to suppress specific pawnlint findings.

## Next line

```pawn
// pawnlint-disable-next-line rule-id -- optional reason
problematic_code();
```

## Block

```pawn
// pawnlint-disable rule-id
problematic_code();
// pawnlint-enable rule-id
```

Use comma-separated IDs for several rules or `all` for every rule:

```pawn
// pawnlint-disable-next-line rule-a, rule-b
// pawnlint-disable all
```

Always pair block `disable` and `enable` directives.

## Invalid suppressions

The `unknown-suppression` rule reports:

- unknown rule IDs;
- missing rule IDs;
- unmatched `enable` directives;
- suppressions that hide no finding.

Parser errors cannot be suppressed. Prefer fixing code over suppressing a
finding, and use `all` sparingly.
