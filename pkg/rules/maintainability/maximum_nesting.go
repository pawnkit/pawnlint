package maintainability

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MaximumNesting struct{}

func (MaximumNesting) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "maximum-nesting",
		Name:            "Maximum nesting",
		Summary:         "Reports functions with deeply nested control statements",
		Explanation:     "Nesting depth increases for if, loop, and switch statements. Else-if chains remain at one level. Inactive and uncertain conditional-compilation branches are ignored.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"complexity", "nesting", "maintainability"},
		Options: []lint.Option{{
			Name: "maximum", Summary: "Maximum permitted nesting depth",
			Type: lint.OptionInteger, Default: int64(4), Minimum: 1, Maximum: 1000, HasMinimum: true, HasMaximum: true,
		}},
	}
}

func (MaximumNesting) Run(ctx *lint.Context) {
	maximum := configuredMaximumNesting(ctx)
	for _, function := range ctx.Walk.OfKind(parser.KindFunctionDefinition) {
		if ctx.Walk.Inactive(function) || function.HasError {
			continue
		}
		depth, deepest := maximumNestingDepth(ctx, function.Field("body"))
		if depth <= maximum || deepest == nil {
			continue
		}
		name := function.Field("name")
		if name == nil {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("function %q reaches nesting depth %d, exceeding the maximum of %d", ctx.Walk.Text(name), depth, maximum),
			Filename: ctx.File.Path,
			Range:    maximumNestingRange(ctx, deepest),
		})
	}
}

func configuredMaximumNesting(ctx *lint.Context) int {
	if ctx.PerRule != nil && ctx.PerRule["maximum-nesting"] != nil {
		if value, ok := ctx.PerRule["maximum-nesting"]["maximum"].(int64); ok && value > 0 {
			return int(value)
		}
	}
	return 4
}

func maximumNestingDepth(ctx *lint.Context, root *parser.Node) (int, *parser.Node) {
	maximum := 0
	var deepest *parser.Node
	var visit func(*parser.Node, int)
	visit = func(node *parser.Node, depth int) {
		if node == nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
			return
		}
		switch node.Kind {
		case parser.KindIfStatement:
			next := depth + 1
			if next > maximum {
				maximum = next
				deepest = node
			}
			visit(node.Field("condition"), next)
			visit(node.Field("consequence"), next)
			alternative := node.Field("alternative")
			if alternative != nil && alternative.Kind == parser.KindIfStatement {
				visit(alternative, depth)
			} else {
				visit(alternative, next)
			}
			return
		case parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement, parser.KindSwitchStatement:
			depth++
			if depth > maximum {
				maximum = depth
				deepest = node
			}
		}
		for _, child := range node.Children {
			visit(child, depth)
		}
	}
	visit(root, 0)
	return maximum, deepest
}

func maximumNestingRange(ctx *lint.Context, node *parser.Node) source.Range {
	if node.Tok.End.Offset > node.Tok.Start.Offset {
		return ctx.Walk.LineTable.Range(node.Tok.Start.Offset, node.Tok.End.Offset)
	}
	return ctx.Walk.Range(node)
}
