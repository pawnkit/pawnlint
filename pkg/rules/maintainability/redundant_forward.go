package maintainability

import (
	"fmt"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RedundantForward struct{}

func (RedundantForward) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "redundant-forward",
		Name:            "Redundant forward",
		Summary:         "Reports forward declarations that are not needed before a definition",
		Explanation:     "A forward declaration is redundant when the same file defines a non-public function without an earlier call that needs the declaration. Includes, macro invocations, unresolved calls, state functions, and declarations with storage effects are ignored.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"functions", "forward", "declarations"},
	}
}

func (RedundantForward) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	forwards := redundantForwardDeclarations(ctx)
	definitions, symbols := redundantForwardDefinitions(ctx)
	for name, declarations := range forwards {
		definitionList := definitions[name]
		if len(declarations) != 1 || len(definitionList) != 1 {
			continue
		}
		declaration := declarations[0]
		definition := definitionList[0]
		if !redundantForwardStoragePreserved(ctx, declaration, definition) || redundantForwardNeeded(ctx, declaration, definition, symbols[name]) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("forward declaration of %q is redundant", name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(declaration.Field("name")),
		})
	}
}

func redundantForwardDeclarations(ctx *lint.Context) map[string][]*parser.Node {
	result := make(map[string][]*parser.Node)
	for _, node := range ctx.Walk.OfKind(parser.KindFunctionDeclaration) {
		name, ok := redundantForwardName(ctx, node)
		if !ok || !walk.HasChildToken(node, token.KwForward) || walk.HasChildToken(node, token.KwStatic) || walk.HasChildToken(node, token.KwStock) {
			continue
		}
		result[name] = append(result[name], node)
	}
	return result
}

func redundantForwardDefinitions(ctx *lint.Context) (map[string][]*parser.Node, map[string]*semantic.Symbol) {
	definitions := make(map[string][]*parser.Node)
	symbols := make(map[string]*semantic.Symbol)
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolFunction || symbol.Decl.Kind != parser.KindFunctionDefinition || symbol.Ambiguous {
			continue
		}
		name, ok := redundantForwardName(ctx, symbol.Decl)
		if !ok {
			continue
		}
		definitions[name] = append(definitions[name], symbol.Decl)
		symbols[name] = symbol
	}
	return definitions, symbols
}

func redundantForwardName(ctx *lint.Context, node *parser.Node) (string, bool) {
	if node == nil || node.HasError || node.Field("state") != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return "", false
	}
	name := node.Field("name")
	if name == nil || name.Kind != parser.KindIdentifier || name.HasError {
		return "", false
	}
	text := ctx.Walk.Text(name)
	return text, text != ""
}

func redundantForwardStoragePreserved(ctx *lint.Context, declaration, definition *parser.Node) bool {
	return !redundantForwardPublic(ctx, declaration) && !redundantForwardPublic(ctx, definition)
}

func redundantForwardPublic(ctx *lint.Context, node *parser.Node) bool {
	if walk.HasChildToken(node, token.KwPublic) {
		return true
	}
	name := node.Field("name")
	return name != nil && strings.HasPrefix(ctx.Walk.Text(name), "@")
}

func redundantForwardNeeded(ctx *lint.Context, declaration, definition *parser.Node, symbol *semantic.Symbol) bool {
	if definition.Start <= declaration.End {
		return false
	}
	for _, reference := range ctx.Semantic.References(symbol) {
		if reference.Kind == semantic.ReferenceCall && reference.Node.Start > declaration.End && reference.Node.Start < definition.Start {
			return true
		}
	}
	for _, reference := range ctx.Semantic.UnresolvedReferences() {
		if reference.Target == semantic.ReferenceFunction && reference.Node.Start > declaration.End && reference.Node.Start < definition.Start {
			return true
		}
	}
	for _, kind := range []parser.Kind{parser.KindDirectiveInclude, parser.KindDirectiveTryInclude, parser.KindMacroInvocation, parser.KindMacroInvocationBlock} {
		for _, node := range ctx.Walk.OfKind(kind) {
			if node.Start > declaration.End && node.Start < definition.Start && !ctx.Walk.Inactive(node) {
				return true
			}
		}
	}
	return false
}
