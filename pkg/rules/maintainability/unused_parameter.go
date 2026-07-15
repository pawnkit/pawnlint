package maintainability

import (
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnusedParameter struct{}

func (UnusedParameter) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unused-parameter",
		Name:            "Unused parameter",
		Summary:         "Reports unused parameters in non-public function definitions",
		Explanation:     explanationUnusedParameter,
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"unused", "parameters", "semantic"},
	}
}

const explanationUnusedParameter = `An unused parameter may indicate dead code or an incomplete function. Public
and command-handler functions are skipped because external signatures may require every parameter.
Functions wrapped by a hooking library (` + "`hook`" + `, ` + "`inline`" + `, and similar
single-word prefixes) are skipped for the same reason. Names beginning with
` + "`_`" + ` or listed in a ` + "`#pragma unused`" + ` directive in the same function are
treated as intentionally unused.`

func (UnusedParameter) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	pragmaUnused := pragmaUnusedNamesByFunction(ctx)
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolParameter || symbol.Ambiguous || strings.HasPrefix(symbol.Name, "_") {
			continue
		}
		if symbol.Function == nil || symbol.Function.Kind != parser.KindFunctionDefinition || hasExternalSignature(ctx, symbol.Function) {
			continue
		}
		if len(ctx.Semantic.References(symbol)) != 0 {
			continue
		}
		if pragmaUnused[symbol.Function][symbol.Name] {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "parameter " + quoteName(symbol.Name) + " is never used",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(symbol.NameNode),
		})
	}
}

func pragmaUnusedNamesByFunction(ctx *lint.Context) map[*parser.Node]map[string]bool {
	result := make(map[*parser.Node]map[string]bool)
	for _, node := range ctx.Walk.OfKind(parser.KindDirectivePragma) {
		function := ctx.Walk.EnclosingFunction(node)
		if function == nil {
			continue
		}
		names := pragmaUnusedNames(ctx.Walk.Text(node))
		if len(names) == 0 {
			continue
		}
		if result[function] == nil {
			result[function] = make(map[string]bool)
		}
		for _, name := range names {
			result[function][name] = true
		}
	}
	return result
}

func pragmaUnusedNames(text string) []string {
	const marker = "unused"
	index := strings.Index(text, marker)
	if index < 0 {
		return nil
	}
	fields := strings.Split(text[index+len(marker):], ",")
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func hasExternalSignature(ctx *lint.Context, function *parser.Node) bool {
	name := function.Field("name")
	if name != nil && ctx.Walk.Text(name) == "main" {
		return true
	}
	tag := strings.TrimSuffix(strings.ToLower(ctx.Walk.Text(function.Field("tag"))), ":")
	if tag == "command" || strings.HasSuffix(tag, "cmd") {
		return true
	}
	for _, child := range function.Children {
		if child.Tok.Kind == token.KwPublic {
			return true
		}
	}
	return walk.HasWrapperStorageQualifier(function)
}
