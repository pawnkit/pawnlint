package performance

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RepeatedStrlen struct{}

func (RepeatedStrlen) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "repeated-strlen-in-loop",
		Name:            "Repeated strlen in loop",
		Summary:         "Reports loop conditions that repeatedly scan an unchanged local string",
		Explanation:     "A loop condition is evaluated on every iteration. Calling strlen there repeatedly scans the same local string when the loop neither writes it nor passes it to another call.",
		Category:        diagnostic.CategoryPerformance,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"strings", "loops", "calls", "performance"},
	}
}

func (RepeatedStrlen) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, kind := range []parser.Kind{parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement} {
		ctx.Walk.IterKind(kind, func(loop *parser.Node) {
			condition := loop.Field("condition")
			if condition == nil || ctx.Walk.Uncertain(loop) {
				return
			}
			for _, call := range ctx.Walk.OfKind(parser.KindCallExpression) {
				if !inside(call, condition) || !isBuiltinStrlen(ctx, call) {
					continue
				}
				arguments := call.Field("arguments")
				if arguments == nil || len(arguments.Children) != 1 {
					continue
				}
				argument := unwrap(arguments.Children[0])
				if argument == nil || argument.Kind != parser.KindIdentifier {
					continue
				}
				symbol := ctx.Semantic.Resolve(argument)
				if symbol == nil || symbol.Kind != semantic.SymbolLocal || symbol.Ambiguous || symbol.Decl.Field("array") == nil || stringMayChange(ctx, loop, call, symbol) {
					continue
				}
				ctx.Report(diagnostic.Diagnostic{
					Message:     fmt.Sprintf("strlen(%s) rescans an unchanged local string on every iteration", symbol.Name),
					Filename:    ctx.File.Path,
					Range:       ctx.Walk.Range(call),
					Suggestions: []diagnostic.Suggestion{{Description: "compute the string length once before the loop"}},
				})
			}
		})
	}
}

func isBuiltinStrlen(ctx *lint.Context, call *parser.Node) bool {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier || ctx.Walk.Text(callee) != "strlen" {
		return false
	}
	if _, ok := ctx.Natives()["strlen"]; !ok {
		return false
	}
	if symbol := ctx.Semantic.Resolve(callee); symbol != nil && !walk.HasChildToken(symbol.Decl, token.KwNative) {
		return false
	}
	if ctx.Project != nil {
		for _, declaration := range ctx.Project.Declarations["strlen"] {
			if declaration.Node.Kind == parser.KindFunctionDefinition {
				return false
			}
		}
	}
	return true
}

func stringMayChange(ctx *lint.Context, loop, strlenCall *parser.Node, symbol *semantic.Symbol) bool {
	for _, reference := range ctx.Semantic.References(symbol) {
		if !inside(reference.Node, loop) {
			continue
		}
		if reference.Kind != semantic.ReferenceRead {
			return true
		}
		for node := reference.Node; node != nil && node != loop; node = ctx.Walk.Parent(node) {
			parent := ctx.Walk.Parent(node)
			if parent != nil && parent.Kind == parser.KindAssignmentExpression && inside(reference.Node, parent.Field("left")) {
				return true
			}
			if parent != nil && parent.Kind == parser.KindUpdateExpression {
				return true
			}
			if parent != nil && parent.Kind == parser.KindCallExpression && parent != strlenCall {
				return true
			}
		}
	}
	return false
}

func inside(node, container *parser.Node) bool {
	return node != nil && container != nil && node.Start >= container.Start && node.End <= container.End
}

func unwrap(node *parser.Node) *parser.Node {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	return node
}
