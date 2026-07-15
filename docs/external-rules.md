# External rules

External rules use a versioned JSON process protocol. Each configured command runs once per build context, reads one request from stdin, and writes one response to stdout.

```toml
[[external-rules]]
name = "project"
command = "./tools/project-rules"
arguments = ["lint"]
timeout-ms = 10000

[external-rules.configuration]
mode = "strict"
```

Relative command paths resolve from the configuration directory. Names must be unique and cannot contain path separators. The default timeout is 10 seconds; the maximum is 300 seconds.

## Request

```json
{
  "protocolVersion": 1,
  "workingDirectory": ".",
  "build": "main",
  "target": "openmp",
  "defines": ["FEATURE"],
  "configuration": {"mode": "strict"},
  "targets": ["gamemodes/main.pwn"],
  "files": [{"path": "gamemodes/main.pwn", "content": "main() {}\n"}]
}
```

`files` contains the project context. `targets` contains the files requested for linting. Paths use forward slashes and are relative to the configuration directory when possible.

## Response

```json
{
  "protocolVersion": 1,
  "diagnostics": [{
    "ruleId": "example",
    "severity": "warning",
    "category": "style",
    "message": "example finding",
    "path": "gamemodes/main.pwn",
    "startOffset": 0,
    "endOffset": 4
  }]
}
```

Severities are `error`, `warning`, `info`, or `hint`. Categories match pawnlint categories. Offsets are zero-based bytes. Paths must match request files. Rule IDs become `external/<name>/<ruleId>`.

Diagnostics may include `code`, `related`, `fix`, and `suggestions`. Related locations and edits must use the diagnostic file. Edits contain `startOffset`, `endOffset`, and `newText`; fixes and suggestions also require `description`.

Invalid output, non-zero exits, cancellation, timeouts, and responses over 8 MiB fail analysis. External diagnostics are not cached.
