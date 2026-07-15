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

func diagnosticIndex(diagnostics []analyzer.Diagnostic, ruleID string) int {
	for index, finding := range diagnostics {
		if finding.RuleID == ruleID {
			return index
		}
	}
	return -1
}
