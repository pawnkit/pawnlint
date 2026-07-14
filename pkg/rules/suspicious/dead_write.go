package suspicious

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DeadWrite struct{}

func (DeadWrite) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "dead-write",
		Name:            "Dead write",
		Summary:         "Reports local assignments whose stored value is never read",
		Explanation:     "An assignment is dead when every following path overwrites the local variable or exits before reading it. Only direct, standalone assignments with unambiguous control flow are checked.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"control-flow", "assignments", "data-flow"},
	}
}

type writeCandidate struct {
	left   *parser.Node
	symbol *semantic.Symbol
	block  *controlflow.Block
}

func (DeadWrite) Run(ctx *lint.Context) {
	if ctx.Flow == nil || ctx.Semantic == nil {
		return
	}
	for _, function := range ctx.Flow.Functions {
		if function.Uncertain {
			continue
		}
		candidates := deadWriteCandidates(ctx, function)
		if len(candidates) == 0 {
			continue
		}
		definitions := make(map[*controlflow.Block]*semantic.Symbol, len(candidates))
		candidateByBlock := make(map[*controlflow.Block]writeCandidate, len(candidates))
		for _, candidate := range candidates {
			definitions[candidate.block] = candidate.symbol
			candidateByBlock[candidate.block] = candidate
		}
		uses := deadWriteUses(ctx, function, candidateByBlock)
		for _, candidate := range candidates {
			if valueReadAfter(candidate.block, candidate.symbol, definitions, uses) {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("value assigned to %q is never read", candidate.symbol.Name),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(candidate.left),
			})
		}
	}
}

func deadWriteCandidates(ctx *lint.Context, function *controlflow.Function) []writeCandidate {
	var result []writeCandidate
	for _, assignment := range ctx.Walk.OfKind(parser.KindAssignmentExpression) {
		if assignment.Tok.Kind != token.Assign || ctx.Walk.EnclosingFunction(assignment) != function.Node {
			continue
		}
		statement := ctx.Walk.Parent(assignment)
		if statement == nil || statement.Kind != parser.KindExpressionStatement || statement.Field("expression") != assignment {
			continue
		}
		left := assignment.Field("left")
		if left == nil || left.Kind != parser.KindIdentifier {
			continue
		}
		symbol := ctx.Semantic.Resolve(left)
		block := function.Block(assignment)
		if symbol == nil || symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal || !function.ReachableBlock(block) {
			continue
		}
		result = append(result, writeCandidate{left: left, symbol: symbol, block: block})
	}
	return result
}

func deadWriteUses(ctx *lint.Context, function *controlflow.Function, candidates map[*controlflow.Block]writeCandidate) map[*controlflow.Block]map[*semantic.Symbol]struct{} {
	uses := make(map[*controlflow.Block]map[*semantic.Symbol]struct{})
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Function != function.Node || symbol.Kind != semantic.SymbolLocal {
			continue
		}
		for _, reference := range ctx.Semantic.References(symbol) {
			block := function.Block(reference.Node)
			candidate, candidateBlock := candidates[block]
			if candidateBlock && candidate.symbol == symbol && candidate.left == reference.Node && reference.Kind == semantic.ReferenceWrite {
				continue
			}
			if uses[block] == nil {
				uses[block] = make(map[*semantic.Symbol]struct{})
			}
			uses[block][symbol] = struct{}{}
		}
	}
	return uses
}

func valueReadAfter(start *controlflow.Block, symbol *semantic.Symbol, definitions map[*controlflow.Block]*semantic.Symbol, uses map[*controlflow.Block]map[*semantic.Symbol]struct{}) bool {
	visited := make(map[*controlflow.Block]bool)
	var search func(*controlflow.Block) bool
	search = func(block *controlflow.Block) bool {
		if block == nil || visited[block] {
			return false
		}
		visited[block] = true
		if _, read := uses[block][symbol]; read {
			return true
		}
		if definitions[block] == symbol {
			return false
		}
		for _, edge := range block.Successors {
			if search(edge.To) {
				return true
			}
		}
		return false
	}
	for _, edge := range start.Successors {
		if search(edge.To) {
			return true
		}
	}
	return false
}
