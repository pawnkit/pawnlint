package maintainability

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RedundantElse struct{}

func (RedundantElse) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "redundant-else",
		Name:            "Redundant else",
		Summary:         "Reports else branches after unconditional control transfer",
		Explanation:     "An else is redundant when the preceding branch always returns, breaks, continues, or jumps away. Removing the else keeps the alternative's scope and comments intact. Uncertain and malformed branches are ignored.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         true,
		Tags:            []string{"control-flow", "branches", "style"},
	}
}

func (RedundantElse) Run(ctx *lint.Context) {
	for _, node := range ctx.Walk.OfKind(parser.KindIfStatement) {
		consequence := node.Field("consequence")
		alternative := node.Field("alternative")
		if alternative == nil || node.HasError || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) || redundantElseInMacro(ctx, node) || !redundantElseTransfers(ctx, consequence) {
			continue
		}
		elseRange, ok := redundantElseRange(ctx, consequence, alternative)
		if !ok {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "else is redundant because the preceding branch cannot continue",
			Filename: ctx.File.Path,
			Range:    elseRange,
			Fix: &diagnostic.Fix{
				Description: "remove the redundant else",
				Edits:       []diagnostic.Edit{{Range: elseRange, NewText: ""}},
			},
		})
	}
}

func redundantElseInMacro(ctx *lint.Context, node *parser.Node) bool {
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindDirectiveDefine, parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock:
			return true
		case parser.KindFunctionDefinition:
			return false
		}
	}
	return false
}

func redundantElseTransfers(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || node.HasError || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return false
	}
	switch node.Kind {
	case parser.KindReturnStatement, parser.KindBreakStatement, parser.KindContinueStatement, parser.KindGotoStatement:
		return true
	case parser.KindBlock, parser.KindConditionalRegion, parser.KindConditionalBranch:
		return redundantElseSequenceTransfers(ctx, node.Children)
	case parser.KindIfStatement:
		alternative := node.Field("alternative")
		return alternative != nil && redundantElseTransfers(ctx, node.Field("consequence")) && redundantElseTransfers(ctx, alternative)
	default:
		return false
	}
}

func redundantElseSequenceTransfers(ctx *lint.Context, nodes []*parser.Node) bool {
	for _, node := range nodes {
		if node == nil || ctx.Walk.Inactive(node) {
			continue
		}
		if node.HasError || ctx.Walk.Uncertain(node) {
			return false
		}
		if redundantElseTransfers(ctx, node) {
			return true
		}
	}
	return false
}

func redundantElseRange(ctx *lint.Context, consequence, alternative *parser.Node) (source.Range, bool) {
	if consequence == nil || alternative == nil {
		return source.Range{}, false
	}
	for _, current := range ctx.File.Parsed.Tokens {
		if current.Kind != token.KwElse || current.Start.Offset < consequence.End || current.End.Offset > alternative.Start || current.Origin != nil {
			continue
		}
		return ctx.File.LineTable.Range(current.Start.Offset, current.End.Offset), true
	}
	return source.Range{}, false
}
