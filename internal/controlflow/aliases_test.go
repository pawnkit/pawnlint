package controlflow_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestAliasesTrackCopiesAndReassignment(t *testing.T) {
	file := parser.Parse([]byte("Use(value) {} main() { new a = 1; new b = a; new c = b; Use(c); b = 2; Use(c); }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	calls := tree.OfKind(parser.KindCallExpression)
	if len(calls) != 2 {
		t.Fatalf("calls = %d", len(calls))
	}
	c := semantics.Resolve(calls[0].Field("arguments").Children[0])
	first := model.Aliases(calls[0], c)
	if !aliasNamesEqual(first, []string{"a", "b", "c"}) {
		t.Fatalf("first aliases = %v", aliasNames(first))
	}
	second := model.Aliases(calls[1], c)
	if !aliasNamesEqual(second, []string{"a", "c"}) {
		t.Fatalf("second aliases = %v", aliasNames(second))
	}
}

func TestAliasesIntersectBranches(t *testing.T) {
	file := parser.Parse([]byte("Use(value) {} main(bool:check) { new a = 1; new b; if (check) b = a; else b = a; Use(b); }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	call := tree.OfKind(parser.KindCallExpression)[0]
	b := semantics.Resolve(call.Field("arguments").Children[0])
	aliases := model.Aliases(call, b)
	if !aliasNamesEqual(aliases, []string{"a", "b"}) {
		t.Fatalf("aliases = %v", aliasNames(aliases))
	}
}

func TestAliasesDetachMutatedArguments(t *testing.T) {
	file := parser.Parse([]byte("Change(&value) { value = 2; } Use(value) {} main() { new a = 1; new b = a; Change(b); Use(a); }"))
	tree := walk.New("x.pwn", file)
	semantics := semantic.Build(file, tree)
	model := controlflow.Build(tree, semantics)
	calls := tree.OfKind(parser.KindCallExpression)
	use := calls[len(calls)-1]
	a := semantics.Resolve(use.Field("arguments").Children[0])
	aliases := model.Aliases(use, a)
	if !aliasNamesEqual(aliases, []string{"a"}) {
		t.Fatalf("aliases = %v", aliasNames(aliases))
	}
}

func aliasNames(symbols []*semantic.Symbol) []string {
	result := make([]string, len(symbols))
	for index, symbol := range symbols {
		result[index] = symbol.Name
	}
	return result
}

func aliasNamesEqual(symbols []*semantic.Symbol, names []string) bool {
	actual := aliasNames(symbols)
	if len(actual) != len(names) {
		return false
	}
	for index := range actual {
		if actual[index] != names[index] {
			return false
		}
	}
	return true
}
