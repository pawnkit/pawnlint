package maintainability

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	discovery "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type RestrictedSyntax struct{}

func (RestrictedSyntax) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "restricted-syntax",
		Name:            "Restricted syntax",
		Summary:         "Reports configured language and dependency restrictions",
		Explanation:     "Project policy can restrict calls to exact functions or natives, include path globs, global variables, direct and mutual recursion, and goto statements. Calls and recursion are reported only when resolution is definite. Inactive and uncertain syntax is skipped.",
		Category:        diagnostic.CategoryRestriction,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"restriction", "policy", "project", "syntax"},
		Options: []lint.Option{
			{Name: "functions", Summary: "Exact function names whose calls are restricted", Type: lint.OptionStringList, Default: []string{}, Validate: validateRestrictedNames},
			{Name: "natives", Summary: "Exact native names whose calls are restricted", Type: lint.OptionStringList, Default: []string{}, Validate: validateRestrictedNames},
			{Name: "includes", Summary: "Include path glob patterns to restrict", Type: lint.OptionStringList, Default: []string{}, Validate: validateRestrictedIncludes},
			{Name: "globals", Summary: "Restrict global variable declarations", Type: lint.OptionBoolean, Default: false},
			{Name: "recursion", Summary: "Restrict direct and mutual recursion", Type: lint.OptionBoolean, Default: false},
			{Name: "goto", Summary: "Restrict goto statements", Type: lint.OptionBoolean, Default: false},
		},
		ConfigExample: `[rules.restricted-syntax]
severity = "warning"
functions = ["LegacyFunction"]
natives = ["printf"]
includes = ["legacy/**"]
globals = true
recursion = true
goto = true`,
	}
}

func validateRestrictedNames(value any) error {
	names, _ := value.([]string)
	for _, name := range names {
		if !restrictedIdentifier.MatchString(name) {
			return fmt.Errorf("entries must be supported Pawn identifiers")
		}
	}
	return nil
}

var restrictedIdentifier = regexp.MustCompile(`^[_@A-Za-z][_@A-Za-z0-9]*$`)

func validateRestrictedIncludes(value any) error {
	patterns, _ := value.([]string)
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) != pattern || pattern == "" {
			return fmt.Errorf("entries must be non-empty include globs")
		}
	}
	return nil
}

func (RestrictedSyntax) Run(ctx *lint.Context) {
	if ctx.Project == nil || ctx.Semantic == nil {
		return
	}
	file := ctx.Project.File(ctx.File.Path)
	if file == nil {
		return
	}
	options := restrictedSyntaxOptions(ctx)
	reportRestrictedIncludes(ctx, file, options.includes)
	reportRestrictedCalls(ctx, file, options.functions, options.natives)
	if options.globals {
		reportRestrictedGlobals(ctx)
	}
	if options.gotoStatements {
		reportRestrictedGoto(ctx)
	}
	if options.recursion {
		reportRestrictedRecursion(ctx, file)
	}
}

type restrictedOptions struct {
	functions      map[string]bool
	natives        map[string]bool
	includes       []string
	globals        bool
	recursion      bool
	gotoStatements bool
}

func restrictedSyntaxOptions(ctx *lint.Context) restrictedOptions {
	options := restrictedOptions{functions: make(map[string]bool), natives: make(map[string]bool)}
	if ctx.PerRule == nil || ctx.PerRule["restricted-syntax"] == nil {
		return options
	}
	values := ctx.PerRule["restricted-syntax"]
	for _, name := range stringOption(values, "functions") {
		options.functions[name] = true
	}
	for _, name := range stringOption(values, "natives") {
		options.natives[name] = true
	}
	options.includes = stringOption(values, "includes")
	options.globals, _ = values["globals"].(bool)
	options.recursion, _ = values["recursion"].(bool)
	options.gotoStatements, _ = values["goto"].(bool)
	return options
}

func stringOption(values map[string]any, name string) []string {
	items, _ := values[name].([]string)
	return items
}

