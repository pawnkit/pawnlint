package lint

import (
	"path/filepath"
	"testing"

	analysis "github.com/pawnkit/pawn-analysis"
	coresource "github.com/pawnkit/pawnkit-core/source"
)

func TestEngineCachesSharedFunctionIndex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.pwn")
	shared := analysis.Analyze([]byte("stock Work() {}\n"), analysis.Options{
		URI: coresource.FileURI(path), CollectFunctionFacts: true,
	})
	engine := NewEngine(nil)
	engine.SharedAnalysis = shared

	first := engine.sharedFunctionIndex()
	second := engine.sharedFunctionIndex()
	if first == nil || first != second {
		t.Fatalf("shared index pointers = %p, %p", first, second)
	}

	engine.SharedAnalysis = analysis.Analyze([]byte("stock Other() {}\n"), analysis.Options{
		URI: coresource.FileURI(path), CollectFunctionFacts: true,
	})
	third := engine.sharedFunctionIndex()
	if third == nil || third == first {
		t.Fatalf("replacement shared index pointers = %p, %p", first, third)
	}
}
