package semantic_test

import (
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestBuildResolvesLocalReferences(t *testing.T) {
	src := []byte("main() { new value; value = value + 1; }\n")
	file := parser.Parse(src)
	model := semantic.Build(file, walk.New("x.pwn", file))
	var local *semantic.Symbol
	for _, symbol := range model.Symbols {
		if symbol.Name == "value" && symbol.Kind == semantic.SymbolLocal {
			local = symbol
		}
	}
	if local == nil {
		t.Fatal("local symbol not collected")
	}
	if got := len(model.References(local)); got != 2 {
		t.Fatalf("references = %d", got)
	}
}

func TestBuildPreservesUnresolvedReferenceKinds(t *testing.T) {
	src := []byte("main() { External(); new value = external_value; }\n")
	file := parser.Parse(src)
	tree := walk.New("test.pwn", file)
	model := semantic.Build(file, tree)
	references := model.UnresolvedReferences()
	if len(references) != 2 {
		t.Fatalf("unresolved references = %d, want 2", len(references))
	}
	if references[0].Target != semantic.ReferenceFunction || references[0].Kind != semantic.ReferenceCall {
		t.Fatalf("function reference = %#v", references[0])
	}
	if references[1].Target != semantic.ReferenceValue || references[1].Kind != semantic.ReferenceRead {
		t.Fatalf("value reference = %#v", references[1])
	}
}

func TestBuildLeavesDuplicateDeclarationsUnresolved(t *testing.T) {
	src := []byte("main() { new value; new value; value = 1; }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	identifiers := tree.OfKind(parser.KindIdentifier)
	if model.Resolve(identifiers[len(identifiers)-1]) != nil {
		t.Fatal("duplicate declaration should be unresolved")
	}
}

func TestBuildCapturesSingleAndUnionTags(t *testing.T) {
	src := []byte("main(bool:flag, {Float,_}:value) { new Float:result; result = value; }\n")
	file := parser.Parse(src)
	model := semantic.Build(file, walk.New("x.pwn", file))
	tags := make(map[string][]string)
	for _, symbol := range model.Symbols {
		tags[symbol.Name] = symbol.Tags
	}
	if got := tags["flag"]; len(got) != 1 || got[0] != "bool" {
		t.Fatalf("flag tags = %v", got)
	}
	if got := tags["value"]; len(got) != 2 || got[0] != "Float" || got[1] != "_" {
		t.Fatalf("value tags = %v", got)
	}
	if got := tags["result"]; len(got) != 1 || got[0] != "Float" {
		t.Fatalf("result tags = %v", got)
	}
}

func TestBuildModelsStateQualifiedFunctions(t *testing.T) {
	src := []byte("Mode()<idle> {} Mode()<running> {} Fallback() {} Fallback()<idle> {} Duplicate()<idle> {} Duplicate()<idle> {}\n")
	file := parser.Parse(src)
	model := semantic.Build(file, walk.New("x.pwn", file))
	var modes, fallbacks, duplicates []*semantic.Symbol
	for _, symbol := range model.Symbols {
		switch symbol.Name {
		case "Mode":
			modes = append(modes, symbol)
		case "Fallback":
			fallbacks = append(fallbacks, symbol)
		case "Duplicate":
			duplicates = append(duplicates, symbol)
		}
	}
	if len(modes) != 2 || modes[0].Ambiguous || modes[1].Ambiguous || modes[0].States[0] != "idle" || modes[1].States[0] != "running" {
		t.Fatalf("state variants = %#v", modes)
	}
	if len(fallbacks) != 2 || fallbacks[0].Ambiguous || fallbacks[1].Ambiguous {
		t.Fatalf("fallback variants = %#v", fallbacks)
	}
	if len(duplicates) != 2 || !duplicates[0].Ambiguous || !duplicates[1].Ambiguous {
		t.Fatalf("duplicate variants = %#v", duplicates)
	}
}

func TestExpressionTagsPropagateConservatively(t *testing.T) {
	src := []byte("Float:GetValue() { return Float:1; } bool:IsReady() { return true; } main(Float:source, Float:values[]) { new Float:a = source + Float:1; new Float:b = values[0]; if (IsReady() == true) {} GetValue(); }\n")
	file := parser.Parse(src)
	tree := walk.New("x.pwn", file)
	model := semantic.Build(file, tree)
	want := map[parser.Kind][]string{
		parser.KindTaggedExpression:    {"Float"},
		parser.KindSubscriptExpression: {"Float"},
	}
	for kind, tags := range want {
		nodes := tree.OfKind(kind)
		if len(nodes) == 0 {
			t.Fatalf("no %s nodes", kind)
		}
		got := model.ExpressionTags(nodes[len(nodes)-1])
		if !sameStrings(got, tags) {
			t.Fatalf("%s tags = %v", kind, got)
		}
	}
	var floatCall, boolCall *parser.Node
	for _, call := range tree.OfKind(parser.KindCallExpression) {
		switch tree.Text(call.Field("function")) {
		case "GetValue":
			floatCall = call
		case "IsReady":
			boolCall = call
		}
	}
	if !sameStrings(model.ExpressionTags(floatCall), []string{"Float"}) {
		t.Fatalf("call tags = %v", model.ExpressionTags(floatCall))
	}
	if !model.Boolean(boolCall) {
		t.Fatal("bool-returning call was not boolean")
	}
	binaries := tree.OfKind(parser.KindBinaryExpression)
	if len(binaries) == 0 || !model.Boolean(binaries[len(binaries)-1]) {
		t.Fatal("comparison was not boolean")
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func TestBuildCapturesNamedEnumConstantAndTags(t *testing.T) {
	src := []byte("enum Colour { Red, Green = 4 }; main() { new Colour:value = Red; value = Green; new size = Colour; }\n")
	file := parser.Parse(src)
	model := semantic.Build(file, walk.New("x.pwn", file))
	wantValues := map[string]int64{"Red": 0, "Green": 4, "Colour": 5}
	for _, symbol := range model.Symbols {
		want, wanted := wantValues[symbol.Name]
		if !wanted {
			continue
		}
		if len(symbol.Tags) != 1 || symbol.Tags[0] != "Colour" {
			t.Fatalf("%s tags = %v", symbol.Name, symbol.Tags)
		}
		identifiers := model.Walk.OfKind(parser.KindIdentifier)
		found := false
		for _, identifier := range identifiers {
			if model.Resolve(identifier) != symbol {
				continue
			}
			value, ok := model.Eval(identifier)
			if !ok || value != want {
				t.Fatalf("%s value = %d, %t", symbol.Name, value, ok)
			}
			found = true
		}
		if !found {
			t.Fatalf("%s reference not resolved", symbol.Name)
		}
	}
}

func TestBuildSkipsNonValueIdentifiers(t *testing.T) {
	src := []byte("new Float; main() { new Float:value; value.member = 1; }\n")
	file := parser.Parse(src)
	model := semantic.Build(file, walk.New("x.pwn", file))
	var global *semantic.Symbol
	for _, symbol := range model.Symbols {
		if symbol.Name == "Float" {
			global = symbol
		}
	}
	if global == nil {
		t.Fatal("global symbol not collected")
	}
	if got := len(model.References(global)); got != 0 {
		t.Fatalf("non-value references = %d", got)
	}
}

func TestBuildResolvesGotoAsLabel(t *testing.T) {
	src := []byte("main() { new done; goto done; done: }\n")
	file := parser.Parse(src)
	model := semantic.Build(file, walk.New("x.pwn", file))
	for _, symbol := range model.Symbols {
		if symbol.Kind == semantic.SymbolLabel && len(model.References(symbol)) != 1 {
			t.Fatalf("label references = %d", len(model.References(symbol)))
		}
	}
}

func TestBuildResolvesForwardedFunctionCall(t *testing.T) {
	src := []byte("forward helper(); main() { helper(); } helper() {}\n")
	file := parser.Parse(src)
	model := semantic.Build(file, walk.New("x.pwn", file))
	var definition *semantic.Symbol
	for _, symbol := range model.Symbols {
		if symbol.Name == "helper" && symbol.Decl.Kind == parser.KindFunctionDefinition {
			definition = symbol
		}
	}
	if definition == nil || len(model.References(definition)) != 1 {
		t.Fatal("forwarded function call not resolved to definition")
	}
}
