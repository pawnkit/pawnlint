package lint

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestFileFactsBuildsAssignmentsAndSymbolsOnDemand(t *testing.T) {
	parsed := parser.Parse([]byte("stock Work() { new value; value = 1; }\n"))
	walkModel := walk.New("test.pwn", parsed)
	semanticModel := semantic.Build(parsed, walkModel)
	ctx := &Context{
		Walk:     walkModel,
		Semantic: semanticModel,
		facts:    newFileFacts(),
	}

	if ctx.facts.assignments != nil || ctx.facts.symbols != nil {
		t.Fatal("expensive facts were built eagerly")
	}
	function := walkModel.OfKind(parser.KindFunctionDefinition)[0]
	if got := ctx.Assignments(function); len(got) != 1 {
		t.Fatalf("assignments = %d, want 1", len(got))
	}
	if ctx.facts.assignments == nil {
		t.Fatal("assignments were not cached")
	}
	if got := ctx.FunctionSymbols(function); len(got) == 0 {
		t.Fatal("function symbols are empty")
	}
	if ctx.facts.symbols == nil {
		t.Fatal("symbols were not cached")
	}
}
