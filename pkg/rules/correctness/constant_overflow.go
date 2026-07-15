package correctness

import (
	"strconv"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ConstantOverflow struct{}

func (ConstantOverflow) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "constant-overflow",
		Name:            "Constant overflow",
		Summary:         "Reports constant arithmetic outside the cell range",
		Explanation:     "Pawn cells are 32-bit values. Definite integer addition, subtraction, multiplication, division, negation, and literals that lose bits are checked before cell wrapping. Floats, bitwise operations, runtime values, macros, and uncertain expressions are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"constants", "arithmetic", "overflow", "cells"},
	}
}

const (
	minimumCellValue = int64(-2147483648)
	maximumCellValue = int64(2147483647)
)

func (ConstantOverflow) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, literal := range ctx.Walk.OfKind(parser.KindLiteral) {
		if literal.Tok.Kind != token.IntLiteral || constantOverflowSkipped(ctx, literal) {
			continue
		}
		if _, ok := constantLiteralCellValue(ctx.Walk.Text(literal)); ok || constantNegatedMinimum(ctx, literal) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "integer literal is outside the 32-bit cell range",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(literal),
		})
	}
	for _, node := range ctx.Walk.OfKind(parser.KindUnaryExpression) {
		if node.Tok.Kind != token.Minus || constantOverflowSkipped(ctx, node) {
			continue
		}
		if constantUnsignedMagnitude(ctx, node.Field("expression")) == uint64(2147483648) {
			continue
		}
		value, ok := constantOverflowValue(ctx, node.Field("expression"))
		if !ok || value != minimumCellValue {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "constant negation overflows the 32-bit cell range",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	}
	for _, node := range ctx.Walk.OfKind(parser.KindBinaryExpression) {
		if constantOverflowSkipped(ctx, node) {
			continue
		}
		operation := constantOverflowOperation(node.Tok.Kind)
		if operation == "" {
			continue
		}
		left, leftOK := constantOverflowValue(ctx, node.Field("left"))
		right, rightOK := constantOverflowValue(ctx, node.Field("right"))
		if !leftOK || !rightOK {
			continue
		}
		value, ok := constantArithmeticValue(node.Tok.Kind, left, right)
		if !ok || value >= minimumCellValue && value <= maximumCellValue {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "constant " + operation + " overflows the 32-bit cell range",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	}
}

func constantOverflowOperation(kind token.Kind) string {
	switch kind {
	case token.Plus:
		return "addition"
	case token.Minus:
		return "subtraction"
	case token.Star:
		return "multiplication"
	case token.Slash:
		return "division"
	default:
		return ""
	}
}

func constantArithmeticValue(kind token.Kind, left, right int64) (int64, bool) {
	switch kind {
	case token.Plus:
		return left + right, true
	case token.Minus:
		return left - right, true
	case token.Star:
		return left * right, true
	case token.Slash:
		if right == 0 {
			return 0, false
		}
		return left / right, true
	default:
		return 0, false
	}
}

func constantOverflowValue(ctx *lint.Context, node *parser.Node) (int64, bool) {
	if node == nil || node.HasError || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return 0, false
	}
	switch node.Kind {
	case parser.KindLiteral:
		if node.Tok.Kind != token.IntLiteral {
			return 0, false
		}
		return constantLiteralCellValue(ctx.Walk.Text(node))
	case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
		return constantOverflowValue(ctx, node.Field("expression"))
	case parser.KindUnaryExpression:
		if node.Tok.Kind == token.Minus {
			if constantUnsignedMagnitude(ctx, node.Field("expression")) == uint64(2147483648) {
				return minimumCellValue, true
			}
			value, ok := constantOverflowValue(ctx, node.Field("expression"))
			if !ok || value == minimumCellValue {
				return 0, false
			}
			return -value, true
		}
		if node.Tok.Kind == token.Plus {
			return constantOverflowValue(ctx, node.Field("expression"))
		}
	}
	return ctx.Constant(node)
}

func constantLiteralCellValue(text string) (int64, bool) {
	text = strings.ReplaceAll(text, "_", "")
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") || strings.HasPrefix(text, "0b") || strings.HasPrefix(text, "0B") {
		base = 0
	}
	value, err := strconv.ParseUint(text, base, 32)
	if err != nil {
		return 0, false
	}
	return int64(int32(uint32(value))), true
}

func constantUnsignedMagnitude(ctx *lint.Context, node *parser.Node) uint64 {
	for node != nil && (node.Kind == parser.KindParenthesizedExpression || node.Kind == parser.KindTaggedExpression) {
		node = node.Field("expression")
	}
	if node == nil || node.Kind != parser.KindLiteral || node.Tok.Kind != token.IntLiteral {
		return 0
	}
	text := strings.ReplaceAll(ctx.Walk.Text(node), "_", "")
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") || strings.HasPrefix(text, "0b") || strings.HasPrefix(text, "0B") {
		return 0
	}
	value, _ := strconv.ParseUint(text, 10, 64)
	return value
}

func constantNegatedMinimum(ctx *lint.Context, literal *parser.Node) bool {
	if constantUnsignedMagnitude(ctx, literal) != uint64(2147483648) {
		return false
	}
	current := literal
	for parent := ctx.Walk.Parent(current); parent != nil; parent = ctx.Walk.Parent(parent) {
		if parent.Kind == parser.KindParenthesizedExpression || parent.Kind == parser.KindTaggedExpression {
			current = parent
			continue
		}
		return parent.Kind == parser.KindUnaryExpression && parent.Tok.Kind == token.Minus && parent.Field("expression") == current
	}
	return false
}

func constantOverflowSkipped(ctx *lint.Context, node *parser.Node) bool {
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
