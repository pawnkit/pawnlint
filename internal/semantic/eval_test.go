package semantic_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestEval(t *testing.T) {
	cases := []struct {
		expression string
		want       int64
	}{
		{"1 + 2 * 3", 7},
		{"0xFFFFFFFF", -1},
		{"!(1 == 1)", 0},
		{"1 ? 4 : 5", 4},
		{"8 >>> 1", 4},
		{"true && cellbits == 32", 1},
	}
	for _, test := range cases {
		src := []byte("main() { new value = " + test.expression + "; }\n")
		file := parser.Parse(src)
		tree := walk.New("x.pwn", file)
		model := semantic.Build(file, tree)
		declarators := tree.OfKind(parser.KindVariableDeclarator)
		value, ok := model.Eval(declarators[0].Field("initializer"))
		if !ok || value != test.want {
			t.Errorf("Eval(%q) = %d, %v; want %d", test.expression, value, ok, test.want)
		}
	}
}

func TestEvalUnknown(t *testing.T) {
	src := []byte("main() { new value = other + 1; }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	if _, ok := model.Eval(tree.OfKind(parser.KindVariableDeclarator)[0].Field("initializer")); ok {
		t.Fatal("unresolved expression should be unknown")
	}
}

func TestEvalNamedConstant(t *testing.T) {
	src := []byte("const zero = 1 - 1; main() { new value = zero; }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	declarators := tree.OfKind(parser.KindVariableDeclarator)
	value, ok := model.Eval(declarators[1].Field("initializer"))
	if !ok || value != 0 {
		t.Fatalf("named constant = %d, %v", value, ok)
	}
}

func TestEvalImplicitEnumValues(t *testing.T) {
	src := []byte("enum { Zero, One, Four = 4, Five, Slots[3], AfterSlots }; main() { new value = AfterSlots; }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	declarators := tree.OfKind(parser.KindVariableDeclarator)
	value, ok := model.Eval(declarators[0].Field("initializer"))
	if !ok || value != 9 {
		t.Fatalf("implicit enum value = %d, %v", value, ok)
	}
}

func TestEvalWithValues(t *testing.T) {
	src := []byte("main() { new input; new result = input + 2; }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	var input *semantic.Symbol
	for _, symbol := range model.Symbols {
		if symbol.Name == "input" {
			input = symbol
		}
	}
	declarators := tree.OfKind(parser.KindVariableDeclarator)
	value, ok := model.EvalWithValues(declarators[1].Field("initializer"), map[*semantic.Symbol]int64{input: 3})
	if !ok || value != 5 {
		t.Fatalf("value = %d, %v", value, ok)
	}
}
