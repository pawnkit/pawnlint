package correctness

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type InfiniteLoop struct{}

func (InfiniteLoop) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "infinite-loop",
		Name:            "Infinite loop",
		Summary:         "Reports loops proven unable to exit",
		Explanation:     "A loop is infinite when its condition is definitely true, its condition values are unchanged, and no reachable break or return exits it. Gotos, macros, uncertain branches, calls in conditions, and unknown values suppress the diagnostic.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"loops", "control-flow", "conditions", "termination"},
	}
}

func (InfiniteLoop) Run(ctx *lint.Context) {
	if ctx.Flow == nil || ctx.Semantic == nil {
		return
	}
	for _, kind := range []parser.Kind{parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement} {
		for _, loop := range ctx.Walk.OfKind(kind) {
			functionNode := ctx.Walk.EnclosingFunction(loop)
			function := ctx.Flow.Function(functionNode)
			if function == nil || function.Uncertain || !function.Reachable(loop) || loop.HasError || ctx.Walk.Inactive(loop) || ctx.Walk.Uncertain(loop) {
				continue
			}
			condition := loop.Field("condition")
			if !infiniteLoopCondition(ctx, loop, condition) || !infiniteLoopHasNoExit(ctx, function, loop) {
				continue
			}
			rangeValue := ctx.Walk.Range(condition)
			if condition == nil {
				rangeValue = ctx.File.LineTable.Range(loop.Tok.Start.Offset, loop.Tok.End.Offset)
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  "loop condition is always true and the loop has no exit",
				Filename: ctx.File.Path,
				Range:    rangeValue,
			})
		}
	}
}

func infiniteLoopCondition(ctx *lint.Context, loop, condition *parser.Node) bool {
	if condition == nil {
		return loop.Kind == parser.KindForStatement
	}
	if constant, ok := ctx.Constant(condition); ok {
		return constant != 0
	}
	symbols, ok := invariantConditionSymbols(ctx, condition)
	if !ok || len(symbols) == 0 || invariantLoopHasUncertainMutation(ctx, loop) {
		return false
	}
	for _, symbol := range symbols {
		if invariantLoopChangesSymbol(ctx, loop, symbol) {
			return false
		}
	}
	value, known := ctx.Eval(condition)
	if !known {
		values, valuesKnown := infiniteInitialValues(ctx, loop, symbols)
		if !valuesKnown {
			return false
		}
		value, known = ctx.Semantic.EvalWithValues(condition, values)
	}
	return known && value != 0
}

func infiniteInitialValues(ctx *lint.Context, loop *parser.Node, symbols []*semantic.Symbol) (map[*semantic.Symbol]int64, bool) {
	values := make(map[*semantic.Symbol]int64, len(symbols))
	for _, symbol := range symbols {
		initializer := symbol.Decl.Field("initializer")
		value, ok := ctx.Constant(initializer)
		if !ok || symbol.Decl.End > loop.Start {
			return nil, false
		}
		for _, reference := range ctx.Semantic.References(symbol) {
			if reference.Node.Start <= symbol.Decl.End || reference.Node.Start >= loop.Start {
				continue
			}
			if reference.Kind != semantic.ReferenceRead || infiniteReferenceInCall(ctx, reference.Node) {
				return nil, false
			}
		}
		values[symbol] = value
	}
	return values, true
}

func infiniteReferenceInCall(ctx *lint.Context, reference *parser.Node) bool {
	for current := reference; current != nil; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindCallExpression {
			return true
		}
		if current.Kind == parser.KindExpressionStatement || current.Kind == parser.KindVariableDeclaration {
			return false
		}
	}
	return false
}

func infiniteLoopHasNoExit(ctx *lint.Context, function *controlflow.Function, loop *parser.Node) bool {
	body := loop.Field("body")
	if body == nil {
		return false
	}
	noExit := true
	var visit func(*parser.Node)
	visit = func(node *parser.Node) {
		if node == nil || !noExit || ctx.Walk.Inactive(node) {
			return
		}
		if ctx.Walk.Uncertain(node) {
			noExit = false
			return
		}
		switch node.Kind {
		case parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindConditionalSplice, parser.KindSharedConditional, parser.KindConditionalFunction,
			parser.KindGotoStatement:
			noExit = false
			return
		case parser.KindReturnStatement:
			if function.Reachable(node) {
				noExit = false
			}
			return
		case parser.KindBreakStatement:
			if function.Reachable(node) && infiniteBreakTargets(ctx, node, loop) {
				noExit = false
			}
			return
		case parser.KindCallExpression:
			if infiniteMacroCall(ctx, node) {
				noExit = false
				return
			}
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(body)
	return noExit
}

func infiniteMacroCall(ctx *lint.Context, call *parser.Node) bool {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return false
	}
	name := ctx.Walk.Text(callee)
	for _, define := range ctx.Walk.KnownDefinesAt(call.Start) {
		if define == name {
			return true
		}
	}
	return false
}

func infiniteBreakTargets(ctx *lint.Context, statement, loop *parser.Node) bool {
	for current := ctx.Walk.Parent(statement); current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindSwitchStatement, parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement:
			return current == loop
		}
	}
	return false
}
