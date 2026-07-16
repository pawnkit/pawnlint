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

type StringConcatenationLoop struct{}

func (StringConcatenationLoop) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "string-concatenation-loop",
		Name:            "String concatenation loop",
		Summary:         "Reports strcat calls that repeatedly scan a growing buffer",
		Explanation:     "Appending to the same string with strcat on every loop iteration repeatedly scans the growing destination. The rule checks unconditional calls on one-dimensional local buffers that survive the loop and ignores reset, accessed, conditional, macro-derived, uncertain, self-appending, and shadowed cases.",
		Category:        diagnostic.CategoryPerformance,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"strings", "loops", "strcat", "performance"},
	}
}

func (StringConcatenationLoop) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	seen := make(map[*parser.Node]map[*semantic.Symbol]struct{})
	for _, call := range ctx.Walk.OfKind(parser.KindCallExpression) {
		if !stringConcatenationNative(ctx, call) || stringConcatenationSkipped(ctx, call) {
			continue
		}
		statement := ctx.Walk.Parent(call)
		loop := loopInvariantNearestLoop(ctx, call)
		if statement == nil || statement.Kind != parser.KindExpressionStatement || statement.Field("expression") != call || loop == nil || loopInvariantUncertain(ctx, loop) {
			continue
		}
		body := loop.Field("body")
		if body == nil || body.Kind != parser.KindBlock || stringConcatenationBlock(ctx, call) != body || stringConcatenationControlBefore(ctx, body, statement) {
			continue
		}
		arguments := call.Field("arguments")
		if arguments == nil || len(arguments.Children) < 2 || stringConcatenationHasError(arguments) {
			continue
		}
		destination := unwrap(arguments.Children[0])
		if destination == nil || destination.Kind != parser.KindIdentifier {
			continue
		}
		symbol := ctx.Semantic.Resolve(destination)
		if symbol == nil || symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal || overwrittenCopyDimensions(symbol.Decl) != 1 || symbol.Decl.Start >= loop.Start || inside(symbol.Decl, loop) {
			continue
		}
		if _, reported := seen[loop][symbol]; reported {
			continue
		}
		if stringConcatenationBufferAccessed(ctx, loop, symbol) || !stringConcatenationUsedAfter(ctx, loop, symbol) {
			continue
		}
		if seen[loop] == nil {
			seen[loop] = make(map[*semantic.Symbol]struct{})
		}
		seen[loop][symbol] = struct{}{}
		ctx.Report(diagnostic.Diagnostic{
			Message:     fmt.Sprintf("strcat repeatedly scans the growing string %q inside the loop", symbol.Name),
			Filename:    ctx.File.Path,
			Range:       ctx.Walk.Range(call),
			Suggestions: []diagnostic.Suggestion{{Description: "append with a tracked write position"}},
		})
	}
}

func stringConcatenationNative(ctx *lint.Context, call *parser.Node) bool {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier || callee.Tok.Origin != nil || ctx.Walk.Text(callee) != "strcat" {
		return false
	}
	native, known := ctx.Natives()["strcat"]
	if !known || native.Name != "strcat" || len(native.Buffers) == 0 || native.Buffers[0].Parameter != 1 {
		return false
	}
	if ctx.Project != nil && ctx.ProjectFile != nil {
		if declaration, ok := ctx.Project.Resolve(ctx.ProjectFile, callee); ok {
			return declaration.Kind == semantic.SymbolFunction && declaration.Valid() && declaration.HasToken(token.KwNative)
		}
		if len(ctx.Project.Declarations["strcat"]) != 0 {
			return false
		}
	}
	if symbol := ctx.Semantic.ResolveAsCallTarget(callee); symbol != nil {
		return symbol.Kind == semantic.SymbolFunction && symbol.Decl != nil && !symbol.Ambiguous && walk.HasChildToken(symbol.Decl, token.KwNative)
	}
	return true
}

func stringConcatenationBufferAccessed(ctx *lint.Context, loop *parser.Node, symbol *semantic.Symbol) bool {
	for _, reference := range ctx.Semantic.References(symbol) {
		if !inside(reference.Node, loop) {
			continue
		}
		if stringConcatenationCallReference(ctx, loop, reference.Node) {
			continue
		}
		return true
	}
	return false
}

func stringConcatenationCallReference(ctx *lint.Context, loop, reference *parser.Node) bool {
	for current := reference; current != nil && current != loop; current = ctx.Walk.Parent(current) {
		if current.Kind != parser.KindCallExpression {
			continue
		}
		if !stringConcatenationNative(ctx, current) || loopInvariantNearestLoop(ctx, current) != loop || stringConcatenationBlock(ctx, current) != loop.Field("body") {
			return false
		}
		statement := ctx.Walk.Parent(current)
		arguments := current.Field("arguments")
		if statement == nil || statement.Kind != parser.KindExpressionStatement || statement.Field("expression") != current || arguments == nil || len(arguments.Children) < 2 {
			return false
		}
		if unwrap(arguments.Children[0]) == reference {
			return true
		}
		if len(arguments.Children) > 2 && inside(reference, arguments.Children[2]) {
			for node := reference; node != nil && node != current; node = ctx.Walk.Parent(node) {
				if node.Kind == parser.KindSizeofExpression {
					return true
				}
			}
		}
		return false
	}
	return false
}

func stringConcatenationUsedAfter(ctx *lint.Context, loop *parser.Node, symbol *semantic.Symbol) bool {
	function := ctx.Walk.EnclosingFunction(loop)
	if function == nil {
		return false
	}
	for _, reference := range ctx.Semantic.References(symbol) {
		if reference.Node.Start > loop.End && inside(reference.Node, function) && reference.Kind != semantic.ReferenceWrite {
			return true
		}
	}
	return false
}

func stringConcatenationControlBefore(ctx *lint.Context, block, statement *parser.Node) bool {
	for _, kind := range []parser.Kind{
		parser.KindIfStatement, parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement,
		parser.KindSwitchStatement, parser.KindReturnStatement, parser.KindGotoStatement, parser.KindBreakStatement,
		parser.KindContinueStatement, parser.KindStateStatement, parser.KindLabelStatement,
	} {
		for _, node := range ctx.Walk.OfKind(kind) {
			if node.Start < statement.Start && stringConcatenationBlock(ctx, node) == block {
				return true
			}
		}
	}
	return false
}

func stringConcatenationSkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || stringConcatenationHasError(node) || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return true
	}
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindDirectiveDefine, parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			return true
		}
	}
	return false
}

func stringConcatenationHasError(node *parser.Node) bool {
	if node == nil || node.HasError {
		return true
	}
	for _, child := range node.Children {
		if stringConcatenationHasError(child) {
			return true
		}
	}
	return false
}

func stringConcatenationBlock(ctx *lint.Context, node *parser.Node) *parser.Node {
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindBlock {
			return current
		}
	}
	return nil
}
