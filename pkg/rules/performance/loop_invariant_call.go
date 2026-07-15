package performance

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type LoopInvariantCall struct{}

func (LoopInvariantCall) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "loop-invariant-call",
		Name:            "Loop-invariant call",
		Summary:         "Reports pure calls repeated with unchanged arguments in loops",
		Explanation:     "A pure call with unchanged arguments returns the same result on every iteration. The rule checks calls marked pure in API metadata and selected deterministic standard-library natives. Mutable arrays, globals, changed locals, unresolved calls, macros, uncertain loops, and strlen calls handled by the dedicated rule are ignored.",
		Category:        diagnostic.CategoryPerformance,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"loops", "calls", "performance", "purity"},
	}
}

func (LoopInvariantCall) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	calls := ctx.Walk.OfKind(parser.KindCallExpression)
	for _, kind := range []parser.Kind{parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement} {
		for _, loop := range ctx.Walk.OfKind(kind) {
			if loop.HasError || ctx.Walk.Inactive(loop) || ctx.Walk.Uncertain(loop) || loopInvariantUncertain(ctx, loop) {
				continue
			}
			for _, call := range calls {
				if loopInvariantNearestLoop(ctx, call) != loop || !loopInvariantRepeatedRegion(loop, call) || call.HasError || call.Tok.Origin != nil || ctx.Walk.Uncertain(call) {
					continue
				}
				name, pure := loopInvariantPureCall(ctx, call)
				if !pure || name == "strlen" || loopInvariantInsidePureCall(ctx, call, loop) {
					continue
				}
				symbols := make(map[*semantic.Symbol]struct{})
				arguments := call.Field("arguments")
				if arguments == nil || !loopInvariantArguments(ctx, arguments, loop, symbols) {
					continue
				}
				changed := false
				for symbol := range symbols {
					if loopInvariantSymbolChanges(ctx, loop, symbol) {
						changed = true
						break
					}
				}
				if changed {
					continue
				}
				ctx.Report(diagnostic.Diagnostic{
					Message:     fmt.Sprintf("pure call %q uses unchanged arguments inside the loop", name),
					Filename:    ctx.File.Path,
					Range:       ctx.Walk.Range(call),
					Suggestions: []diagnostic.Suggestion{{Description: "compute the result once before the loop"}},
				})
			}
		}
	}
}

func loopInvariantArguments(ctx *lint.Context, arguments, loop *parser.Node, symbols map[*semantic.Symbol]struct{}) bool {
	if arguments == nil {
		return false
	}
	for _, argument := range arguments.Children {
		if !loopInvariantExpression(ctx, argument, loop, symbols) {
			return false
		}
	}
	return true
}

func loopInvariantExpression(ctx *lint.Context, node, loop *parser.Node, symbols map[*semantic.Symbol]struct{}) bool {
	if node == nil || node.HasError || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return false
	}
	if _, known := ctx.Semantic.Eval(node); known {
		return true
	}
	switch node.Kind {
	case parser.KindLiteral:
		return true
	case parser.KindSizeofExpression, parser.KindTagofExpression:
		return true
	case parser.KindIdentifier:
		symbol := ctx.Semantic.Resolve(node)
		if symbol == nil || symbol.Ambiguous {
			_, boolean := ctx.Semantic.BooleanLiteral(node)
			return boolean
		}
		if symbol.Constant {
			return true
		}
		if symbol.Kind != semantic.SymbolLocal && symbol.Kind != semantic.SymbolParameter || invariantArraySymbol(symbol) || inside(symbol.Decl, loop) {
			return false
		}
		symbols[symbol] = struct{}{}
		return true
	case parser.KindParenthesizedExpression:
		return loopInvariantExpression(ctx, node.Field("expression"), loop, symbols)
	case parser.KindUnaryExpression, parser.KindUpdateExpression:
		if node.Kind == parser.KindUpdateExpression || node.Tok.Kind == token.PlusPlus || node.Tok.Kind == token.MinusMinus {
			return false
		}
		return loopInvariantExpression(ctx, node.Field("expression"), loop, symbols)
	case parser.KindBinaryExpression:
		return loopInvariantExpression(ctx, node.Field("left"), loop, symbols) && loopInvariantExpression(ctx, node.Field("right"), loop, symbols)
	case parser.KindTernaryExpression:
		return loopInvariantExpression(ctx, node.Field("condition"), loop, symbols) && loopInvariantExpression(ctx, node.Field("consequence"), loop, symbols) && loopInvariantExpression(ctx, node.Field("alternative"), loop, symbols)
	case parser.KindTaggedExpression:
		return loopInvariantExpression(ctx, node.Field("expression"), loop, symbols)
	case parser.KindCallExpression:
		_, pure := loopInvariantPureCall(ctx, node)
		return pure && loopInvariantArguments(ctx, node.Field("arguments"), loop, symbols)
	default:
		return false
	}
}

