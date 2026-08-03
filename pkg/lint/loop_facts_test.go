package lint

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestLoopFactsGroupCallsByNearestLoop(t *testing.T) {
	file := parser.Parse([]byte("main() { while (outer) { Work(); while (inner) { Work(); } } }\n"))
	tree := walk.New("loops.pwn", file)
	ctx := &Context{Walk: tree}
	loops := tree.OfKind(parser.KindWhileStatement)
	calls := ctx.LoopCalls()
	if len(loops) != 2 || len(calls[loops[0]]) != 1 || len(calls[loops[1]]) != 1 {
		t.Fatalf("loop calls = %#v", calls)
	}
	if ctx.EnclosingLoop(tree.OfKind(parser.KindCallExpression)[0]) != loops[0] {
		t.Fatal("outer call has the wrong enclosing loop")
	}
}
