package correctness

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnparenthesizedMacro struct{}

func (UnparenthesizedMacro) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unparenthesized-macro",
		Name:            "Unparenthesized macro",
		Summary:         "Reports function-like macros whose replacement list or parameters lack protective parentheses",
		Explanation:     explanationUnparenthesizedMacro,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  true,
		Fixable:         true,
		Tags:            []string{"macros", "preprocessor"},
	}
}

const explanationUnparenthesizedMacro = `A function-like ` + "`#define`" + ` is expanded by simple text substitution. If a
parameter is used next to an operator without its own parentheses, or the
whole replacement list is not itself parenthesized, the operators at a call
site can silently change the computed result (the classic ` + "`#define SQUARE(x) x*x`" + `
called as ` + "`SQUARE(a+b)`" + ` bug). The rule only inspects replacement lists the
parser could parse as a single expression or statement, and only flags
parameter references or replacement lists that are direct operands of an
operator; already-parenthesized, call-argument, and subscript positions are
left alone. The fix always wraps the exact reported span in parentheses, which
never changes what it evaluates to.`

func (UnparenthesizedMacro) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindDirectiveDefine, func(define *parser.Node) {
		if ctx.Walk.Uncertain(define) {
			return
		}
		value := define.Field("value")
		if value == nil || value.Kind == parser.KindMacroBody || value.Kind == parser.KindRaw {
			return
		}
		macroName := ctx.Walk.Text(define.Field("name"))

		if macroOperandNeedsParens(value.Kind) && !signedLiteral(value) {
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("replacement list of macro %q should be parenthesized to avoid operator-precedence surprises at call sites", macroName),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(value),
				Fix:      parenthesizeFix(ctx, value, "wrap the macro replacement list in parentheses"),
			})
		}

		params := define.Field("parameters")
		if params == nil || len(params.Children) == 0 {
			return
		}
		paramNames := make(map[string]struct{}, len(params.Children))
		for _, p := range params.Children {
			if name := ctx.Walk.Text(p); name != "" {
				paramNames[name] = struct{}{}
			}
		}
		if len(paramNames) == 0 {
			return
		}

		walkMacroReplacement(value, nil, func(node, parent *parser.Node) {
			if node.Kind != parser.KindIdentifier || parent == nil {
				return
			}
			name := ctx.Walk.Text(node)
			if _, ok := paramNames[name]; !ok {
				return
			}
			if !macroOperandNeedsParens(parent.Kind) {
				return
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("parameter %q of macro %q should be parenthesized in the replacement list", name, macroName),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(node),
				Fix:      parenthesizeFix(ctx, node, "wrap the macro parameter in parentheses"),
			})
		})
	})
}

func signedLiteral(node *parser.Node) bool {
	if node.Kind != parser.KindUnaryExpression || len(node.Children) != 1 {
		return false
	}
	return (node.Tok.Kind == token.Plus || node.Tok.Kind == token.Minus) && node.Children[0].Kind == parser.KindLiteral
}

func macroOperandNeedsParens(kind parser.Kind) bool {
	switch kind {
	case parser.KindBinaryExpression, parser.KindUnaryExpression,
		parser.KindTernaryExpression, parser.KindUpdateExpression:
		return true
	default:
		return false
	}
}

func walkMacroReplacement(node, parent *parser.Node, visit func(node, parent *parser.Node)) {
	if node == nil {
		return
	}
	visit(node, parent)
	for _, child := range node.Children {
		walkMacroReplacement(child, node, visit)
	}
}

func parenthesizeFix(ctx *lint.Context, node *parser.Node, description string) *diagnostic.Fix {
	r := ctx.Walk.Range(node)
	return &diagnostic.Fix{
		Description: description,
		Edits: []diagnostic.Edit{{
			Range:   r,
			NewText: "(" + ctx.Walk.Text(node) + ")",
		}},
	}
}
