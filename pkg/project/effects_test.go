package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pawnkit/pawnlint/internal/semantic"
)

func TestFunctionEffects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte(`new shared;
const limit = 4;
Pure(value) { return value + 1; }
PureConstant(value) { return value + limit; }
ReadGlobal() { return shared; }
WriteGlobal() { shared = 1; }
Mutate(&value) { value = 1; }
ForwardMutation(&value) { Mutate(value); }
LocalMutation(value) { Mutate(value); return value; }
WrapPure(value) { return Pure(value); }
Unknown() { print("value"); }
`)
	model, err := Build([]Source{{Path: path, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		complete bool
		pure     bool
		reads    int
		writes   int
		mutated  []int
		calls    int
	}{
		{name: "Pure", complete: true, pure: true},
		{name: "PureConstant", complete: true, pure: true},
		{name: "ReadGlobal", complete: true, reads: 1},
		{name: "WriteGlobal", complete: true, writes: 1},
		{name: "Mutate", complete: true, mutated: []int{0}},
		{name: "ForwardMutation", complete: true, mutated: []int{0}, calls: 1},
		{name: "LocalMutation", complete: true, pure: true, calls: 1},
		{name: "WrapPure", complete: true, pure: true, calls: 1},
		{name: "Unknown"},
	}
	for _, test := range cases {
		function := onlyDeclaration(t, model, test.name, semantic.SymbolFunction)
		effects, ok := model.FunctionEffects(function)
		if !ok || effects.Complete != test.complete || effects.Pure != test.pure || len(effects.ReadsGlobals) != test.reads || len(effects.WritesGlobals) != test.writes || len(effects.Calls) != test.calls || !sameEffectIndexes(effects.MutatedParameters, test.mutated) {
			t.Errorf("%s effects = %#v, %v", test.name, effects, ok)
		}
	}
}

func TestFunctionEffectsAreImmutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	source := []byte("new shared;\nReadGlobal() { return shared; }\n")
	model, err := Build([]Source{{Path: path, Content: source}}, Options{WorkingDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	function := onlyDeclaration(t, model, "ReadGlobal", semantic.SymbolFunction)
	effects, ok := model.FunctionEffects(function)
	if !ok || len(effects.ReadsGlobals) != 1 {
		t.Fatalf("effects = %#v, %v", effects, ok)
	}
	effects.ReadsGlobals[0].Name = "changed"
	next, _ := model.FunctionEffects(function)
	if next.ReadsGlobals[0].Name != "shared" {
		t.Fatalf("cached effects were mutated: %#v", next)
	}
}

func TestFunctionEffectsPropagateAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	includePath := filepath.Join(dir, "shared.inc")
	include := []byte("new shared;\nLeaf(&value) { value = shared; }\nMiddle(&value) { Leaf(value); }\n")
	if err := os.WriteFile(includePath, include, 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "main.pwn")
	source := []byte("#include \"shared.inc\"\nTop(&value) { Middle(value); }\nRecur(value) { if (value) return Recur(value - 1); return value; }\n")
	model, err := Build([]Source{{Path: path, Content: source}}, Options{WorkingDir: dir, DefinesComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	top := onlyDeclaration(t, model, "Top", semantic.SymbolFunction)
	effects, ok := model.FunctionEffects(top)
	if !ok || !effects.Complete || effects.Pure || len(effects.ReadsGlobals) != 1 || !sameEffectIndexes(effects.MutatedParameters, []int{0}) || len(effects.Calls) != 1 {
		t.Fatalf("Top effects = %#v, %v", effects, ok)
	}
	recur := onlyDeclaration(t, model, "Recur", semantic.SymbolFunction)
	effects, ok = model.FunctionEffects(recur)
	if !ok || !effects.Complete || !effects.Pure || len(effects.Calls) != 1 {
		t.Fatalf("Recur effects = %#v, %v", effects, ok)
	}
}

func sameEffectIndexes(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
