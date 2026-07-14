package openmp

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnescapedSQLFormat struct{}

func (UnescapedSQLFormat) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unescaped-sql-format",
		Name:            "Unescaped SQL format argument",
		Summary:         "Reports mysql_format calls using %s for a non-literal string argument",
		Explanation:     explanationUnescapedSQLFormat,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"sql", "security", "format", "native"},
	}
}

const explanationUnescapedSQLFormat = `mysql_format inserts ` + "`%s`" + ` arguments into the query exactly as given, while
` + "`%e`" + ` escapes them first. A non-literal ` + "`%s`" + ` argument that carries a player
name, chat input, or any other untrusted value is a SQL injection risk:

` + "```pawn" + `
mysql_format(handle, query, sizeof(query), "SELECT * FROM users WHERE name = '%s'", name);
` + "```" + `

Use ` + "`%e`" + ` instead so the value is escaped:

` + "```pawn" + `
mysql_format(handle, query, sizeof(query), "SELECT * FROM users WHERE name = '%e'", name);
` + "```" + `

The rule only flags ` + "`%s`" + ` arguments that are not a literal string written
directly in the call, since a literal cannot carry untrusted input. It only
recognizes calls literally named ` + "`mysql_format`" + `. No fix is offered because
some ` + "`%s`" + ` arguments are genuinely safe (e.g. a hardcoded column name).`

func (UnescapedSQLFormat) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		if node.HasError || ctx.Walk.Uncertain(node) {
			return
		}
		callee := node.Field("function")
		if callee == nil || callee.Kind != parser.KindIdentifier || callee.HasError || ctx.Walk.Text(callee) != "mysql_format" {
			return
		}
		if ctx.Semantic != nil && ctx.Semantic.Resolve(callee) != nil {
			return
		}
		arguments := node.Field("arguments")
		if _, ok := argumentCount(ctx, arguments); !ok || hasNamedArgument(arguments) || len(arguments.Children) < 4 {
			return
		}
		formatNode := arguments.Children[3]
		value, ok := literalString(ctx, formatNode)
		if !ok {
			return
		}
		specifiers, ok := formatSpecifierSequence(value)
		if !ok {
			return
		}
		variadic := arguments.Children[4:]
		for i, specifier := range specifiers {
			if specifier != 's' || i >= len(variadic) {
				continue
			}
			if _, isLiteral := literalString(ctx, variadic[i]); isLiteral {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  "mysql_format uses '%s' for a non-literal argument; use '%e' to escape it",
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(variadic[i]),
			})
		}
	})
}

func formatSpecifierSequence(value string) ([]byte, bool) {
	var specifiers []byte
	for i := 0; i < len(value); i++ {
		if value[i] != '%' {
			continue
		}
		i++
		if i >= len(value) {
			return nil, false
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
		if i >= len(value) || !sqlFormatSpecifier(value[i]) {
			return nil, false
		}
		specifiers = append(specifiers, value[i])
	}
	return specifiers, true
}

func sqlFormatSpecifier(value byte) bool {
	return value == 'e' || formatSpecifier(value)
}
