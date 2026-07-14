package openmp

import (
	"fmt"
	"sort"

	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type OverwrittenResourceHandle struct{}

func (OverwrittenResourceHandle) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "overwritten-resource-handle",
		Name:            "Overwritten resource handle",
		Summary:         "Reports resource handles overwritten before any use or release",
		Explanation:     "Replacing a local file or SQLite handle loses the previous resource. The rule reports only two direct acquisitions connected by one linear control-flow path with no intervening reference.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"resource", "handle", "database", "file", "control-flow"},
	}
}

func (OverwrittenResourceHandle) Run(ctx *lint.Context) {
	if ctx.Flow == nil || ctx.Semantic == nil {
		return
	}
	bySymbol := resourceAcquisitions(ctx)
	for symbol, acquisitions := range bySymbol {
		sort.Slice(acquisitions, func(i, j int) bool { return acquisitions[i].call.Start < acquisitions[j].call.Start })
		for i := 1; i < len(acquisitions); i++ {
			current := acquisitions[i]
			for j := i - 1; j >= 0; j-- {
				previous := acquisitions[j]
				function := ctx.Flow.Function(symbol.Function)
				if function == nil || !resourceReachesAcquisition(ctx, function, symbol, previous, current) {
					continue
				}
				ctx.Report(diagnostic.Diagnostic{
					Message:  fmt.Sprintf("assigning resource from %q overwrites unreleased handle %q", current.name, symbol.Name),
					Filename: ctx.File.Path,
					Range:    ctx.Walk.Range(current.write),
				})
				break
			}
		}
	}
}

func resourceReachesAcquisition(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol, previous, current resourceAcquisition) bool {
	if previous.block == nil || current.block == nil || previous.block == current.block {
		return false
	}
	references := resourceReferencesByBlock(ctx, function, symbol)
	visited := make(map[*controlflow.Block]bool)
	var visit func(*controlflow.Block) bool
	visit = func(block *controlflow.Block) bool {
		if block == nil || visited[block] || !function.ReachableBlock(block) {
			return false
		}
		if block == current.block {
			return true
		}
		visited[block] = true
		for _, reference := range references[block] {
			if block == previous.block && reference.Node.Start <= previous.call.End {
				continue
			}
			if resourceOwnershipEscapes(ctx, symbol, reference, previous.release) {
				return false
			}
		}
		for _, edge := range block.Successors {
			if visit(edge.To) {
				return true
			}
		}
		return false
	}
	return visit(previous.block)
}
