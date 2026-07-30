package project

import (
	"os"
	"path/filepath"
	"slices"
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

func TestCanonicalPrefersExtractedSampctlResources(t *testing.T) {
	root := t.TempDir()
	paths := []string{
		"gamemodes/main.pwn",
		"dependencies/sscanf/pawn.json",
		"dependencies/sscanf/sscanf2.inc",
		"dependencies/.resources/sscanf-7b5726/sscanf2.inc",
	}
	for _, relative := range paths {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `{"entry":"gamemodes/main.pwn","dependencies":["Y-Less/sscanf:v2.13.8"]}`
	if err := os.WriteFile(filepath.Join(root, "pawn.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Canonical(filepath.Join(root, "gamemodes", "main.pwn"), nil)
	if err != nil {
		t.Fatal(err)
	}
	roots := IncludeRoots(loaded)
	resource := filepath.Join(root, "dependencies", ".resources", "sscanf-7b5726")
	source := filepath.Join(root, "dependencies", "sscanf")
	if !slices.Contains(roots, resource) || slices.Contains(roots, source) {
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
