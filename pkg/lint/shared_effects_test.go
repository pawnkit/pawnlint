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
	effects, ok := sharedFunctionEffects(shared, leaf)
	if !ok || !effects.Complete || len(effects.MutatedParameters) != 1 || effects.MutatedParameters[0] != 0 {
		t.Fatalf("leaf effects = %#v, found = %v", effects, ok)
	}
	wrapper := model.Declarations["Wrapper"][0]
	effects, ok = sharedFunctionEffects(shared, wrapper)
	if !ok || !effects.Complete || len(effects.MutatedParameters) != 1 ||
		effects.MutatedParameters[0] != 0 {
		t.Fatalf("wrapper effects = %#v, found = %v", effects, ok)
	}
	for id, facts := range shared.FunctionFacts {
		if len(facts.CallSites) != 0 {
			delete(shared.FunctionFacts, id)
			break
		}
	}
	effects, ok = sharedFunctionEffects(shared, wrapper)
	if !ok || effects.Complete {
		t.Fatalf("missing wrapper facts = %#v, found = %v", effects, ok)
	}
	wrapper.Node.Field("name").Start++
	effects, ok = sharedFunctionEffects(shared, wrapper)
	if !ok || effects.Complete {
		t.Fatalf("shifted wrapper = %#v, found = %v", effects, ok)
	}
}

func TestSharedFunctionEffectsKeepsIncompleteFacts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	text := []byte("stock Work(&value) { Missing(value); }\n")
	model, err := project.Build(
		[]project.Source{{Path: path, Content: text}},
		project.Options{WorkingDir: dir, DefinesComplete: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	declaration := model.Declarations["Work"][0]
	shared := analysis.Analyze(text, analysis.Options{
		URI: coresource.FileURI(path), CollectFunctionFacts: true,
	})
	effects, ok := sharedFunctionEffects(shared, declaration)
	if !ok || effects.Complete || effects.Pure {
		t.Fatalf("effects = %#v, ok = %v", effects, ok)
	}
}
