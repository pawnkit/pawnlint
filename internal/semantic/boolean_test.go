package semantic

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestBoolean(t *testing.T) {
	src := []byte("main() { new bool:flag; new value; if (flag == true && value < 2) return; }")
	file := parser.Parse(src)
	tree := walk.New("test.pwn", file)
	model := Build(file, tree)
	identifiers := make(map[string][]*parser.Node)
	for _, node := range tree.OfKind(parser.KindIdentifier) {
		identifiers[tree.Text(node)] = append(identifiers[tree.Text(node)], node)
	}
	if !model.Boolean(identifiers["flag"][1]) {
		t.Fatal("bool-tagged identifier was not boolean")
	}
	if model.Boolean(identifiers["value"][1]) {
		t.Fatal("untagged identifier was boolean")
	}
	if value, ok := model.BooleanLiteral(identifiers["true"][0]); !ok || !value {
		t.Fatal("true was not recognized")
	}
	for _, node := range tree.OfKind(parser.KindBinaryExpression) {
		if !model.Boolean(node) {
			t.Fatalf("binary boolean expression %q was not recognized", tree.Text(node))
		}
	}
}
