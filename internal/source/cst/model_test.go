package cst_test

import (
	"testing"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source/cst"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestPointerAndCompactModelsMatch(t *testing.T) {
	source := []byte("#define ENABLED\n#if defined ENABLED\nstock Float:Value(Float:input) { return input + 1.0; }\n#else\nstock Float:Value(Float:input) { return input; }\n#endif\n")
	pointerFile := parser.Parse(source)
	compactFile := parser.ParseCompact(source, parser.ParseOptions{})
	pointer := cst.Pointer(walk.NewWithDefineContext("x.pwn", pointerFile, nil, nil, true))
	compact := cst.Compact(walk.NewCompactWithDefineContext("x.pwn", compactFile, nil, nil, true))
	compareNodes(t, pointer, pointer.Root(), compact, compact.Root())
	if pointer.TokenCount() != compact.TokenCount() {
		t.Fatalf("token count differs")
	}
	for index := 0; index < pointer.TokenCount(); index++ {
		left, right := pointer.Token(index), compact.Token(index)
		if left.Kind() != right.Kind() || left.Start() != right.Start() || left.End() != right.End() || left.Text() != right.Text() || left.EndsLine() != right.EndsLine() {
			t.Fatalf("token %d differs", index)
		}
	}
	for _, kind := range []parser.Kind{parser.KindFunctionDefinition, parser.KindIdentifier, parser.KindReturnStatement} {
		left := pointer.OfKind(kind)
		right := compact.OfKind(kind)
		if len(left) != len(right) {
			t.Fatalf("kind %v count differs", kind)
		}
		for index := range left {
			if left[index].Text() != right[index].Text() || pointer.Inactive(left[index]) != compact.Inactive(right[index]) || pointer.Uncertain(left[index]) != compact.Uncertain(right[index]) {
				t.Fatalf("kind %v node %d differs", kind, index)
			}
		}
	}
}

func compareNodes(t *testing.T, leftModel *cst.Model, left cst.Node, rightModel *cst.Model, right cst.Node) {
	t.Helper()
	if !left.Valid() || !right.Valid() || left.Kind() != right.Kind() || left.Start() != right.Start() || left.End() != right.End() || left.HasError() != right.HasError() || left.MissingSemi() != right.MissingSemi() || left.TokenKind() != right.TokenKind() || left.TokenText() != right.TokenText() || left.Text() != right.Text() || left.ChildCount() != right.ChildCount() || leftModel.Range(left) != rightModel.Range(right) {
		t.Fatalf("nodes differ: %v %q and %v %q", left.Kind(), left.Text(), right.Kind(), right.Text())
	}
	for index := 0; index < left.ChildCount(); index++ {
		compareNodes(t, leftModel, left.Child(index), rightModel, right.Child(index))
	}
}
