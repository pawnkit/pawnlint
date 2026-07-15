package openmp

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type SetTimerExFormatArgumentCount struct{}

func (SetTimerExFormatArgumentCount) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "settimerex-format-argument-count",
		Name:            "SetTimerEx format argument count",
		Summary:         "Reports SetTimerEx() calls whose specifier string and argument count differ",
		Explanation:     explanationSetTimerExFormatArgumentCount,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"timer", "format", "arguments"},
	}
}

const explanationSetTimerExFormatArgumentCount = `SetTimerEx's specifier string lists one letter per packed argument: i or d
(integer), f (float), s (string), b (boolean), a (array, immediately followed
by its own i size specifier). A mismatch between the specifiers and the
argument list passes the wrong values to the callback:

` + "```pawn" + `
SetTimerEx("OnDone", 1000, false, "dd", playerid); // "dd" needs 2 arguments
` + "```" + `

The rule only checks calls with a literal specifier string using the
documented letters; anything else is skipped rather than guessed at.`

func (SetTimerExFormatArgumentCount) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		native, _, ok := calledNative(ctx, node)
		if !ok || native.Name != "SetTimerEx" {
			return
		}
		arguments := node.Field("arguments")
		count, ok := argumentCount(ctx, arguments)
		if !ok || hasNamedArgument(arguments) || count < 4 {
			return
		}
		formatNode := arguments.Children[3]
		spec, ok := literalString(ctx, formatNode)
		if !ok {
			return
		}
		required, ok := setTimerExSpecifierCount(spec)
		if !ok {
			return
		}
		provided := count - 4
		if required == provided {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("specifier string requires %d %s, but %d %s provided", required, argumentWord(required), provided, providedVerb(provided)),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(formatNode),
		})
	})
}

func setTimerExSpecifierCount(spec string) (count int, ok bool) {
	for i := 0; i < len(spec); i++ {
		switch spec[i] {
		case 'i', 'd', 'a', 's', 'f', 'b':
			count++
		default:
			return 0, false
		}
	}
	return count, true
}
