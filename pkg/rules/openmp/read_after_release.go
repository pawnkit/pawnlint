package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ReadAfterRelease struct{}

func (ReadAfterRelease) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "read-after-release",
		Name:            "Read after release",
		Summary:         "Reports local resource handles used after release",
		Explanation:     "A local handle is invalid after release or ownership transfer. The rule follows definite scalar aliases and simple project wrappers, then stops when ownership becomes ambiguous.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"resource", "handle", "lifetime", "control-flow", "api"},
	}
}

func (ReadAfterRelease) Run(ctx *lint.Context) {
	if ctx.Flow == nil || ctx.Semantic == nil {
		return
	}
	reported := make(map[*parser.Node]bool)
	for symbol, acquisitions := range resourceAcquisitions(ctx) {
		function := ctx.Flow.Function(symbol.Function)
		if function == nil || function.Uncertain {
			continue
		}
		for _, acquisition := range acquisitions {
			for _, release := range resourceReleases(ctx, function, symbol, acquisition) {
				for _, use := range resourceUsesAfterRelease(ctx, function, symbol, release) {
					if reported[use] {
						continue
					}
					reported[use] = true
					ctx.Report(diagnostic.Diagnostic{
						Message:  fmt.Sprintf("resource handle %q is used after release", symbol.Name),
						Filename: ctx.File.Path,
						Range:    ctx.Walk.Range(use),
					})
				}
			}
		}
	}
}

type resourceRelease struct {
	reference *parser.Node
	call      *parser.Node
}

func resourceReleases(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol, acquisition resourceAcquisition) []resourceRelease {
	references := resourceReferencesByBlock(ctx, function, symbol)
	visited := make(map[*controlflow.Block]bool)
	var releases []resourceRelease
	var visit func(*controlflow.Block)
	visit = func(block *controlflow.Block) {
		if block == nil || visited[block] || !function.ReachableBlock(block) {
			return
		}
		visited[block] = true
		for _, reference := range references[block] {
			if block == acquisition.block && reference.Node.Start <= acquisition.call.End {
				continue
			}
			if call := resourceReleaseCall(ctx, reference.Node, acquisition.release); call != nil {
				releases = append(releases, resourceRelease{reference: reference.Node, call: call})
				return
			}
			if resourceOwnershipEscapes(ctx, symbol, reference, acquisition.release) {
				return
			}
		}
		for _, edge := range block.Successors {
			visit(edge.To)
		}
	}
	visit(acquisition.block)
	return releases
}

func resourceReleaseCall(ctx *lint.Context, reference *parser.Node, releaser string) *parser.Node {
	for node := reference; node != nil; node = ctx.Walk.Parent(node) {
		parent := ctx.Walk.Parent(node)
		if parent == nil {
			return nil
		}
		if parent.Kind != parser.KindCallExpression {
			continue
		}
		arguments := parent.Field("arguments")
		if !nodeInField(reference, arguments) {
			return nil
		}
		if resourceCallTransfers(ctx, parent, reference, releaser, make(map[string]bool)) {
			return parent
		}
		return nil
	}
	return nil
}

func resourceUsesAfterRelease(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol, release resourceRelease) []*parser.Node {
	references := resourceReferencesByBlock(ctx, function, symbol)
	start := function.Block(release.reference)
	visited := make(map[*controlflow.Block]bool)
	seen := make(map[*parser.Node]bool)
	var uses []*parser.Node
	var visit func(*controlflow.Block)
	visit = func(block *controlflow.Block) {
		if block == nil || visited[block] || !function.ReachableBlock(block) {
			return
		}
		visited[block] = true
		for _, reference := range references[block] {
			if block == start && reference.Node.Start <= release.call.End {
				continue
			}
			if reference.Kind == semantic.ReferenceWrite {
				return
			}
			if !seen[reference.Node] {
				seen[reference.Node] = true
				uses = append(uses, reference.Node)
			}
		}
		for _, edge := range block.Successors {
			visit(edge.To)
		}
	}
	visit(start)
	return uses
}
