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

Reuse an analyzer in long-lived processes to cache unchanged parsed files:

```go
session := analyzer.New()
result, err := session.Analyze(ctx, request)
```

`Result.Diagnostics` uses one-based positions and byte offsets. `SafeEdits`
contains applicable fixes, while `Suggestions` contains guidance and optional
edits. Actions reference their diagnostic index. `Migrations` reports renamed
rule IDs. `Cache` reports configured cache hits and misses.

Configuration, build contexts, variants, overrides, baselines, and diagnostic
limits match the CLI. Requested buffers replace disk content. Build file globs
do not restrict requested buffers. Cancellation is checked between contexts and
files. Without `ConfigPath`, analysis uses defaults and does not discover files.

Configured external rules also run through this API and honor request cancellation.
