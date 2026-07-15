package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type SwappedArguments struct{}

func (SwappedArguments) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "swapped-arguments",
		Name:            "Swapped arguments",
		Summary:         "Reports native arguments whose tags match each other's parameters",
		Explanation:     "Two positional arguments are reported only when both have one definite tag, both expected parameter tags are distinct, and exchanging the arguments resolves both mismatches. Named, macro-structured, incomplete, and locally overridden calls are skipped.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"calls", "arguments", "api", "tags"},
	}
}

func (SwappedArguments) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(call *parser.Node) {
		native, name, ok := calledNative(ctx, call)
		if !ok {
			return
		}
		arguments := call.Field("arguments")
		count, valid := argumentCount(ctx, arguments)
		if !valid || hasNamedArgument(arguments) || count != len(arguments.Children) || count > len(native.Parameters) {
			return
		}
		actual := make([]string, count)
		for index, argument := range arguments.Children {
			actual[index], _ = definiteArgumentTag(ctx, argument)
		}
		mismatches := make([]int, 0, 2)
		for index := range actual {
			expected := native.Parameters[index].Tag
			if actual[index] != "" && expected != "" && actual[index] != expected {
				mismatches = append(mismatches, index)
			}
		}
		if len(mismatches) != 2 {
			return
		}
		first, second := mismatches[0], mismatches[1]
		firstExpected := native.Parameters[first].Tag
		secondExpected := native.Parameters[second].Tag
		if firstExpected == secondExpected || actual[first] != secondExpected || actual[second] != firstExpected {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("arguments %d and %d to %q appear to be swapped", first+1, second+1, name),
			Filename: ctx.File.Path,
			Range:    ctx.File.LineTable.Range(arguments.Children[first].Start, arguments.Children[second].End),
		})
	})
}

func definiteArgumentTag(ctx *lint.Context, node *parser.Node) (string, bool) {
	if tags := ctx.Semantic.ExpressionTags(node); len(tags) == 1 {
		return tags[0], true
	}
	node = unwrapParentheses(node)
	if node == nil || node.Kind != parser.KindCallExpression {
		return "", false
	}
	native, _, ok := calledNative(ctx, node)
	if !ok || native.ReturnTag == "" {
		return "", false
	}
	return native.ReturnTag, true
}
