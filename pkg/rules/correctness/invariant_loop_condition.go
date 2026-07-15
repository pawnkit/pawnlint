package correctness

import (
	"fmt"
	"sort"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type InvariantLoopCondition struct{}

func (InvariantLoopCondition) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "invariant-loop-condition",
		Name:            "Invariant loop condition",
		Summary:         "Reports loop conditions unchanged by their loop",
		Explanation:     "A condition based only on unchanged local scalars has the same result on every iteration. Conditions with calls, parameters, globals, arrays, macros, assignments, or uncertain references are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"loops", "conditions", "data-flow", "semantic"},
	}
}

func (InvariantLoopCondition) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, kind := range []parser.Kind{parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement} {
		for _, loop := range ctx.Walk.OfKind(kind) {
			condition := loop.Field("condition")
			if condition == nil || loop.HasError || ctx.Walk.Inactive(loop) || ctx.Walk.Uncertain(loop) {
				continue
			}
			if _, constant := ctx.Semantic.Eval(condition); constant {
				continue
			}
			symbols, ok := invariantConditionSymbols(ctx, condition)
			if !ok || len(symbols) == 0 || invariantLoopHasUncertainMutation(ctx, loop) {
				continue
			}
			changed := false
			for _, symbol := range symbols {
				if invariantLoopChangesSymbol(ctx, loop, symbol) {
					changed = true
					break
				}
			}
			if changed {
				continue
			}
			names := make([]string, 0, len(symbols))
			for _, symbol := range symbols {
				names = append(names, symbol.Name)
			}
			sort.Strings(names)
			ctx.Report(diagnostic.Diagnostic{
				Message:  invariantLoopMessage(names),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(condition),
			})
		}
	}
}

func invariantConditionSymbols(ctx *lint.Context, condition *parser.Node) ([]*semantic.Symbol, bool) {
	symbols := make(map[*semantic.Symbol]struct{})
	valid := true
	var visit func(*parser.Node)
	visit = func(node *parser.Node) {
		if node == nil || !valid {
			return
		}
		if node.HasError || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
			valid = false
			return
		}
		switch node.Kind {
		case parser.KindCallExpression, parser.KindSubscriptExpression, parser.KindAssignmentExpression, parser.KindUpdateExpression,
			parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			valid = false
			return
		case parser.KindUnaryExpression:
			if node.Tok.Kind == token.PlusPlus || node.Tok.Kind == token.MinusMinus {
				valid = false
				return
			}
		case parser.KindIdentifier:
			symbol := ctx.Semantic.Resolve(node)
			if symbol == nil {
				if _, boolean := ctx.Semantic.BooleanLiteral(node); !boolean {
					valid = false
				}
				return
			}
			if symbol.Constant {
				return
			}
			if symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal || invariantArraySymbol(symbol) {
				valid = false
				return
			}
			symbols[symbol] = struct{}{}
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(condition)
	if !valid {
		return nil, false
	}
	result := make([]*semantic.Symbol, 0, len(symbols))
	for symbol := range symbols {
		result = append(result, symbol)
	}
	return result, true
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

func invariantLoopHasUncertainMutation(ctx *lint.Context, loop *parser.Node) bool {
	for _, kind := range []parser.Kind{parser.KindMacroInvocation, parser.KindMacroInvocationBlock, parser.KindConditionalSplice} {
		for _, node := range ctx.Walk.OfKind(kind) {
			if invariantMutationNode(loop, node) {
				return true
			}
		}
	}
	return false
}

func invariantLoopChangesSymbol(ctx *lint.Context, loop *parser.Node, symbol *semantic.Symbol) bool {
	for _, reference := range ctx.Semantic.References(symbol) {
		if !invariantMutationNode(loop, reference.Node) {
			continue
		}
		if reference.Kind != semantic.ReferenceRead || invariantReferenceMayChange(ctx, loop, reference.Node) {
			return true
		}
	}
	return false
}

func invariantMutationNode(loop, node *parser.Node) bool {
	if loop == nil || node == nil {
		return false
	}
	if invariantInside(node, loop.Field("body")) {
		return true
	}
	return loop.Kind == parser.KindForStatement && invariantInside(node, loop.Field("increment"))
}

func invariantReferenceMayChange(ctx *lint.Context, loop, reference *parser.Node) bool {
	for current := reference; current != nil && current != loop; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindCallExpression || current.Kind == parser.KindUpdateExpression || current.Kind == parser.KindUnaryExpression && (current.Tok.Kind == token.PlusPlus || current.Tok.Kind == token.MinusMinus) {
			return true
		}
	}
	return false
}

func invariantInside(node, container *parser.Node) bool {
	return node != nil && container != nil && node.Start >= container.Start && node.End <= container.End
}

func invariantLoopMessage(names []string) string {
	if len(names) == 1 {
		return fmt.Sprintf("loop condition depends on %q, which the loop never changes", names[0])
	}
	quoted := make([]string, len(names))
	for index, name := range names {
		quoted[index] = fmt.Sprintf("%q", name)
	}
	return "loop condition depends only on " + strings.Join(quoted, ", ") + ", which the loop never changes"
}
