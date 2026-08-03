# Performance

Run the local benchmarks with:

```console
task bench
```

`BenchmarkRules` reports one row for every registered rule. Each row includes
time, bytes, and allocations, so a slow rule can be measured without the
other rules hiding its cost.

For a quick comparison, keep the command and fixture unchanged:

```console
go test ./internal/bench -run '^$' -bench '^BenchmarkRules$' -benchmem -count=5
```

Benchmark numbers depend on the machine. Use repeated runs and `benchstat`
before treating a small change as meaningful. The real-world benchmark covers
larger projects and is separate from these focused measurements.
