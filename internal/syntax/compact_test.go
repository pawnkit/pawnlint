package syntax_test

import (
	"reflect"
	"testing"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func TestCompactTreeMatchesPointerTree(t *testing.T) {
	source := []byte("stock Float:Value(const input) { if (input) return 1.0; return 0.0; }\n")
	pointer := parser.Parse(source)
	compact := syntax.NewCompactTree(parser.ParseCompact(source, parser.ParseOptions{}))
	compareNodes(t, pointer.Root, compact, compact.Root())
	functions := compact.OfKind(parser.KindFunctionDefinition)
	if len(functions) != 1 {
		t.Fatalf("functions = %v", functions)
	}
	name := compact.Field(functions[0], "name")
	if compact.Text(name) != "Value" || compact.TokenKind(name) != token.Identifier || compact.Parent(name) != functions[0] {
		t.Fatalf("name = %q kind=%v parent=%v", compact.Text(name), compact.TokenKind(name), compact.Parent(name))
	}
}

func TestCompactDiagnosticsMatchPointerParser(t *testing.T) {
	for _, source := range [][]byte{
		[]byte("main( { return 1; }"),
		[]byte("main() { if (value > 0)) return value; }"),
		[]byte("#if defined FEATURE\nmain() {\n#endif\n"),
	} {
		pointer := parser.Parse(source)
		compact := parser.ParseForLinter(source)
		if pointer.Broken != compact.Broken || !reflect.DeepEqual(pointer.Diagnostics, compact.Diagnostics) {
			t.Fatalf("diagnostics differ for %q\npointer: %#v\ncompact: %#v", source, pointer.Diagnostics, compact.Diagnostics)
		}
	}
}

func compareNodes(t *testing.T, pointer *parser.Node, compact *syntax.CompactTree, node syntax.NodeID) {
	t.Helper()
	if pointer == nil || !compact.Valid(node) {
		t.Fatalf("pointer=%v compact=%v", pointer, node)
	}
	if pointer.Kind != compact.Kind(node) || pointer.Start != compact.Start(node) || pointer.End != compact.End(node) {
		t.Fatalf("pointer=%v:%d:%d compact=%v:%d:%d", pointer.Kind, pointer.Start, pointer.End, compact.Kind(node), compact.Start(node), compact.End(node))
	}
	if pointer.Tok.Kind != compact.TokenKind(node) || pointer.HasError != compact.HasError(node) || pointer.MissingSemi != compact.MissingSemi(node) {
		t.Fatalf("node %v metadata differs", pointer.Kind)
	}
	if len(pointer.Children) != compact.ChildCount(node) {
		t.Fatalf("node %v children=%d compact=%d", pointer.Kind, len(pointer.Children), compact.ChildCount(node))
	}
	for index, child := range pointer.Children {
		compareNodes(t, child, compact, compact.Child(node, index))
	}
}
