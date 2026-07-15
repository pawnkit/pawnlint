package correctness

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DivisionByZero struct{}

func (DivisionByZero) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "division-by-zero",
		Name:            "Division by zero",
		Summary:         "Reports division or remainder by a constant zero",
		Explanation:     "Division and remainder by zero are invalid. The rule reports only operands that can be evaluated with certainty.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"constants", "arithmetic", "control-flow"},
	}
}

func (DivisionByZero) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	check := func(right *parser.Node) {
		if value, ok := ctx.Eval(right); !ok || value != 0 {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "division or remainder by constant zero",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(right),
		})
	}
	ctx.Walk.IterKind(parser.KindBinaryExpression, func(node *parser.Node) {
		if node.Tok.Kind != token.Slash && node.Tok.Kind != token.Percent {
			return
		}
		check(node.Field("right"))
	})
	ctx.Walk.IterKind(parser.KindAssignmentExpression, func(node *parser.Node) {
		if node.Tok.Kind != token.SlashAssign && node.Tok.Kind != token.PercentAssign {
			return
		}
		check(node.Field("right"))
	})
}

type InvalidShiftCount struct{}

func (InvalidShiftCount) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "invalid-shift-count",
		Name:            "Invalid shift count",
		Summary:         "Reports constant shift counts outside the 32-bit cell width",
		Explanation:     "Pawn cells are 32 bits wide. A constant shift count must be between 0 and 31.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"constants", "bitwise", "control-flow"},
	}
}

func (InvalidShiftCount) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	check := func(node *parser.Node) {
		switch node.Tok.Kind {
		case token.Shl, token.Shr, token.Ushr, token.ShlAssign, token.ShrAssign, token.UshrAssign:
		default:
			return
		}
		right := node.Field("right")
		value, ok := ctx.Eval(right)
		if !ok || value >= 0 && value < 32 {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "constant shift count must be between 0 and 31",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(right),
		})
	}
	ctx.Walk.IterKind(parser.KindBinaryExpression, check)
	ctx.Walk.IterKind(parser.KindAssignmentExpression, check)
}

type InvalidArraySize struct{}

func (InvalidArraySize) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "negative-or-zero-array-size",
		Name:            "Negative or zero array size",
		Summary:         "Reports array dimensions that evaluate to zero or less",
		Explanation:     "A declared array dimension must be greater than zero. The rule reports only dimensions that can be evaluated with certainty.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"constants", "arrays", "semantic"},
	}
}

type OutOfBoundsConstantIndex struct{}

func (OutOfBoundsConstantIndex) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "out-of-bounds-constant-index",
		Name:            "Out-of-bounds constant index",
		Summary:         "Reports constant indexes outside a known array dimension",
		Explanation:     "A constant index must be between zero and the array size minus one. The rule checks direct indexing when both the symbol and first dimension are known.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"constants", "arrays", "control-flow"},
	}
}

func (OutOfBoundsConstantIndex) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindSubscriptExpression, func(node *parser.Node) {
		array := node.Field("array")
		index := node.Field("index")
		if array == nil || array.Kind != parser.KindIdentifier {
			return
		}
		symbol := ctx.Semantic.Resolve(array)
		if symbol == nil || symbol.Decl == nil {
			return
		}
		dimension := symbol.Decl.Field("array")
		if dimension == nil {
			return
		}
		size, sizeOK := ctx.Constant(dimension.Field("size"))
		value, valueOK := ctx.Eval(index)
		if !sizeOK || !valueOK || value >= 0 && value < size {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("constant index %d is outside array size %d", value, size),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(index),
		})
	})
}

func (InvalidArraySize) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindDimension, func(node *parser.Node) {
		parent := ctx.Walk.Parent(node)
		if parent == nil || (parent.Kind != parser.KindVariableDeclarator && parent.Kind != parser.KindParameter) {
			return
		}
		size := node.Field("size")
		value, ok := ctx.Constant(size)
		if !ok || value > 0 {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "array size must be greater than zero",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(size),
		})
	})
}