func loopInvariantPureCall(ctx *lint.Context, call *parser.Node) (string, bool) {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier || callee.Tok.Origin != nil {
		return "", false
	}
	name := ctx.Walk.Text(callee)
	native, nativeKnown := ctx.Natives()[name]
	function, functionKnown := ctx.Functions()[name]
	if !nativeKnown || !native.Pure {
		if !functionKnown || !function.Pure {
			return "", false
		}
	}
	if ctx.Project != nil && ctx.ProjectFile != nil {
		if declaration, ok := ctx.Project.Resolve(ctx.ProjectFile, callee); ok {
			if declaration.Kind != semantic.SymbolFunction || declaration.Node == nil {
				return "", false
			}
			if walk.HasChildToken(declaration.Node, token.KwNative) {
				return name, nativeKnown && native.Pure
			}
			return name, functionKnown && function.Pure
		}
		if len(ctx.Project.Declarations[name]) != 0 {
			return "", false
		}
	}
	if symbol := ctx.Semantic.ResolveAsCallTarget(callee); symbol != nil {
		if symbol.Kind != semantic.SymbolFunction || symbol.Decl == nil || symbol.Ambiguous {
			return "", false
		}
		if walk.HasChildToken(symbol.Decl, token.KwNative) {
			return name, nativeKnown && native.Pure
		}
		return name, functionKnown && function.Pure
	}
	return name, nativeKnown && native.Pure || functionKnown && function.Pure
}

func loopInvariantSymbolChanges(ctx *lint.Context, loop *parser.Node, symbol *semantic.Symbol) bool {
	for _, reference := range ctx.Semantic.References(symbol) {
		if !loopInvariantMutationRegion(loop, reference.Node) {
			continue
		}
		if reference.Kind != semantic.ReferenceRead {
			return true
		}
		for current := reference.Node; current != nil && current != loop; current = ctx.Walk.Parent(current) {
			if current.Kind == parser.KindCallExpression {
				if _, pure := loopInvariantPureCall(ctx, current); !pure {
					return true
				}
			}
		}
	}
	return false
}

func loopInvariantUncertain(ctx *lint.Context, loop *parser.Node) bool {
	for _, kind := range []parser.Kind{parser.KindMacroInvocation, parser.KindMacroInvocationBlock, parser.KindConditionalSplice} {
		for _, node := range ctx.Walk.OfKind(kind) {
			if inside(node, loop) {
				return true
			}
		}
	}
	return false
}

func loopInvariantNearestLoop(ctx *lint.Context, node *parser.Node) *parser.Node {
	for current := ctx.Walk.Parent(node); current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement:
			return current
		}
	}
	return nil
}

func loopInvariantRepeatedRegion(loop, node *parser.Node) bool {
	return inside(node, loop.Field("condition")) || inside(node, loop.Field("body")) || loop.Kind == parser.KindForStatement && inside(node, loop.Field("increment"))
}

func loopInvariantMutationRegion(loop, node *parser.Node) bool {
	return inside(node, loop.Field("condition")) || inside(node, loop.Field("body")) || loop.Kind == parser.KindForStatement && inside(node, loop.Field("increment"))
}

func loopInvariantInsidePureCall(ctx *lint.Context, call, loop *parser.Node) bool {
	for current := ctx.Walk.Parent(call); current != nil && current != loop; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindCallExpression {
			_, pure := loopInvariantPureCall(ctx, current)
			return pure
		}
	}
	return false
}

func invariantArraySymbol(symbol *semantic.Symbol) bool {
	if symbol == nil || symbol.Decl == nil {
		return true
	}
	for _, child := range symbol.Decl.Children {
		if child.Kind == parser.KindDimension {
			return true
		}
	}
	return false
}
