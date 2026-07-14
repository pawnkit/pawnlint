package openmp

import (
	"fmt"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type FormatArgumentCount struct{}

func (FormatArgumentCount) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "format-argument-count",
		Name:            "Format argument count",
		Summary:         "Reports literal format strings whose placeholders and arguments differ",
		Explanation:     "Pawn format placeholders consume variadic arguments in order. The rule checks direct calls to known formatted natives when the format is a literal and every specifier is recognized. Dynamic strings and named arguments are skipped.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"format", "arguments", "native", "api"},
	}
}

func (FormatArgumentCount) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		native, _, ok := calledNative(ctx, node)
		if !ok || native.FormatParameter == 0 {
			return
		}
		arguments := node.Field("arguments")
		count, ok := argumentCount(ctx, arguments)
		if !ok || hasNamedArgument(arguments) {
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
		formatNode := arguments.Children[formatIndex]
		value, ok := literalString(ctx, formatNode)
		if !ok {
			return
		}
		required, ok := formatArgumentCount(value)
		if !ok {
			return
		}
		provided := count - fixed
		if required == provided {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("format string requires %d %s, but %d %s provided", required, argumentWord(required), provided, providedVerb(provided)),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(formatNode),
		})
	})
}

func literalString(ctx *lint.Context, node *parser.Node) (string, bool) {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	if node == nil || node.HasError {
		return "", false
	}
	if node.Kind == parser.KindStringConcat {
		var value string
		for _, child := range node.Children {
			part, ok := literalString(ctx, child)
			if !ok {
				return "", false
			}
			value += part
		}
		return value, true
	}
	if node.Kind != parser.KindLiteral || node.Tok.Kind != token.StringLiteral && node.Tok.Kind != token.PackedString {
		return "", false
	}
	if literalContinues(ctx, node) {
		return "", false
	}
	value := node.Tok.Text(ctx.File.Source)
	start := -1
	quotes := 0
	escaped := false
	for i := range value {
		if value[i] == '\\' && !escaped {
			escaped = true
			continue
		}
		if value[i] == '"' && !escaped {
			quotes++
			if start >= 0 && i != len(value)-1 {
				return "", false
			}
			start = i + 1
		}
		escaped = false
	}
	if quotes != 2 || start != len(value) || value[len(value)-1] != '"' {
		return "", false
	}
	first := strings.IndexByte(value, '"')
	return value[first+1 : len(value)-1], true
}

func literalContinues(ctx *lint.Context, node *parser.Node) bool {
	if ctx == nil || ctx.File == nil || ctx.File.Parsed == nil || node == nil {
		return false
	}
	for index := range ctx.File.Parsed.Tokens {
		current := &ctx.File.Parsed.Tokens[index]
		if current.Start.Offset != node.Tok.Start.Offset || current.End.Offset != node.Tok.End.Offset || index+1 >= len(ctx.File.Parsed.Tokens) {
			continue
		}
		next := &ctx.File.Parsed.Tokens[index+1]
		return next.Kind == token.Identifier || next.Kind == token.StringLiteral || next.Kind == token.PackedString
	}
	return false
}

func formatArgumentCount(value string) (int, bool) {
	count := 0
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			continue
		}
		i++
		if i >= len(value) {
			return 0, false
		}
		if value[i] == '%' {
			continue
		}
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		if i < len(value) && value[i] == '.' {
			i++
			for i < len(value) && value[i] >= '0' && value[i] <= '9' {
				i++
			}
		}
		if i >= len(value) || !formatSpecifier(value[i]) {
			return 0, false
		}
		count++
	}
	return count, true
}

func formatSpecifier(value byte) bool {
	switch value {
	case 'i', 'd', 's', 'f', 'c', 'x', 'b', 'q':
		return true
	default:
		return false
	}
}
