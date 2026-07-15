package maintainability

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type FunctionLength struct{}

func (FunctionLength) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "function-length",
		Name:            "Function length",
		Summary:         "Reports functions spanning too many source lines",
		Explanation:     "Physical lines are counted from the function signature through the end of its body, including blank and comment lines. Inactive and uncertain conditional-compilation branches are excluded.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"size", "functions", "maintainability"},
		Options: []lint.Option{{
			Name: "maximum", Summary: "Maximum physical lines per function",
			Type: lint.OptionInteger, Default: int64(100), Minimum: 1, Maximum: 1_000_000, HasMinimum: true, HasMaximum: true,
		}},
	}
}

func (FunctionLength) Run(ctx *lint.Context) {
	maximum := configuredFunctionLengthMaximum(ctx)
	for _, function := range ctx.Walk.OfKind(parser.KindFunctionDefinition) {
		if function.HasError || ctx.Walk.Inactive(function) || ctx.Walk.Uncertain(function) {
			continue
		}
		lines := activeFunctionLines(ctx, function)
		if lines <= maximum {
			continue
		}
		name := function.Field("name")
		if name == nil {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("function %q spans %d lines, exceeding the maximum of %d", ctx.Walk.Text(name), lines, maximum),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(name),
		})
	}
}

func configuredFunctionLengthMaximum(ctx *lint.Context) int {
	if ctx.PerRule != nil && ctx.PerRule["function-length"] != nil {
		if value, ok := ctx.PerRule["function-length"]["maximum"].(int64); ok && value > 0 {
			return int(value)
		}
	}
	return 100
}

func activeFunctionLines(ctx *lint.Context, function *parser.Node) int {
	startLine := ctx.Walk.LineTable.Lookup(function.Start).Line
	endLine := ctx.Walk.LineTable.Lookup(function.End).Line
	if endLine < startLine {
		return 0
	}
	var excluded map[int]bool
	var visit func(*parser.Node)
	visit = func(node *parser.Node) {
		if node == nil {
			return
		}
		if node.Kind == parser.KindConditionalBranch && excludedFunctionLengthBranch(ctx, node) {
			first := ctx.Walk.LineTable.Lookup(node.Start).Line
			last := ctx.Walk.LineTable.Lookup(node.End).Line
			if excluded == nil {
				excluded = make(map[int]bool, last-first+1)
			}
			for line := first; line <= last; line++ {
				if line >= startLine && line <= endLine {
					excluded[line] = true
				}
			}
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(function.Field("body"))
	return endLine - startLine + 1 - len(excluded)
}

func excludedFunctionLengthBranch(ctx *lint.Context, branch *parser.Node) bool {
	for _, child := range branch.Children {
		if ctx.Walk.Inactive(child) || ctx.Walk.Uncertain(child) {
			return true
		}
	}
	return false
}
