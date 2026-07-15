package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type FormatArgumentTag struct{}

func (FormatArgumentTag) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "format-argument-tag",
		Name:            "Format argument tag",
		Summary:         "Reports definite tag mismatches in formatted native calls",
		Explanation:     "The rule checks literal formats used by natives with formatParameter metadata. %f requires Float values, while integer specifiers reject Float values. String and library-dependent specifiers are skipped.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"format", "arguments", "native", "api", "tags"},
	}
}

func (FormatArgumentTag) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(call *parser.Node) {
		native, name, ok := calledNative(ctx, call)
		if !ok || native.FormatParameter == 0 {
			return
		}
		arguments := call.Field("arguments")
		count, valid := argumentCount(ctx, arguments)
		if !valid || hasNamedArgument(arguments) || count != len(arguments.Children) {
			return
		}
		fixed := 0
		for _, parameter := range native.Parameters {
			if !parameter.Variadic {
				fixed++
			}
		}
		formatIndex := native.FormatParameter - 1
		if formatIndex >= len(arguments.Children) || count < fixed {
			return
		}
		value, literal := literalString(ctx, arguments.Children[formatIndex])
		if !literal {
			return
		}
		specifiers, recognized := formatSpecifiers(value)
		if !recognized || len(specifiers) != count-fixed {
			return
		}
		for index, specifier := range specifiers {
			argument := arguments.Children[fixed+index]
			tag, known := definiteExpressionTag(ctx, argument)
			if !known || formatTagCompatible(specifier, tag) {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  formatTagMessage(name, index, specifier, tag),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(argument),
			})
		}
	})
}

func definiteExpressionTag(ctx *lint.Context, node *parser.Node) (string, bool) {
	if tags := ctx.ExpressionTags(node); len(tags) == 1 {
		return tags[0], true
	}
	node = unwrapParentheses(node)
	if node == nil || node.HasError {
		return "", false
	}
	if node.Kind == parser.KindLiteral {
		switch node.Tok.Kind {
		case token.FloatLiteral:
			return "Float", true
		case token.IntLiteral, token.CharLiteral, token.KwNull:
			return "", true
		}
	}
	if node.Kind == parser.KindIdentifier {
		symbol := ctx.Semantic.Resolve(node)
		if symbol != nil && !symbol.Ambiguous && len(symbol.Tags) == 0 {
			return "", true
		}
	}
	if node.Kind == parser.KindCallExpression {
		native, _, ok := calledNative(ctx, node)
		if ok {
			return native.ReturnTag, true
		}
	}
	if _, known := ctx.Eval(node); known {
		return "", true
	}
	return "", false
}

func formatTagCompatible(specifier byte, tag string) bool {
	switch specifier {
	case 'f':
		return tag == "Float"
	case 'i', 'd', 'c', 'x', 'b':
		return tag != "Float"
	default:
		return true
	}
}

func formatTagMessage(native string, index int, specifier byte, tag string) string {
	actual := "untagged"
	if tag != "" {
		actual = "tag " + tag
	}
	if specifier == 'f' {
		return fmt.Sprintf("format argument %d to %q uses %%f and requires tag Float, but is %s", index+1, native, actual)
	}
	return fmt.Sprintf("format argument %d to %q uses %%%c for a cell value, but has %s", index+1, native, specifier, actual)
}
