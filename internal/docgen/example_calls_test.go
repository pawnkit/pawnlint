package docgen

import (
	"strings"
	"testing"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func TestRuleExamplesAreSelfContained(t *testing.T) {
	metadata, err := api.Merge("openmp")
	if err != nil {
		t.Fatal(err)
	}
	compatibilityCalls := make(map[string]struct{})
	for name := range api.DeprecatedFunctions("openmp") {
		compatibilityCalls[name] = struct{}{}
	}
	for name := range api.UnsupportedFunctions("openmp") {
		compatibilityCalls[name] = struct{}{}
	}
	compatibilityCalls["mysql_format"] = struct{}{}
	for _, rule := range rules.Default().Sorted() {
		for _, name := range []string{"invalid", "valid"} {
			source, ok := readExample(rule.ID, name)
			if !ok {
				continue
			}
			t.Run(rule.ID+"/"+name, func(t *testing.T) {
				if strings.Contains(source, "// …") {
					t.Fatal("example is truncated")
				}
				parsed := parser.Parse([]byte(source))
				tree := walk.New(rule.ID+".pwn", parsed)
				model := semantic.Build(parsed, tree)
				declared := make(map[string]struct{})
				collectDeclaredCalls(tree, declared)
				if support, ok := readExampleSupport(rule.ID, name); ok {
					supportParsed := parser.Parse([]byte(support))
					collectDeclaredCalls(walk.New(rule.ID+".inc", supportParsed), declared)
				}
				defines := make(map[string]struct{})
				for _, name := range tree.KnownDefinesAt(len(parsed.Source) + 1) {
					defines[name] = struct{}{}
				}
				for _, call := range tree.OfKind(parser.KindCallExpression) {
					function := call.Field("function")
					if function == nil || function.Kind != parser.KindIdentifier || tree.Inactive(call) || tree.Uncertain(call) {
						continue
					}
					name := tree.Text(function)
					if _, ok := declared[name]; ok {
						continue
					}
					if _, ok := defines[name]; ok || name == "if" {
						continue
					}
					if model.ResolveAsCallTarget(function) != nil {
						continue
					}
					if _, ok := metadata.Natives[name]; ok {
						continue
					}
					if _, ok := metadata.Functions[name]; ok {
						continue
					}
					if _, ok := compatibilityCalls[name]; ok {
						continue
					}
					t.Errorf("unknown call %q at offset %d", name, call.Start)
				}
				for _, reference := range model.UnresolvedReferences() {
					if reference.Target == semantic.ReferenceFunction || tree.Inactive(reference.Node) || tree.Uncertain(reference.Node) || insideDefine(tree, reference.Node) {
						continue
					}
					name := tree.Text(reference.Node)
					if _, ok := defines[name]; ok {
						continue
					}
					if _, ok := metadata.Constants[name]; ok {
						continue
					}
					t.Errorf("unknown value %q at offset %d", name, reference.Node.Start)
				}
			})
		}
	}
}

func collectDeclaredCalls(tree *walk.Model, declared map[string]struct{}) {
	for _, kind := range []parser.Kind{parser.KindFunctionDefinition, parser.KindFunctionDeclaration} {
		for _, function := range tree.OfKind(kind) {
			if identifier := function.Field("name"); identifier != nil {
				declared[tree.Text(identifier)] = struct{}{}
			}
		}
	}
}

func insideDefine(tree *walk.Model, node *parser.Node) bool {
	for _, ancestor := range tree.Ancestors(node) {
		if ancestor.Kind == parser.KindDirectiveDefine {
			return true
		}
	}
	return false
}
