package analyzer_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/analyzer"
)

func TestAnalyzeReturnsDiagnosticsAndSafeEdits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	result, err := analyzer.Analyze(context.Background(), analyzer.Request{
		WorkingDirectory: dir,
		Sources: []analyzer.Source{{
			Path:    path,
			Content: []byte("main() { if (value); { return; } }\n"),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	index := diagnosticIndex(result.Diagnostics, "empty-condition-body")
	if index < 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	if result.Diagnostics[index].Range.Start.Line != 1 || result.Diagnostics[index].Range.Start.Column < 1 {
		t.Fatalf("range = %+v", result.Diagnostics[index].Range)
	}
	if len(result.SafeEdits) != 1 || result.SafeEdits[0].DiagnosticIndex != index || len(result.SafeEdits[0].Edits) != 1 || result.SafeEdits[0].Edits[0].NewText != "" {
		t.Fatalf("safe edits = %+v", result.SafeEdits)
	}
}

func TestAnalyzeReturnsSuggestions(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(configPath, []byte("[rules]\nlegacy-include = \"warning\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := analyzer.Analyze(context.Background(), analyzer.Request{
		ConfigPath: configPath,
		Sources: []analyzer.Source{{
			Path:    "main.pwn",
			Content: []byte("#include <a_samp>\nmain() {}\n"),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	index := diagnosticIndex(result.Diagnostics, "legacy-include")
	if index < 0 || len(result.Suggestions) != 1 || result.Suggestions[0].DiagnosticIndex != index || result.Suggestions[0].Title == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAnalyzeUsesConfiguredBuildAndInMemoryEntry(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pawnlint.toml")
	path := filepath.Join(dir, "main.pwn")
	config := "[[builds]]\nname = \"feature\"\nentry = \"main.pwn\"\ndefines = [\"FEATURE\"]\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := []byte("main() {\n#if defined(FEATURE)\nif (value); { return; }\n#endif\n}\n")
	result, err := analyzer.Analyze(context.Background(), analyzer.Request{
		ConfigPath: configPath,
		Build:      "feature",
		Sources:    []analyzer.Source{{Path: path, Content: content}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if diagnosticIndex(result.Diagnostics, "empty-condition-body") < 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeUsesRequestIncludePaths(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "includes")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "helper.inc"), []byte("stock Helper() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := analyzer.Analyze(context.Background(), analyzer.Request{
		WorkingDirectory: dir,
		IncludePaths:     []string{includeDir},
		Sources: []analyzer.Source{{
			Path:    "main.pwn",
			Content: []byte("#include <helper>\nmain() {}\n"),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if diagnosticIndex(result.Diagnostics, "missing-include") >= 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeCommonIncludeMacrosAndEmit(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "include")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	includes := map[string]string{
		"open.mp.inc":          "#define _INC_open_mp\n",
		"streamer.inc":         "#define _streamer_included\n",
		"PawnPlus.inc":         "#define _PawnPlus_included\n#define __TAG(%0) %0\n",
		"vehicle-streamer.inc": "#if !defined _PawnPlus_included\n#error PawnPlus must be included first.\n#endif\n#if !defined _streamer_included\n#error Streamer must be included first.\n#endif\nenum E_STATUS { __TAG(VEHICLE_PANEL_STATUS):panel_status };\nforward Handle(__TAG(WEAPON):weaponid);\n",
	}
	for name, content := range includes {
		if err := os.WriteFile(filepath.Join(includeDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := "#include <open.mp>\n#include <streamer>\n#include <PawnPlus>\n#include <vehicle-streamer>\nmain() {\n#emit NOP\n}\n"
	result, err := analyzer.Analyze(context.Background(), analyzer.Request{
		WorkingDirectory: dir,
		IncludePaths:     []string{includeDir},
		Sources:          []analyzer.Source{{Path: filepath.Join(dir, "main.pwn"), Content: []byte(source)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeRejectsUnknownBuild(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pawnlint.toml")
	config := "[[builds]]\nname = \"main\"\nentry = \"main.pwn\"\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := analyzer.Analyze(context.Background(), analyzer.Request{
		ConfigPath: configPath,
		Build:      "missing",
		Sources:    []analyzer.Source{{Path: "main.pwn", Content: []byte("main() {}\n")}},
	})
	if err == nil {
		t.Fatal("unknown build accepted")
	}
}

func TestAnalyzeHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := analyzer.Analyze(ctx, analyzer.Request{Sources: []analyzer.Source{{Path: "main.pwn", Content: []byte("main() {}\n")}}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestAnalyzeRunsConfiguredExternalRules(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pawnlint.toml")
	scriptPath := filepath.Join(dir, "external.sh")
	config := "[[external-rules]]\nname = \"custom\"\ncommand = \"./external.sh\"\n"
	script := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n' '{\"protocolVersion\":1,\"diagnostics\":[{\"ruleId\":\"example\",\"severity\":\"warning\",\"category\":\"style\",\"message\":\"external finding\",\"path\":\"main.pwn\",\"startOffset\":0,\"endOffset\":4}]}'\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := analyzer.Analyze(context.Background(), analyzer.Request{
		ConfigPath: configPath,
		Sources:    []analyzer.Source{{Path: "main.pwn", Content: []byte("main() {}\n")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	index := diagnosticIndex(result.Diagnostics, "external/custom/example")
	if index < 0 || result.Diagnostics[index].Path != filepath.Join(dir, "main.pwn") || result.Diagnostics[index].Range.End.Offset != 4 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestAnalyzeIncrementalCache(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pawnlint.toml")
	if err := os.WriteFile(configPath, []byte("cache = \".pawnlint-cache\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	request := analyzer.Request{
		ConfigPath: configPath,
		Sources: []analyzer.Source{{
			Path:    "main.pwn",
			Content: []byte("main() { if (value); { return; } }\n"),
		}},
	}
	first, err := analyzer.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Cache.Hits != 0 || first.Cache.Misses != 1 || diagnosticIndex(first.Diagnostics, "empty-condition-body") < 0 {
		t.Fatalf("first result = %+v", first)
	}
	second, err := analyzer.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Cache.Hits != 1 || second.Cache.Misses != 0 || diagnosticIndex(second.Diagnostics, "empty-condition-body") < 0 {
		t.Fatalf("second result = %+v", second)
	}
	request.Sources[0].Content = append(request.Sources[0].Content, '\n')
	changed, err := analyzer.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Cache.Hits != 0 || changed.Cache.Misses != 1 {
		t.Fatalf("changed cache stats = %+v", changed.Cache)
	}
	entries, err := filepath.Glob(filepath.Join(dir, ".pawnlint-cache", "*.json"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("cache entries = %v, err = %v", entries, err)
	}
}

func TestAnalyzerSessionReusesProjectParsing(t *testing.T) {
	dir := t.TempDir()
	request := analyzer.Request{
		WorkingDirectory: dir,
		Sources: []analyzer.Source{{
			Path:    "main.pwn",
			Content: []byte("main() {}\n"),
		}},
	}
	session := analyzer.New()
	first, err := session.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := session.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Diagnostics) != len(second.Diagnostics) {
		t.Fatalf("diagnostics changed: first=%d second=%d", len(first.Diagnostics), len(second.Diagnostics))
	}
}

func diagnosticIndex(diagnostics []analyzer.Diagnostic, ruleID string) int {
	for index, finding := range diagnostics {
		if finding.RuleID == ruleID {
			return index
		}
	}
	return -1
}
