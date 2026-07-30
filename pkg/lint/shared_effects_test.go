package lint

import (
	"path/filepath"
	"testing"

	analysis "github.com/pawnkit/pawn-analysis"
	coresource "github.com/pawnkit/pawnkit-core/source"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func TestSharedLeafFunctionEffects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	text := []byte("stock Leaf(&value) { value = 1; }\nstock Wrapper(&value) { Leaf(value); }\n")
	shared := analysis.Analyze(text, analysis.Options{
		URI: coresource.FileURI(path), CollectFunctionFacts: true,
	})
	model, err := project.Build(
		[]project.Source{{Path: path, Content: text}},
		project.Options{WorkingDir: dir, DefinesComplete: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	leaf := model.Declarations["Leaf"][0]
	effects, ok := sharedLeafFunctionEffects(shared, leaf)
	if !ok || !effects.Complete || len(effects.MutatedParameters) != 1 || effects.MutatedParameters[0] != 0 {
		t.Fatalf("leaf effects = %#v, found = %v", effects, ok)
	}
	wrapper := model.Declarations["Wrapper"][0]
	if _, ok := sharedLeafFunctionEffects(shared, wrapper); ok {
		t.Fatal("transitive function used direct shared effects")
	}
}
