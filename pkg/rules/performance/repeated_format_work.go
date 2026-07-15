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

type RepeatedFormatWork struct{}

func (RepeatedFormatWork) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "repeated-format-work",
		Name:            "Repeated format work",
		Summary:         "Reports invariant formatting repeated before a buffer is used",
		Explanation:     "Formatting the same values into an untouched local buffer on every loop iteration repeats work when the buffer is only consumed after the loop. The rule checks direct format, Format, and strformat statements with invariant inputs and ignores mutable substitutions, conditional calls, macros, uncertain loops, shadowed natives, and buffers accessed in the loop.",
		Category:        diagnostic.CategoryPerformance,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"strings", "formatting", "loops", "performance"},
	}
}

func (RepeatedFormatWork) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, call := range ctx.Walk.OfKind(parser.KindCallExpression) {
		name, ok := repeatedFormatNative(ctx, call)
		if !ok || repeatedFormatSkipped(ctx, call) {
			continue
		}
		statement := ctx.Walk.Parent(call)
		loop := loopInvariantNearestLoop(ctx, call)
		if statement == nil || statement.Kind != parser.KindExpressionStatement || statement.Field("expression") != call || loop == nil || loopInvariantUncertain(ctx, loop) {
			continue
		}
		body := loop.Field("body")
		if body == nil || body.Kind != parser.KindBlock || repeatedFormatBlock(ctx, call) != body || repeatedFormatControlBefore(ctx, body, statement) {
			continue
		}
		arguments := call.Field("arguments")
		minimum := 3
		if name == "strformat" {
			minimum = 4
		}
		if arguments == nil || len(arguments.Children) < minimum || repeatedFormatHasError(arguments) {
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
		symbols := make(map[*semantic.Symbol]struct{})
		invariant := true
		for _, argument := range arguments.Children[1:] {
			if !loopInvariantExpression(ctx, argument, loop, symbols) {
				invariant = false
				break
			}
		}
		if !invariant {
			continue
		}
		for input := range symbols {
			if repeatedFormatInputChanges(ctx, call, loop, input) {
				invariant = false
				break
			}
		}
		if !invariant || repeatedFormatBufferAccessed(ctx, call, loop, destination, symbol) || !repeatedFormatUsedAfter(ctx, loop, symbol) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:     fmt.Sprintf("%s repeatedly formats unchanged values into %q before the buffer is used", name, symbol.Name),
			Filename:    ctx.File.Path,
			Range:       ctx.Walk.Range(call),
			Suggestions: []diagnostic.Suggestion{{Description: "format the buffer once at its point of use"}},
		})
	}
}

func repeatedFormatInputChanges(ctx *lint.Context, call, loop *parser.Node, symbol *semantic.Symbol) bool {
	for _, reference := range ctx.Semantic.References(symbol) {
		if !loopInvariantMutationRegion(loop, reference.Node) || inside(reference.Node, call) {
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

func repeatedFormatNative(ctx *lint.Context, call *parser.Node) (string, bool) {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier || callee.Tok.Origin != nil {
		return "", false
	}
	name := ctx.Walk.Text(callee)
	if name != "format" && name != "Format" && name != "strformat" {
		return "", false
	}
	native, known := ctx.Natives()[name]
	if !known || native.Name != name || native.FormatParameter == 0 || len(native.Buffers) == 0 || native.Buffers[0].Parameter != 1 {
		return "", false
	}
	if ctx.Project != nil && ctx.ProjectFile != nil {
		if declaration, ok := ctx.Project.Resolve(ctx.ProjectFile, callee); ok {
			return name, declaration.Kind == semantic.SymbolFunction && declaration.Node != nil && walk.HasChildToken(declaration.Node, token.KwNative)
		}
		if len(ctx.Project.Declarations[name]) != 0 {
			return "", false
		}
	}
	if symbol := ctx.Semantic.ResolveAsCallTarget(callee); symbol != nil {
		return name, symbol.Kind == semantic.SymbolFunction && symbol.Decl != nil && !symbol.Ambiguous && walk.HasChildToken(symbol.Decl, token.KwNative)
	}
	return name, true
}

func repeatedFormatBufferAccessed(ctx *lint.Context, call, loop, destination *parser.Node, symbol *semantic.Symbol) bool {
	for _, reference := range ctx.Semantic.References(symbol) {
		if !inside(reference.Node, loop) {
			continue
		}
		if reference.Node == destination {
			continue
		}
		if inside(reference.Node, call) && repeatedFormatSizeofReference(ctx, call, reference.Node) {
			continue
		}
		return true
	}
	return false
}

func repeatedFormatSizeofReference(ctx *lint.Context, call, reference *parser.Node) bool {
	for current := reference; current != nil && current != call; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindSizeofExpression {
			return true
		}
	}
	return false
}

func repeatedFormatUsedAfter(ctx *lint.Context, loop *parser.Node, symbol *semantic.Symbol) bool {
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

func repeatedFormatControlBefore(ctx *lint.Context, block, statement *parser.Node) bool {
	for _, kind := range []parser.Kind{
		parser.KindIfStatement, parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement,
		parser.KindSwitchStatement, parser.KindReturnStatement, parser.KindGotoStatement, parser.KindBreakStatement,
		parser.KindContinueStatement, parser.KindStateStatement, parser.KindLabelStatement,
	} {
		for _, node := range ctx.Walk.OfKind(kind) {
			if node.Start < statement.Start && repeatedFormatBlock(ctx, node) == block {
				return true
			}
		}
	}
	return false
}

func repeatedFormatSkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || repeatedFormatHasError(node) || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
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

func repeatedFormatHasError(node *parser.Node) bool {
	if node == nil || node.HasError {
		return true
	}
	for _, child := range node.Children {
		if repeatedFormatHasError(child) {
			return true
		}
	}
	return false
}

func repeatedFormatBlock(ctx *lint.Context, node *parser.Node) *parser.Node {
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindBlock {
			return current
		}
	}
	return nil
}