func reportRestrictedIncludes(ctx *lint.Context, file *project.File, patterns []string) {
	for _, include := range file.Includes {
		if include == nil || include.Node == nil || include.Uncertain || include.Path == "" {
			continue
		}
		path := strings.ReplaceAll(include.Path, "\\", "/")
		for _, pattern := range patterns {
			if !discovery.MatchGlob(pattern, path) {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message: fmt.Sprintf("include %q is restricted by pattern %q", include.Path, pattern),
				Range:   file.Walk.Range(include.Node.Field("path")),
			})
			break
		}
	}
}

func reportRestrictedCalls(ctx *lint.Context, file *project.File, functions, natives map[string]bool) {
	if len(functions) == 0 && len(natives) == 0 {
		return
	}
	for _, call := range ctx.Walk.OfKind(parser.KindCallExpression) {
		if ctx.Walk.Uncertain(call) || ctx.Walk.Inactive(call) {
			continue
		}
		callee := call.Field("function")
		if callee == nil || callee.Kind != parser.KindIdentifier {
			continue
		}
		name := ctx.Walk.Text(callee)
		if !functions[name] && !natives[name] {
			continue
		}
		kind, resolved := restrictedCallKind(ctx, file, callee)
		if !resolved || kind == "function" && !functions[name] || kind == "native" && !natives[name] {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message: fmt.Sprintf("call to %s %q is restricted", kind, name),
			Range:   ctx.Walk.Range(callee),
		})
	}
}

func restrictedCallKind(ctx *lint.Context, file *project.File, callee *parser.Node) (string, bool) {
	if declaration, ok := ctx.Project.Resolve(file, callee); ok && declaration.Kind == semantic.SymbolFunction && declaration.Node != nil {
		if walk.HasChildToken(declaration.Node, token.KwNative) {
			return "native", true
		}
		return "function", true
	}
	if symbol := ctx.Semantic.ResolveAsCallTarget(callee); symbol != nil {
		if symbol.Kind != semantic.SymbolFunction || symbol.Decl == nil || symbol.Ambiguous {
			return "", false
		}
		if walk.HasChildToken(symbol.Decl, token.KwNative) {
			return "native", true
		}
		return "function", true
	}
	if _, ok := ctx.Natives()[ctx.Walk.Text(callee)]; ok {
		return "native", true
	}
	return "", false
}

func reportRestrictedGlobals(ctx *lint.Context) {
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolGlobal || symbol.Ambiguous || symbol.Decl == nil || symbol.NameNode == nil || ctx.Walk.Uncertain(symbol.Decl) || ctx.Walk.Inactive(symbol.Decl) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message: fmt.Sprintf("global variable %q is restricted", symbol.Name),
			Range:   ctx.Walk.Range(symbol.NameNode),
		})
	}
}

func reportRestrictedGoto(ctx *lint.Context) {
	for _, statement := range ctx.Walk.OfKind(parser.KindGotoStatement) {
		if statement.HasError || ctx.Walk.Uncertain(statement) || ctx.Walk.Inactive(statement) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{Message: "goto statement is restricted", Range: ctx.Walk.Range(statement)})
	}
}

func reportRestrictedRecursion(ctx *lint.Context, file *project.File) {
	if ctx.Project.CallGraph == nil || !ctx.Project.InProgram(file) {
		return
	}
	for _, component := range ctx.Project.CallGraph.RecursiveComponents() {
		members := make(map[*parser.Node]bool, len(component))
		names := make([]string, 0, len(component))
		for _, function := range component {
			members[function.Node] = true
			names = append(names, function.Name)
		}
		for _, function := range component {
			if function.File != file || function.Symbol == nil {
				continue
			}
			message := fmt.Sprintf("function %q participates in restricted recursion cycle %s", function.Name, strings.Join(names, " -> "))
			if len(component) == 1 {
				message = fmt.Sprintf("function %q uses restricted direct recursion", function.Name)
			}
			diagnosticValue := diagnostic.Diagnostic{Message: message, Range: file.Walk.Range(function.Symbol.NameNode)}
			for _, call := range ctx.Project.CallGraph.Outgoing(function) {
				if call.File == file && members[call.Callee.Node] {
					diagnosticValue.Notes = append(diagnosticValue.Notes, diagnostic.RelatedLocation{
						Range:   file.Walk.Range(call.Node.Field("function")),
						Message: fmt.Sprintf("recursive call to %q is here", call.Callee.Name),
					})
					break
				}
			}
			ctx.Report(diagnosticValue)
		}
	}
}
