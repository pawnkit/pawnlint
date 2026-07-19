package editor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/editor"
)

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
