package controlflow

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestEvalCachesNodeResults(t *testing.T) {
	file := parser.Parse([]byte("main() { new value = 2; if (value == 2) {} }"))
	tree := walk.New("main.pwn", file)
	model := Build(tree, semantic.Build(file, tree))
	node := tree.OfKind(parser.KindIfStatement)[0].Field("condition")

	if _, ok := model.Eval(node); !ok {
		t.Fatal("condition was not evaluated")
	}
	if _, ok := model.Eval(node); !ok {
		t.Fatal("cached condition was not evaluated")
	}
	if len(model.eval) != 1 {
		t.Fatalf("cached evaluations = %d, want 1", len(model.eval))
	}
}
