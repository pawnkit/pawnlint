package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalLoadsManifestIncludeRoots(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "include"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"entry":"gamemodes/main.pwn","pawnkit":{"schemaVersion":1,"profile":"openmp","includePaths":["include"]}}`
	if err := os.WriteFile(filepath.Join(root, "pawn.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Canonical(filepath.Join(root, "gamemodes", "main.pwn"), nil)
	if err != nil {
		t.Fatal(err)
	}
	roots := IncludeRoots(loaded)
	if len(roots) < 2 || roots[0] != filepath.Join(root, "gamemodes") {
		t.Fatalf("IncludeRoots() = %v", roots)
	}
}

func TestCanonicalAllowsManifestlessSources(t *testing.T) {
	loaded, err := Canonical(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatal("expected no canonical project")
	}
}
