package semantic_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestEquivalentExpressions(t *testing.T) {
	src := []byte("main() { new value; if (value > 0) {} else if ((value > 0)) {} }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	ifs := tree.OfKind(parser.KindIfStatement)
	if len(ifs) != 2 || !model.Equivalent(ifs[0].Field("condition"), ifs[1].Field("condition")) {
		t.Fatal("conditions should be equivalent")
	}
	if !model.Pure(ifs[0].Field("condition")) {
		t.Fatal("comparison should be pure")
	}
}

func TestCallsAreNotPure(t *testing.T) {
	src := []byte("main() { if (Check()) {} }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	condition := tree.OfKind(parser.KindIfStatement)[0].Field("condition")
	if model.Pure(condition) {
		t.Fatal("call condition should not be pure")
	}
}

func TestEquivalentSyntaxIgnoresTrivia(t *testing.T) {
	src := []byte("main() { if (value) { result = 1; } else { /* same */ result=1; } }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	statement := tree.OfKind(parser.KindIfStatement)[0]
	if !model.EquivalentSyntax(statement.Field("consequence"), statement.Field("alternative")) {
		t.Fatal("branches should be equivalent")
	}
}

func TestEquivalentSyntaxRejectsDifferentTokens(t *testing.T) {
	src := []byte("main() { if (value) result = 1; else result = 2; }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	statement := tree.OfKind(parser.KindIfStatement)[0]
	if model.EquivalentSyntax(statement.Field("consequence"), statement.Field("alternative")) {
		t.Fatal("branches should differ")
	}
}
