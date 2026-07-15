# Analyzer API

Use `pkg/analyzer` to analyze in-memory buffers without CLI output formatting.

```go
result, err := analyzer.Analyze(ctx, analyzer.Request{
	ConfigPath: "pawnlint.toml",
	Build:      "main",
	Sources: []analyzer.Source{{
		Path:    filename,
		Content: content,
	}},
})
```

`Result.Diagnostics` contains public one-based positions and byte offsets.
`SafeEdits` contains machine-safe actions. `Suggestions` contains non-applicable
guidance and optional edits. Each action references its diagnostic index.

Configuration, build contexts, variants, overrides, baselines, and diagnostic
limits match the CLI. Requested buffers replace disk content. Build file globs
do not restrict requested buffers. Cancellation is checked between contexts and
files. Without `ConfigPath`, analysis uses defaults and does not discover files.
