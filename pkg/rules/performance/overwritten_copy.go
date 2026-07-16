package performance

import (
	"fmt"
	"math"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type OverwrittenCopy struct{}

type overwrittenCopyCandidate struct {
	call   *parser.Node
	block  *parser.Node
	symbol *semantic.Symbol
	start  int64
	end    int64
}

func (OverwrittenCopy) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "overwritten-copy",
		Name:            "Overwritten copy",
		Summary:         "Reports memory copies overwritten before any access",
		Explanation:     "A memcpy has no useful effect when the next access to its local destination is an independent memcpy that covers the entire written byte range. The rule requires direct statements, one-dimensional local buffers, constant ranges, and the same lexical block. Partial, dynamic, branched, macro-derived, read, escaped, and self-copy cases are ignored.",
		Category:        diagnostic.CategoryPerformance,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"copies", "buffers", "memcpy", "performance"},
	}
}

func (OverwrittenCopy) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	var candidates []overwrittenCopyCandidate
	for _, call := range ctx.Walk.OfKind(parser.KindCallExpression) {
		candidate, ok := overwrittenCopyInfo(ctx, call)
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	for index, candidate := range candidates {
		for next := index + 1; next < len(candidates); next++ {
			overwrite := candidates[next]
			if overwrite.call.Start <= candidate.call.Start || overwrite.block != candidate.block || overwrite.symbol != candidate.symbol {
				continue
			}
			if overwrittenCopyAccessBetween(ctx, candidate, overwrite.call.Start) || overwrittenCopyControlBetween(ctx, candidate, overwrite.call.Start) {
				break
			}
			if overwrittenCopyReadsDestination(ctx, overwrite.call, candidate.symbol) || overwrite.start > candidate.start || overwrite.end < candidate.end {
				break
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:     fmt.Sprintf("copy into %q is completely overwritten before the buffer is accessed", candidate.symbol.Name),
				Filename:    ctx.File.Path,
				Range:       ctx.Walk.Range(candidate.call),
				Suggestions: []diagnostic.Suggestion{{Description: "remove the overwritten copy"}},
			})
			break
		}
	}
}

func overwrittenCopyControlBetween(ctx *lint.Context, candidate overwrittenCopyCandidate, before int) bool {
	for _, kind := range []parser.Kind{
		parser.KindIfStatement, parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement,
		parser.KindSwitchStatement, parser.KindReturnStatement, parser.KindGotoStatement, parser.KindBreakStatement,
		parser.KindContinueStatement, parser.KindStateStatement, parser.KindLabelStatement,
	} {
		for _, node := range ctx.Walk.OfKind(kind) {
			if node.Start > candidate.call.End && node.Start < before && overwrittenCopyBlock(ctx, node) == candidate.block {
				return true
			}
		}
	}
	return false
}

func overwrittenCopyInfo(ctx *lint.Context, call *parser.Node) (overwrittenCopyCandidate, bool) {
	if !overwrittenCopyNative(ctx, call) || overwrittenCopySkipped(ctx, call) {
		return overwrittenCopyCandidate{}, false
	}
	statement := ctx.Walk.Parent(call)
	if statement == nil || statement.Kind != parser.KindExpressionStatement || statement.Field("expression") != call {
		return overwrittenCopyCandidate{}, false
	}
	arguments := call.Field("arguments")
	if arguments == nil || len(arguments.Children) < 4 || overwrittenCopyHasError(arguments) {
		return overwrittenCopyCandidate{}, false
	}
	symbol, base, ok := overwrittenCopyDestination(ctx, arguments.Children[0])
	if !ok {
		return overwrittenCopyCandidate{}, false
	}
	index, indexOK := ctx.Eval(arguments.Children[2])
	length, lengthOK := ctx.Eval(arguments.Children[3])
	if !indexOK || !lengthOK || index < 0 || length <= 0 || base > math.MaxInt64-index || base+index > math.MaxInt64-length {
		return overwrittenCopyCandidate{}, false
	}
	start := base + index
	return overwrittenCopyCandidate{call: call, block: overwrittenCopyBlock(ctx, call), symbol: symbol, start: start, end: start + length}, true
}

func overwrittenCopyDestination(ctx *lint.Context, node *parser.Node) (*semantic.Symbol, int64, bool) {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	base := int64(0)
	identifier := node
	if node != nil && node.Kind == parser.KindSubscriptExpression {
		if node.Tok.Kind == token.LBrace || node.Field("array") == nil || node.Field("array").Kind != parser.KindIdentifier {
			return nil, 0, false
		}
		index, ok := ctx.Eval(node.Field("index"))
		if !ok || index < 0 || index > math.MaxInt64/4 {
			return nil, 0, false
		}
		base = index * 4
		identifier = node.Field("array")
	}
	if identifier == nil || identifier.Kind != parser.KindIdentifier {
		return nil, 0, false
	}
	symbol := ctx.Semantic.Resolve(identifier)
	if symbol == nil || symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal || overwrittenCopyDimensions(symbol.Decl) != 1 {
		return nil, 0, false
	}
	return symbol, base, true
}

func overwrittenCopyAccessBetween(ctx *lint.Context, candidate overwrittenCopyCandidate, before int) bool {
	for _, reference := range ctx.Semantic.References(candidate.symbol) {
		if reference.Node.Start > candidate.call.End && reference.Node.Start < before {
			return true
		}
	}
	return false
}

func overwrittenCopyReadsDestination(ctx *lint.Context, call *parser.Node, destination *semantic.Symbol) bool {
	arguments := call.Field("arguments")
	if arguments == nil || len(arguments.Children) < 2 {
		return true
	}
	source, _, ok := overwrittenCopyDestination(ctx, arguments.Children[1])
	return ok && source == destination
}

func overwrittenCopyNative(ctx *lint.Context, call *parser.Node) bool {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier || ctx.Walk.Text(callee) != "memcpy" {
		return false
	}
	if native, known := ctx.Natives()["memcpy"]; !known || native.Name != "memcpy" {
		return false
	}
	if ctx.Project != nil && ctx.ProjectFile != nil {
		if declaration, ok := ctx.Project.Resolve(ctx.ProjectFile, callee); ok {
			return declaration.Kind == semantic.SymbolFunction && declaration.Valid() && declaration.HasToken(token.KwNative)
		}
		if len(ctx.Project.Declarations["memcpy"]) != 0 {
			return false
		}
	}
	if symbol := ctx.Semantic.ResolveAsCallTarget(callee); symbol != nil {
		return symbol.Kind == semantic.SymbolFunction && symbol.Decl != nil && !symbol.Ambiguous && walk.HasChildToken(symbol.Decl, token.KwNative)
	}
	return true
}

func overwrittenCopySkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || overwrittenCopyHasError(node) || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
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

func overwrittenCopyHasError(node *parser.Node) bool {
	if node == nil || node.HasError {
		return true
	}
	for _, child := range node.Children {
		if overwrittenCopyHasError(child) {
			return true
		}
	}
	return false
}

func overwrittenCopyBlock(ctx *lint.Context, node *parser.Node) *parser.Node {
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindBlock {
			return current
		}
	}
	return nil
}

func overwrittenCopyDimensions(node *parser.Node) int {
	count := 0
	if node == nil {
		return count
	}
	for _, child := range node.Children {
		if child.Kind == parser.KindDimension {
			count++
		}
	}
	return count
}
