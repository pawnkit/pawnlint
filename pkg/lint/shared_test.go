package lint

import (
	"path/filepath"
	"testing"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawn-analysis/preprocess"
	coresource "github.com/pawnkit/pawnkit-core/source"
)

func TestSharedDiagnosticFilesCachesLineTables(t *testing.T) {
	registry := coresource.NewRegistry()
	uri := coresource.FileURI("include.inc")
	fileID := registry.Intern(uri)
	result := &analysis.Result{
		Registry: registry,
		Preprocess: &preprocess.Result{
			Files: []preprocess.FileInfo{{URI: uri.String(), Content: []byte("value\n")}},
		},
	}
	files := newSharedDiagnosticFiles(result, "main.pwn", nil)
	first := files.get(fileID)
	second := files.get(fileID)
	if first.lines != second.lines {
		t.Fatal("shared diagnostic line table was rebuilt")
	}
	if filepath.Base(first.filename) != "include.inc" {
		t.Fatalf("filename = %q", first.filename)
	}
}
