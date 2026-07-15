package correctness

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type SignednessMismatch struct{}

func (SignednessMismatch) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "signedness-mismatch",
		Name:            "Signedness mismatch",
		Summary:         "Reports packed-character comparisons with negative values",
		Explanation:     "Packed-character selection produces values from 0 through 255. Comparing one with a definitely negative cell value usually indicates a sentinel or storage mistake. Unknown ranges, ordinary cell subscripts, macros, and uncertain expressions are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"signedness", "packed", "characters", "comparisons"},
	}
}

func (SignednessMismatch) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, comparison := range ctx.Walk.OfKind(parser.KindBinaryExpression) {
		if !signednessComparisonOperator(comparison.Tok.Kind) || signednessMismatchSkipped(ctx, comparison) {
			continue
		}
		left := comparison.Field("left")
		right := comparison.Field("right")
		var value *parser.Node
		if packedCharacterSelection(left) && !packedCharacterSelection(right) {
			value = right
		} else if packedCharacterSelection(right) && !packedCharacterSelection(left) {
			value = left
		} else {
			continue
		}
		valueRange, ok := expressionComparisonRange(ctx, value)
		if !ok || valueRange.maximum >= 0 {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "comparison mixes an unsigned packed character with a negative value",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(comparison),
		})
	}
}

func signednessComparisonOperator(kind token.Kind) bool {
	switch kind {
	case token.Eq, token.NotEq, token.Lt, token.Gt, token.LtEq, token.GtEq:
		return true
	default:
		return false
	}
}

func signednessMismatchSkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || node.HasError || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
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
