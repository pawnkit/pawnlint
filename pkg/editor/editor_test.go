package editor_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawn-analysis/preprocess"
	coresource "github.com/pawnkit/pawnkit-core/source"
	"github.com/pawnkit/pawnlint/pkg/editor"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestRealProjectEditorLatency(t *testing.T) {
	root := os.Getenv("PAWN_REAL_PROJECT_DIR")
	if root == "" {
		t.Skip("PAWN_REAL_PROJECT_DIR is not set")
	}
	path := filepath.Join(root, "src", "safw.pwn")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cache := project.NewParseCache()

	started := time.Now()
	if _, err := editor.DiagnoseWithCache(path, content, root, cache, nil); err != nil {
		t.Fatal(err)
	}
	t.Logf("cold editor lint took %s", time.Since(started))

	started = time.Now()
	if _, err := editor.DiagnoseWithCache(path, content, root, cache, nil); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	t.Logf("warm editor lint took %s", elapsed)
	if elapsed > time.Second {
		t.Fatalf("warm editor lint exceeded one second: %s", elapsed)
	}
}

func TestDiagnoseContextStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := editor.DiagnoseContextWithCache(ctx, "main.pwn", []byte("main() {}\n"), t.TempDir(), nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestDiagnoseNoConfig(t *testing.T) {
	dir := t.TempDir()
	src := []byte("main() {\n\treturn 1;\n}\n")

	if _, err := editor.Diagnose(filepath.Join(dir, "gamemode.pwn"), src, dir); err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
}

func TestDiagnoseParseError(t *testing.T) {
	dir := t.TempDir()
	src := []byte("main( {\n")

	diags, err := editor.Diagnose(filepath.Join(dir, "broken.pwn"), src, dir)
	if err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}

	found := false
	for _, d := range diags {
		if d.RuleID == "parse-error" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a parse-error diagnostic, got %+v", diags)
	}
}

func TestDiagnoseUsesDiscoveredConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".pawnlint.toml"), []byte("target = \"openmp\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := []byte("main() {\n\treturn 1;\n}\n")

	if _, err := editor.Diagnose(filepath.Join(dir, "gamemode.pwn"), src, dir); err != nil {
		t.Fatalf("Diagnose() error = %v", err)
	}
}

func TestDiagnoseUsesManifestIncludePaths(t *testing.T) {
	dir := t.TempDir()
	includeDir := filepath.Join(dir, "includes")
	if err := os.Mkdir(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "helper.inc"), []byte("stock Helper() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"entry":"main.pwn","pawnkit":{"schemaVersion":1,"profile":"openmp","includePaths":["includes"]}}`
	if err := os.WriteFile(filepath.Join(dir, "pawn.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	diags, err := editor.Diagnose(
		filepath.Join(dir, "main.pwn"),
		[]byte("#include <helper>\nmain() { Helper(); }\n"),
		dir,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range diags {
		if item.RuleID == "missing-include" {
			t.Fatalf("diagnostics = %+v", diags)
		}
	}
}

func TestDiagnoseDeduplicatesSharedSemanticDiagnostics(t *testing.T) {
	dir := t.TempDir()
	diags, err := editor.Diagnose(
		filepath.Join(dir, "gamemode.pwn"),
		[]byte("main() { new value; value(); }"),
		dir,
	)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, item := range diags {
		if item.RuleID == "non-callable-symbol" {
			count++
		}
		if item.RuleID == "pawn-analysis:sema/not-callable" {
			t.Fatalf("duplicate shared diagnostic: %+v", diags)
		}
	}
	if count != 1 {
		t.Fatalf("expected one non-callable diagnostic, got %+v", diags)
	}
}

func TestDiagnoseIncludesSharedArgumentCount(t *testing.T) {
	dir := t.TempDir()
	diags, err := editor.Diagnose(
		filepath.Join(dir, "gamemode.pwn"),
		[]byte("Helper(a, b) {} main() { Helper(1); }"),
		dir,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range diags {
		if item.RuleID == "pawn-analysis:sema/argument-count" {
			return
		}
	}
	t.Fatalf("shared argument-count diagnostic missing: %+v", diags)
}

func TestDiagnoseWithCacheUsesSharedAnalysisForOwnFileOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gamemode.pwn")
	content := []byte("#include \"bad.inc\"\nHelper(a, b) {}\nmain() { Helper(1); }\n")
	resolver := preprocess.MapResolver{"bad.inc": []byte("new value;\nvalue();\n")}

	shared := analysis.Analyze(content, analysis.Options{
		URI: coresource.FileURI(path), Includes: resolver, RetainExpanded: true,
	})
	diags, err := editor.DiagnoseWithCache(path, content, dir, nil, shared)
	if err != nil {
		t.Fatal(err)
	}
	foundArgCount := false
	for _, item := range diags {
		if item.Filename != path {
			t.Fatalf("diagnostic attributed to a file other than %q: %+v", path, item)
		}
		if item.RuleID == "pawn-analysis:sema/argument-count" {
			foundArgCount = true
		}
		if item.RuleID == "pawn-analysis:sema/not-callable" {
			t.Fatalf("include's own diagnostic leaked into the root file's results: %+v", diags)
		}
	}
	if !foundArgCount {
		t.Fatalf("expected the root file's own shared diagnostic, got %+v", diags)
	}
}
