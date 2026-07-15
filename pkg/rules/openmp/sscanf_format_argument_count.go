package openmp

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type SscanfFormatArgumentCount struct{}

func (SscanfFormatArgumentCount) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "sscanf-format-argument-count",
		Name:            "sscanf format argument count",
		Summary:         "Reports sscanf() calls whose format string and argument count differ",
		Explanation:     explanationSscanfFormatArgumentCount,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"sscanf", "format", "arguments"},
	}
}

const explanationSscanfFormatArgumentCount = `sscanf's format string lists one specifier per output argument (aside from
zero-argument directives like ` + "`p<delim>`" + ` and ` + "`{skipped}`" + ` groups). A mismatch
between the specifier count and the argument list either fails silently or
crashes at runtime:

` + "```pawn" + `
sscanf(params, "dd", id); // "dd" needs 2 arguments, only 1 given
` + "```" + `

The rule only checks calls with a literal format string and recognizes the
documented specifier set; format strings using unrecognized letters are
skipped rather than guessed at.`

func (SscanfFormatArgumentCount) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		if !isSscanfCall(ctx, node) {
			return
		}
		arguments := node.Field("arguments")
		count, ok := argumentCount(ctx, arguments)
		if !ok || hasNamedArgument(arguments) || count < 2 {
			return
		}
		formatNode := arguments.Children[1]
		spec, ok := literalString(ctx, formatNode)
		if !ok {
			return
		}
		required, ok := sscanfSpecifierCount(spec)
		if !ok {
			return
		}
		provided := count - 2
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

func isSscanfCall(ctx *lint.Context, call *parser.Node) bool {
	callee := call.Field("function")
	return callee != nil && callee.Kind == parser.KindIdentifier && ctx.Walk.Text(callee) == "sscanf"
}

func sscanfSpecifierCount(spec string) (count int, ok bool) {
	depth := 0
	i := 0
	for i < len(spec) {
		switch c := spec[i]; {
		case c == '{':
			depth++
			i++
		case c == '}':
			if depth == 0 {
				return 0, false
			}
			depth--
			i++
		case c == 'p' || c == 'P' || c == '?':
			i++
			if i >= len(spec) || spec[i] != '<' {
				return 0, false
			}
			end := indexByteFrom(spec, i, '>')
			if end < 0 {
				return 0, false
			}
			i = end + 1
		case isSscanfSpecifierLetter(c):
			i++
			for i < len(spec) {
				switch spec[i] {
				case '<':
					end := indexByteFrom(spec, i, '>')
					if end < 0 {
						return 0, false
					}
					i = end + 1
					continue
				case '[':
					end := indexByteFrom(spec, i, ']')
					if end < 0 {
						return 0, false
					}
					i = end + 1
					continue
				case '(':
					end, matched := matchParen(spec, i)
					if !matched {
						return 0, false
					}
					i = end + 1
					continue
				}
				break
			}
			if depth == 0 {
				count++
			}
		case isASCIILetter(c):
			return 0, false
		default:
			i++
		}
	}
	if depth != 0 {
		return 0, false
	}
	return count, true
}

func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isSscanfSpecifierLetter(c byte) bool {
	switch c {
	case 'd', 'i', 'f', 'b', 'h', 'x', 'o', 'n', 'c', 's', 'z', 'q', 'u', 'r', 'm', 'k',
		'D', 'I', 'F', 'B', 'H', 'X', 'O', 'N', 'C', 'S', 'Z', 'Q', 'U', 'R', 'M', 'K',
		'a', 'A':
		return true
	default:
		return false
	}
}

func indexByteFrom(s string, from int, b byte) int {
	for i := from; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func matchParen(s string, open int) (int, bool) {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}
