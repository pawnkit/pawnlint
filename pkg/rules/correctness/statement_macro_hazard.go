package correctness

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type StatementMacroHazard struct{}

func (StatementMacroHazard) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "statement-macro-hazard",
		Name:            "Statement macro hazard",
		Summary:         "Reports statement macros unsafe in unbraced control flow",
		Explanation:     "A function-like macro with multiple unwrapped statements, an embedded terminating semicolon, or an unmatched if can change surrounding control flow. The rule accepts single expressions, blocks, do-while wrappers, and complete if-else expansions. Uncertain, inactive, malformed, and declaration-generating macros are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"macros", "statements", "control-flow"},
	}
}

func (StatementMacroHazard) Run(ctx *lint.Context) {
	for _, define := range ctx.Walk.OfKind(parser.KindDirectiveDefine) {
		if define.HasError || define.Field("parameters") == nil || ctx.Walk.Inactive(define) || ctx.Walk.Uncertain(define) {
			continue
		}
		value := define.Field("value")
		if value == nil || value.HasError {
			continue
		}
		tokens := statementMacroTokens(ctx, value)
		if len(tokens) == 0 || !statementMacroBalanced(tokens) || statementMacroDeclaration(tokens[0].Kind) || statementMacroWrapped(value, tokens) {
			continue
		}
		reason := statementMacroReason(value, tokens)
		if reason == "" {
			continue
		}
		name := ctx.Walk.Text(define.Field("name"))
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("macro %q %s", name, reason),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(value),
		})
	}
}

func statementMacroReason(value *parser.Node, tokens []token.Token) string {
	semicolons := statementMacroTopLevelSemicolons(tokens)
	if tokens[len(tokens)-1].Kind == token.Semicolon {
		return "includes a terminating semicolon that can break surrounding control flow"
	}
	if semicolons > 0 {
		if tokens[0].Kind == token.KwIf && statementMacroHasTopLevelElse(tokens) {
			return ""
		}
		return "expands to multiple unwrapped statements"
	}
	if value.Kind == parser.KindIfStatement && value.Field("alternative") == nil || tokens[0].Kind == token.KwIf && !statementMacroHasTopLevelElse(tokens) {
		return "expands to an unmatched if statement"
	}
	return ""
}

func statementMacroBalanced(tokens []token.Token) bool {
	parentheses := 0
	brackets := 0
	braces := 0
	for _, tok := range tokens {
		switch tok.Kind {
		case token.LParen:
			parentheses++
		case token.RParen:
			parentheses--
		case token.LBracket:
			brackets++
		case token.RBracket:
			brackets--
		case token.LBrace:
			braces++
		case token.RBrace:
			braces--
		}
		if parentheses < 0 || brackets < 0 || braces < 0 {
			return false
		}
	}
	return parentheses == 0 && brackets == 0 && braces == 0
}

func statementMacroWrapped(value *parser.Node, tokens []token.Token) bool {
	if value.Kind == parser.KindBlock || tokens[0].Kind == token.LBrace && tokens[len(tokens)-1].Kind == token.RBrace {
		return true
	}
	if tokens[0].Kind != token.KwDo || statementMacroTopLevelSemicolons(tokens) != 0 {
		return false
	}
	for _, tok := range tokens {
		if tok.Kind == token.KwWhile {
			return true
		}
	}
	return false
}

func statementMacroDeclaration(kind token.Kind) bool {
	switch kind {
	case token.KwForward, token.KwNative, token.KwPublic, token.KwStock:
		return true
	default:
		return false
	}
}

func statementMacroTokens(ctx *lint.Context, value *parser.Node) []token.Token {
	if ctx.File == nil || ctx.File.Parsed == nil {
		return nil
	}
	var result []token.Token
	for _, tok := range ctx.File.Parsed.Tokens {
		if tok.Start.Offset < value.Start || tok.End.Offset > value.End || tok.Kind == token.EOF || tok.Kind.IsTrivia() {
			continue
		}
		result = append(result, tok)
	}
	return result
}

func statementMacroTopLevelSemicolons(tokens []token.Token) int {
	parentheses := 0
	brackets := 0
	braces := 0
	count := 0
	for _, tok := range tokens {
		switch tok.Kind {
		case token.LParen:
			parentheses++
		case token.RParen:
			parentheses--
		case token.LBracket:
			brackets++
		case token.RBracket:
			brackets--
		case token.LBrace:
			braces++
		case token.RBrace:
			braces--
		case token.Semicolon:
			if parentheses == 0 && brackets == 0 && braces == 0 {
				count++
			}
		}
	}
	return count
}

func statementMacroHasTopLevelElse(tokens []token.Token) bool {
	parentheses := 0
	brackets := 0
	braces := 0
	for _, tok := range tokens {
		switch tok.Kind {
		case token.LParen:
			parentheses++
		case token.RParen:
			parentheses--
		case token.LBracket:
			brackets++
		case token.RBracket:
			brackets--
		case token.LBrace:
			braces++
		case token.RBrace:
			braces--
		case token.KwElse:
			if parentheses == 0 && brackets == 0 && braces == 0 {
				return true
			}
		}
	}
	return false
}
