package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
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
		Explanation:     "A local initialized from a known file or SQLite resource creator must be released on every path before the function exits. Tracking stops conservatively when ownership escapes to user code or another value.",
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

func resourceReachesExit(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol, acquisition *parser.Node, releaser string) bool {
	references := resourceReferencesByBlock(ctx, function, symbol)
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
			if visit(edge.To) {
				return true
			}
		}
		return false
	}
	return visit(start)
}
