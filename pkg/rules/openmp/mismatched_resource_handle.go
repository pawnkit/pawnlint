package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MismatchedResourceHandle struct{}

func (MismatchedResourceHandle) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "mismatched-resource-handle",
		Name:            "Mismatched resource handle",
		Summary:         "Reports handles passed to the wrong resource releaser",
		Explanation:     "File, database, and database-result handles have distinct release functions. The rule reports calls only when the argument has one definite incompatible resource tag.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"resource", "handle", "database", "file", "tag"},
	}
}

func (MismatchedResourceHandle) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(call *parser.Node) {
		callable, ok := calledResourceFunction(ctx, call)
		expected, release := resourceReleaserTag(ctx, callable)
		if !ok || !release {
			return
		}
		arguments := call.Field("arguments")
		if _, ok := argumentCount(ctx, arguments); !ok || hasNamedArgument(arguments) || len(arguments.Children) == 0 {
			return
		}
		argument := arguments.Children[0]
		actual, ok := resourceTag(ctx, argument)
		if !ok || actual == expected {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("%q releases %s handles, but this argument has tag %s", callable.name, expected, actual),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(argument),
		})
	})
}

func resourceReleaserTag(ctx *lint.Context, callable resourceCallable) (string, bool) {
	if len(callable.parameters) == 0 || callable.parameters[0].Tag == "" {
		return "", false
	}
	for _, native := range ctx.Natives() {
		if native.Release == callable.name {
			return callable.parameters[0].Tag, true
		}
	}
	for _, function := range ctx.Functions() {
		if function.Release == callable.name {
			return callable.parameters[0].Tag, true
		}
	}
	return "", false
}

func resourceTag(ctx *lint.Context, node *parser.Node) (string, bool) {
	if tags := ctx.ExpressionTags(node); len(tags) == 1 {
		return tags[0], true
	}
	node = unwrapParentheses(node)
	if node == nil || node.Kind != parser.KindCallExpression {
		return "", false
	}
	callable, ok := calledResourceFunction(ctx, node)
	if !ok || callable.returnTag == "" {
		return "", false
	}
	return callable.returnTag, true
}
