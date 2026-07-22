package walk_test

import (
	"reflect"
	"slices"
	"testing"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func TestCompactModelMatchesPointerModel(t *testing.T) {
	source := []byte("#define LOCAL\n#if defined LOCAL\nstock run() { new active; return active; }\n#else\nstock run() { new inactive; return inactive; }\n#endif\n#undef LOCAL\n#if defined EXPORTED\nnew exported;\n#endif\n")
	pointerFile := parser.Parse(source)
	compactFile := parser.ParseForLinter(source)
	if compactFile.Tokens != nil || compactFile.Trivia != nil {
		t.Fatal("linter parse retained tokens or trivia")
	}
	snapshotOffset := len(source) - len("#if defined EXPORTED\nnew exported;\n#endif\n")
	snapshots := []walk.DefineSnapshot{{Offset: snapshotOffset, Defines: []string{"EXPORTED"}}}
	pointer := walk.NewWithDefineContext("x.pwn", pointerFile, []string{"CONFIGURED"}, snapshots, true)
	compact := walk.NewCompactWithDefineContext("x.pwn", compactFile, []string{"CONFIGURED"}, snapshots, true)
	pointerNodes := pointer.All()
	compactNodes := compact.All()
	if len(pointerNodes) != len(compactNodes) {
		t.Fatalf("nodes = %d, compact = %d", len(pointerNodes), len(compactNodes))
	}
	for index, node := range pointerNodes {
		compactNode := compactNodes[index]
		if node.Kind != compact.Tree.Kind(compactNode) || node.Start != compact.Tree.Start(compactNode) || node.End != compact.Tree.End(compactNode) {
			t.Fatalf("node %d differs", index)
		}
		if pointer.Text(node) != compact.Text(compactNode) || pointer.Range(node) != compact.Range(compactNode) {
			t.Fatalf("node %d source data differs", index)
		}
		if pointer.Uncertain(node) != compact.Uncertain(compactNode) || pointer.Inactive(node) != compact.Inactive(compactNode) {
			t.Fatalf("node %d state differs", index)
		}
		if pointer.IsInsideConditionalBranch(node) != compact.IsInsideConditionalBranch(compactNode) {
			t.Fatalf("node %d conditional ancestry differs", index)
		}
		compareCompactRelation(t, pointerNodes, compactNodes, pointer.Parent(node), compact.Parent(compactNode), "parent")
		compareCompactRelation(t, pointerNodes, compactNodes, pointer.PrevSibling(node), compact.PrevSibling(compactNode), "previous sibling")
		compareCompactRelation(t, pointerNodes, compactNodes, pointer.NextSibling(node), compact.NextSibling(compactNode), "next sibling")
		compareCompactRelation(t, pointerNodes, compactNodes, pointer.EnclosingFunction(node), compact.EnclosingFunction(compactNode), "enclosing function")
	}
	for _, offset := range []int{0, snapshotOffset - 1, snapshotOffset + 1, len(source) + 1} {
		if !reflect.DeepEqual(pointer.KnownDefinesAt(offset), compact.KnownDefinesAt(offset)) {
			t.Fatalf("defines at %d differ", offset)
		}
	}
	for _, kind := range []parser.Kind{parser.KindFunctionDefinition, parser.KindVariableDeclarator, parser.KindReturnStatement} {
		if len(pointer.OfKind(kind)) != len(compact.OfKind(kind)) {
			t.Fatalf("kind %v count differs", kind)
		}
	}
}

func TestEndinputStopsPointerAndCompactModels(t *testing.T) {
	source := []byte("#if defined GUARD\n#endinput\n#endif\n#define AFTER\nnew value;\n")
	pointer := walk.NewWithDefineContext("x.inc", parser.Parse(source), []string{"GUARD"}, nil, true)
	compact := walk.NewCompactWithDefineContext("x.inc", parser.ParseForLinter(source), []string{"GUARD"}, nil, true)
	pointerValue := pointer.OfKind(parser.KindVariableDeclarator)
	compactValue := compact.OfKind(parser.KindVariableDeclarator)
	if len(pointerValue) != 1 || !pointer.Inactive(pointerValue[0]) {
		t.Fatalf("pointer declaration is active: %#v", pointerValue)
	}
	if len(compactValue) != 1 || !compact.Inactive(compactValue[0]) {
		t.Fatalf("compact declaration is active: %#v", compactValue)
	}
	if slices.Contains(pointer.KnownDefinesAt(len(source)+1), "AFTER") || slices.Contains(compact.KnownDefinesAt(len(source)+1), "AFTER") {
		t.Fatal("define after #endinput became active")
	}
}

func TestCompilerConstantsMatchPointerAndCompactModels(t *testing.T) {
	source := []byte("#if cellbits == 32 && charbits == 8\nnew active;\n#else\nnew inactive;\n#endif\n")
	pointer := walk.New("x.inc", parser.Parse(source))
	compact := walk.NewCompact("x.inc", parser.ParseForLinter(source))
	pointerValues := pointer.OfKind(parser.KindVariableDeclarator)
	compactValues := compact.OfKind(parser.KindVariableDeclarator)
	if len(pointerValues) != 2 || pointer.Inactive(pointerValues[0]) || !pointer.Inactive(pointerValues[1]) {
		t.Fatalf("pointer branch state is wrong")
	}
	if len(compactValues) != 2 || compact.Inactive(compactValues[0]) || !compact.Inactive(compactValues[1]) {
		t.Fatalf("compact branch state is wrong")
	}
}

func compareCompactRelation(t *testing.T, pointerNodes []*parser.Node, compactNodes []syntax.NodeID, pointer *parser.Node, compact syntax.NodeID, relation string) {
	t.Helper()
	if pointer == nil {
		if compact != syntax.NoNode {
			t.Fatalf("%s differs", relation)
		}
		return
	}
	for index, node := range pointerNodes {
		if node == pointer {
			if compactNodes[index] != compact {
				t.Fatalf("%s differs", relation)
			}
			return
		}
	}
	t.Fatalf("pointer %s was not indexed", relation)
}
