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

type RedundantInitialization struct{}

func (RedundantInitialization) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "redundant-initialization",
		Name:            "Redundant initialization",
		Summary:         "Reports local initial values overwritten before any read",
		Explanation:     "A pure scalar initializer is redundant when every following path overwrites the local or exits before reading its initial value. Static locals, loop declarations, side effects, uncertain flow, and non-standalone writes are skipped.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"control-flow", "initialization", "assignments", "data-flow"},
	}
}

type initializerCandidate struct {
	initializer *parser.Node
	symbol      *semantic.Symbol
	block       *controlflow.Block
}

func (RedundantInitialization) Run(ctx *lint.Context) {
	if ctx.Flow == nil || ctx.Semantic == nil {
		return
	}
	for _, function := range ctx.Flow.Functions {
		if function.Uncertain {
			continue
		}
		writes := deadWriteCandidates(ctx, function)
		definitions := make(map[*controlflow.Block]*semantic.Symbol, len(writes))
		byBlock := make(map[*controlflow.Block]writeCandidate, len(writes))
		for _, write := range writes {
			definitions[write.block] = write.symbol
			byBlock[write.block] = write
		}
		uses := deadWriteUses(ctx, function, byBlock)
		for _, candidate := range redundantInitializerCandidates(ctx, function) {
			if !hasReachableDefinition(candidate.block, candidate.symbol, definitions) || valueReadAfter(candidate.block, candidate.symbol, definitions, uses) {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("initial value of %q is overwritten before it is read", candidate.symbol.Name),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(candidate.initializer),
			})
		}
	}
}

func redundantInitializerCandidates(ctx *lint.Context, function *controlflow.Function) []initializerCandidate {
	var result []initializerCandidate
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol == nil || symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal || symbol.Function != function.Node || symbol.Decl == nil {
			continue
		}
		initializer := symbol.Decl.Field("initializer")
		if initializer == nil || !ctx.Semantic.Pure(initializer) {
			continue
		}
		if len(symbol.Decl.Children) != 0 && hasDimension(symbol.Decl) {
			continue
		}
		declaration := ctx.Walk.Parent(symbol.Decl)
		if declaration == nil || hasStaticToken(declaration) {
			continue
		}
		block := function.Block(symbol.Decl)
		if block == nil || !function.ReachableBlock(block) || block.Node != nil && block.Node.Kind == parser.KindForStatement || initializerReadInBlock(ctx, function, symbol, initializer, block) {
			continue
		}
		result = append(result, initializerCandidate{initializer: initializer, symbol: symbol, block: block})
	}
	return result
}

func hasDimension(declaration *parser.Node) bool {
	for _, child := range declaration.Children {
		if child.Kind == parser.KindDimension {
			return true
		}
	}
	return false
}

func hasStaticToken(declaration *parser.Node) bool {
	for _, child := range declaration.Children {
		if child.Tok.Kind == token.KwStatic {
			return true
		}
	}
	return false
}

func initializerReadInBlock(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol, initializer *parser.Node, block *controlflow.Block) bool {
	for _, reference := range ctx.Semantic.References(symbol) {
		if function.Block(reference.Node) != block || reference.Kind == semantic.ReferenceWrite {
			continue
		}
		if reference.Node.Start >= initializer.Start {
			return true
		}
	}
	return false
}

func hasReachableDefinition(start *controlflow.Block, symbol *semantic.Symbol, definitions map[*controlflow.Block]*semantic.Symbol) bool {
	visited := make(map[*controlflow.Block]bool)
	var search func(*controlflow.Block) bool
	search = func(block *controlflow.Block) bool {
		if block == nil || visited[block] {
			return false
		}
		visited[block] = true
		if definitions[block] == symbol {
			return true
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
