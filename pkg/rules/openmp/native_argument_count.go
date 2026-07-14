package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type NativeArgumentCount struct{}

func (NativeArgumentCount) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "native-argument-count",
		Name:            "Native argument count",
		Summary:         "Reports calls with an impossible number of arguments for a known native",
		Explanation:     "Known open.mp and SA-MP native signatures define their required, optional, and variadic parameters. The rule reports only calls outside that permitted range and skips locally defined functions. Macro-concatenated argument fragments are grouped using the parser token stream.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"native", "arguments", "api"},
	}
}

func (NativeArgumentCount) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		native, name, ok := calledNative(ctx, node)
		if !ok {
			return
		}
		count, ok := argumentCount(ctx, node.Field("arguments"))
		if !ok {
			return
		}
		minimum, maximum, variadic := nativeArity(native)
		if count >= minimum && (variadic || count <= maximum) {
			return
		}
		message := fmt.Sprintf("native %q expects %d %s, but %d %s provided", name, minimum, argumentWord(minimum), count, providedVerb(count))
		if variadic {
			message = fmt.Sprintf("native %q expects at least %d %s, but %d %s provided", name, minimum, argumentWord(minimum), count, providedVerb(count))
		} else if minimum != maximum {
			message = fmt.Sprintf("native %q expects between %d and %d arguments, but %d %s provided", name, minimum, maximum, count, providedVerb(count))
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  message,
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	})
}

func nativeArity(native api.Native) (minimum, maximum int, variadic bool) {
	for _, parameter := range native.Parameters {
		if parameter.Variadic {
			variadic = true
			continue
		}
		maximum++
		if !parameter.Default {
			minimum++
		}
	}
	return minimum, maximum, variadic
}
