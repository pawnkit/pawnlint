package semantic_test

import (
	"reflect"
	"testing"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestCompactSymbolsAndReferencesMatchPointerModel(t *testing.T) {
	source := []byte("enum Color { Red, Green }\nforward Float:GetValue(Float:value);\nstock Float:GetValue(Float:value) { new Float:local = value; local += 1.0; return local; }\nmain() { new Float:local; local = GetValue(local); Missing(local); }\n")
	pointerFile := parser.Parse(source)
	compactFile := parser.ParseForLinter(source)
	pointerWalk := walk.New("x.pwn", pointerFile)
	compactWalk := walk.NewCompact("x.pwn", compactFile)
	pointer := semantic.Build(pointerFile, pointerWalk)
	compact := semantic.BuildCompact(compactFile, compactWalk)
	if len(pointer.Symbols) != len(compact.Symbols) {
		t.Fatalf("symbols = %d, compact = %d", len(pointer.Symbols), len(compact.Symbols))
	}
	for index, symbol := range pointer.Symbols {
		other := compact.Symbols[index]
		if symbol.Name != other.Name || symbol.Kind != other.Kind || symbol.Ambiguous != other.Ambiguous || symbol.Constant != other.Constant || symbol.Tag != other.Tag || !reflect.DeepEqual(symbol.Tags, other.Tags) || !reflect.DeepEqual(symbol.States, other.States) || symbol.StateRaw != other.StateRaw {
			t.Fatalf("symbol %d differs\npointer: %#v\ncompact: %#v", index, symbol, other)
		}
		if symbol.Decl.Start != compactWalk.Tree.Start(other.Decl) || symbol.NameNode.Start != compactWalk.Tree.Start(other.NameNode) {
			t.Fatalf("symbol %d locations differ", index)
		}
		pointerValue, pointerKnown := pointer.ConstantValue(symbol)
		compactValue, compactKnown := compact.ConstantValue(other)
		if pointerValue != compactValue || pointerKnown != compactKnown {
			t.Fatalf("symbol %d constant differs", index)
		}
		pointerReferences := pointer.References(symbol)
		compactReferences := compact.References(other)
		if len(pointerReferences) != len(compactReferences) {
			t.Fatalf("symbol %d references = %d, compact = %d", index, len(pointerReferences), len(compactReferences))
		}
		for refIndex, reference := range pointerReferences {
			if reference.Kind != compactReferences[refIndex].Kind || reference.Node.Start != compactWalk.Tree.Start(compactReferences[refIndex].Node) {
				t.Fatalf("symbol %d reference %d differs", index, refIndex)
			}
		}
	}
	pointerIdentifiers := pointerWalk.OfKind(parser.KindIdentifier)
	compactIdentifiers := compactWalk.OfKind(parser.KindIdentifier)
	if len(pointerIdentifiers) != len(compactIdentifiers) {
		t.Fatalf("identifier count = %d, compact = %d", len(pointerIdentifiers), len(compactIdentifiers))
	}
	for index, node := range pointerIdentifiers {
		left := pointer.Resolve(node)
		right := compact.Resolve(compactIdentifiers[index])
		if (left == nil) != (right == nil) {
			t.Fatalf("identifier %q resolution differs", pointerWalk.Text(node))
		}
		if left != nil && (left.Name != right.Name || left.Kind != right.Kind) {
			t.Fatalf("identifier %q target differs", pointerWalk.Text(node))
		}
	}
	pointerUnresolved := pointer.UnresolvedReferences()
	compactUnresolved := compact.UnresolvedReferences()
	if len(pointerUnresolved) != len(compactUnresolved) {
		t.Fatalf("unresolved = %d, compact = %d", len(pointerUnresolved), len(compactUnresolved))
	}
	for index, reference := range pointerUnresolved {
		other := compactUnresolved[index]
		if reference.Kind != other.Kind || reference.Target != other.Target || reference.Node.Start != compactWalk.Tree.Start(other.Node) {
			t.Fatalf("unresolved reference %d differs", index)
		}
	}
	pointerNodes := pointerWalk.All()
	compactNodes := compactWalk.All()
	if len(pointerNodes) != len(compactNodes) {
		t.Fatalf("node count differs")
	}
	for index, node := range pointerNodes {
		other := compactNodes[index]
		pointerValue, pointerKnown := pointer.Eval(node)
		compactValue, compactKnown := compact.Eval(other)
		if pointerValue != compactValue || pointerKnown != compactKnown {
			t.Fatalf("node %d evaluation differs", index)
		}
		if !reflect.DeepEqual(pointer.ExpressionTags(node), compact.ExpressionTags(other)) {
			t.Fatalf("node %d tags differ", index)
		}
		pointerTag, pointerTagged := pointer.ExpressionTag(node)
		compactTag, compactTagged := compact.ExpressionTag(other)
		if pointerTag != compactTag || pointerTagged != compactTagged {
			t.Fatalf("node %d tag differs", index)
		}
		if pointer.Boolean(node) != compact.Boolean(other) {
			t.Fatalf("node %d boolean differs", index)
		}
		if pointer.Pure(node) != compact.Pure(other) {
			t.Fatalf("node %d purity differs", index)
		}
		for rightIndex, right := range pointerNodes {
			compactRight := compactNodes[rightIndex]
			if pointer.Equivalent(node, right) != compact.Equivalent(other, compactRight) {
				t.Fatalf("nodes %d and %d equivalence differs", index, rightIndex)
			}
			if pointer.EquivalentSyntax(node, right) != compact.EquivalentSyntax(other, compactRight) {
				t.Fatalf("nodes %d and %d syntax equivalence differs", index, rightIndex)
			}
		}
	}
}
