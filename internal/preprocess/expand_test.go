package preprocess

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestExpandNestedFunctionMacros(t *testing.T) {
	source := []byte(`#define ADD(%0,%1) ((%0) + (%1))
#define TWICE(%0) ADD(%0, %0)

main()
{
	new value = TWICE(2);
}
`)
	parsed := parser.Parse(source)
	tree := walk.New("test.pwn", parsed)
	result := Expand(parsed, tree, 7)
	if !result.Complete || result.Parsed.HasParseErrors() {
		t.Fatalf("expansion incomplete or invalid: complete=%v diagnostics=%#v source=%s", result.Complete, result.Parsed.Diagnostics, result.Source)
	}
	expandedWalk := walk.New("test.pwn", result.Parsed)
	semantics := semantic.Build(result.Parsed, expandedWalk)
	declarators := expandedWalk.OfKind(parser.KindVariableDeclarator)
	if len(declarators) != 1 {
		t.Fatalf("declarators = %d, source=%s", len(declarators), result.Source)
	}
	if value, ok := semantics.Eval(declarators[0].Field("initializer")); !ok || value != 4 {
		t.Fatalf("value = %d, %v", value, ok)
	}
	foundChain := false
	for _, current := range result.Parsed.Tokens {
		if current.Kind != token.IntLiteral || current.Text(result.Source) != "2" || current.Origin == nil {
			continue
		}
		macros := make(map[string]bool)
		for origin := current.Origin; origin != nil; origin = origin.Parent {
			if origin.Span.File != 7 {
				t.Fatalf("origin file = %d", origin.Span.File)
			}
			macros[origin.Macro] = true
		}
		foundChain = macros["TWICE"] && macros["ADD"]
	}
	if !foundChain {
		t.Fatal("nested macro origin chain was not preserved")
	}
}

func TestExpandObjectMacroAndUndef(t *testing.T) {
	source := []byte(`#define LIMIT 10
new first = LIMIT;
#undef LIMIT
new second = LIMIT;
`)
	parsed := parser.Parse(source)
	result := Expand(parsed, walk.New("test.pwn", parsed), 1)
	if !result.Complete || result.Parsed.HasParseErrors() {
		t.Fatalf("expansion incomplete or invalid: complete=%v source=%s", result.Complete, result.Source)
	}
	expandedWalk := walk.New("test.pwn", result.Parsed)
	declarators := expandedWalk.OfKind(parser.KindVariableDeclarator)
	if len(declarators) != 2 || expandedWalk.Text(declarators[0].Field("initializer")) != "10" || expandedWalk.Text(declarators[1].Field("initializer")) != "LIMIT" {
		t.Fatalf("expanded source = %s", result.Source)
	}
}

func TestCompactInputExpansionMatchesPointerInput(t *testing.T) {
	source := []byte(`#define ADD(%0,%1) ((%0) + (%1))
#define VALUE 2
main()
{
	return ADD(VALUE, 3);
}
`)
	pointerFile := parser.Parse(source)
	pointerTree := walk.NewWithDefineContext("test.pwn", pointerFile, nil, nil, true)
	pointerResult, _ := ExpandCompactWithState(pointerFile, pointerTree, 7, nil, nil)
	compactFile := parser.ParseCompact(source, parser.ParseOptions{})
	compactTree := walk.NewCompactWithDefineContext("test.pwn", compactFile, nil, nil, true)
	compactResult, _ := ExpandCompactSyntaxWithState(compactFile, compactTree, 7, nil, nil)
	if pointerResult.Complete != compactResult.Complete || pointerResult.Changed != compactResult.Changed || !reflect.DeepEqual(pointerResult.Source, compactResult.Source) || !reflect.DeepEqual(pointerResult.Parsed, compactResult.Parsed) {
		t.Fatalf("compact input expansion differs\npointer: %#v\ncompact: %#v", pointerResult, compactResult)
	}
}

func TestUnsupportedMacroExpansionIsIncomplete(t *testing.T) {
	source := []byte(`#define STRINGIFY(%0) #%0
new value[] = STRINGIFY(test);
`)
	parsed := parser.Parse(source)
	result := Expand(parsed, walk.New("test.pwn", parsed), 1)
	if result.Complete {
		t.Fatal("stringizing expansion reported complete")
	}
}

func TestExpandBoundsChainedMacroBlowup(t *testing.T) {
	var b strings.Builder
	b.WriteString("#define M0 X\n")
	for level := 1; level <= 7; level++ {
		fmt.Fprintf(&b, "#define M%d %s\n", level, strings.TrimSuffix(strings.Repeat(fmt.Sprintf("M%d ", level-1), 10), " "))
	}
	b.WriteString("new value = M7;\n")
	source := []byte(b.String())
	parsed := parser.Parse(source)
	tree := walk.New("test.pwn", parsed)

	done := make(chan Result, 1)
	go func() { done <- Expand(parsed, tree, 1) }()
	select {
	case result := <-done:
		if result.Complete {
			t.Fatal("expansion of a chain exceeding the token cap reported complete")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("expansion did not bound chained macro growth quickly")
	}
}

func FuzzExpand(f *testing.F) {
	for _, source := range []string{
		"#define VALUE 1\nnew value = VALUE;\n",
		"#define APPLY(%0) (%0)\nmain(){return APPLY(1);}\n",
		"#define LOOP LOOP\nnew value = LOOP;\n",
	} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 100_000 {
			t.Skip()
		}
		parsed := parser.Parse([]byte(source))
		result := Expand(parsed, walk.New("fuzz.pwn", parsed), 1)
		if len(result.Source) > 10_000_000 {
			t.Fatalf("expanded source too large: %d", len(result.Source))
		}
	})
}

func BenchmarkExpandWithoutMacros(b *testing.B) {
	source := []byte("main() {\n" + strings.Repeat("new value = 1;\n", 2_000) + "}\n")
	parsed := parser.Parse(source)
	tree := walk.New("bench.pwn", parsed)
	b.ResetTimer()
	for range b.N {
		result := Expand(parsed, tree, 1)
		if result.Changed {
			b.Fatal("unexpected expansion")
		}
	}
}

func BenchmarkExpandNestedMacros(b *testing.B) {
	source := []byte("#define ADD(%0,%1) ((%0) + (%1))\n#define TWICE(%0) ADD(%0, %0)\nmain() {\n" + strings.Repeat("new value = TWICE(2);\n", 1_000) + "}\n")
	parsed := parser.Parse(source)
	tree := walk.New("bench.pwn", parsed)
	b.ResetTimer()
	for range b.N {
		result := Expand(parsed, tree, 1)
		if !result.Changed || !result.Complete {
			b.Fatal("expansion failed")
		}
	}
}
