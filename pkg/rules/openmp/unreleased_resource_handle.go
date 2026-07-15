package openmp

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnreleasedResourceHandle struct{}

func (UnreleasedResourceHandle) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unreleased-resource-handle",
		Name:            "Unreleased resource handle",
		Summary:         "Reports local resource handles that can reach function exit without release",
		Explanation:     "A local initialized from a known resource creator must be released on every path. The rule follows definite scalar aliases and simple project wrappers, then stops when ownership becomes ambiguous.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"resource", "handle", "database", "file", "control-flow"},
	}
}

func (UnreleasedResourceHandle) Run(ctx *lint.Context) {
	if ctx.Flow == nil || ctx.Semantic == nil {
		return
	}
	for symbol, acquisitions := range resourceAcquisitions(ctx) {
		function := ctx.Flow.Function(symbol.Function)
		if function == nil || function.Uncertain {
			continue
		}
		for _, acquisition := range acquisitions {
			if !function.ReachableBlock(acquisition.block) || !resourceReachesExit(ctx, function, symbol, acquisition.call, acquisition.release) {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("resource handle %q returned by %q may reach function exit without being released", symbol.Name, acquisition.name),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(acquisition.call),
			})
		}
	}
}

type controlflowEdge struct {
	from, to *controlflow.Block
}

func resourceReachesExit(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol, acquisition *parser.Node, releaser string) bool {
	references := resourceReferencesByBlock(ctx, function, symbol)
	unacquiredEdges := resourceUnacquiredGuardEdges(ctx, function, symbol)
	start := function.Block(acquisition)
	visited := make(map[*controlflow.Block]bool)
	var visit func(*controlflow.Block) bool
	visit = func(block *controlflow.Block) bool {
		if block == nil || visited[block] || !function.ReachableBlock(block) {
			return false
		}
		visited[block] = true
		for _, reference := range references[block] {
			if block == start && reference.Node.Start <= acquisition.End {
				continue
			}
			if resourceOwnershipEscapes(ctx, symbol, reference, releaser) {
				return false
			}
		}
		if block == function.Exit {
			return true
		}
		for _, edge := range block.Successors {
			if unacquiredEdges[controlflowEdge{block, edge.To}] {
				continue
			}
			if visit(edge.To) {
				return true
			}
		}
		return false
	}
	return visit(start)
}

func resourceUnacquiredGuardEdges(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol) map[controlflowEdge]bool {
	result := make(map[controlflowEdge]bool)
	for _, block := range function.Blocks {
		ifNode := block.Node
		if ifNode == nil || ifNode.Kind != parser.KindIfStatement {
			continue
		}
		truthyWhenTrue, ok := conditionTestsSymbolTruthiness(ctx, ifNode.Field("condition"), symbol)
		if !ok {
			continue
		}
		consequenceEntry := function.Block(ifNode.Field("consequence"))
		var alternativeEntry *controlflow.Block
		if alternative := ifNode.Field("alternative"); alternative != nil {
			alternativeEntry = function.Block(alternative)
		}
		truthyEntry, falsyEntry := consequenceEntry, alternativeEntry
		if !truthyWhenTrue {
			truthyEntry, falsyEntry = alternativeEntry, consequenceEntry
		}
		if falsyEntry != nil {
			result[controlflowEdge{block, falsyEntry}] = true
			continue
		}
		for _, edge := range block.Successors {
			if edge.To != truthyEntry {
				result[controlflowEdge{block, edge.To}] = true
			}
		}
	}
	return result
}

func conditionTestsSymbolTruthiness(ctx *lint.Context, condition *parser.Node, symbol *semantic.Symbol) (truthyWhenTrue bool, ok bool) {
	condition = unwrapParentheses(condition)
	if condition == nil {
		return false, false
	}
	if condition.Kind == parser.KindIdentifier {
		if ctx.Semantic.Resolve(condition) == symbol {
			return true, true
		}
		return false, false
	}
	if condition.Kind == parser.KindUnaryExpression && condition.Tok.Kind == token.Bang {
		operand := unwrapParentheses(condition.Field("expression"))
		if operand != nil && operand.Kind == parser.KindIdentifier && ctx.Semantic.Resolve(operand) == symbol {
			return false, true
		}
	}
	return false, false
}
