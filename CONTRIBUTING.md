# Contributing

PawnKit is maintained by volunteers, so reviews may take a little time.

New rules, false-positive fixes, and reduced Pawn examples are welcome. A rule
change should include accepted and rejected cases, plus a fix test when the rule
can edit source.

Run the full check before opening a pull request:

```sh
task check
CGO_ENABLED=1 go test -race ./...
```

Rules should report behavior the analyzer can prove. Keep target API facts in
`pawn-api` and shared language fixtures in `pawn-corpus`. Regenerate rule docs
when configuration or diagnostics change.
