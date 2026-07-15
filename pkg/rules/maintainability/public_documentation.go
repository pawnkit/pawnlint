package maintainability

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type PublicDocumentation struct{}

var publicDocumentationStorage = []string{"public"}

func (PublicDocumentation) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "public-documentation",
		Name:            "Public documentation",
		Summary:         "Reports selected functions without complete API documentation",
		Explanation:     "Selected functions require an adjacent Doxygen block or consecutive triple-slash comments. Documentation can require descriptions, parameter tags, and return tags. Name patterns and exclusions limit the documented surface.",
		Category:        diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"documentation", "policy", "functions"},
		Options: []lint.Option{
			{Name: "storage", Summary: "Function storage qualifiers to document", Type: lint.OptionStringList, Default: publicDocumentationStorage, Choices: []string{"public", "stock", "native", "forward"}, Validate: validatePublicDocumentationStorage},
			{Name: "include", Summary: "Function name regular expressions to include", Type: lint.OptionStringList, Default: []string{}, Validate: validateNamingPatterns},
			{Name: "exclude", Summary: "Function name regular expressions to exclude", Type: lint.OptionStringList, Default: []string{}, Validate: validateNamingPatterns},
			{Name: "minimum-description-length", Summary: "Minimum description length", Type: lint.OptionInteger, Default: int64(1), Minimum: 1, Maximum: 1000, HasMinimum: true, HasMaximum: true},
			{Name: "require-parameters", Summary: "Require matching parameter tags", Type: lint.OptionBoolean, Default: true},
			{Name: "require-return", Summary: "Require a return tag", Type: lint.OptionBoolean, Default: false},
		},
	}
}

func validatePublicDocumentationStorage(value any) error {
	values, _ := value.([]string)
	if len(values) == 0 {
		return fmt.Errorf("must contain at least one entry")
	}
	return nil
}

type publicDocumentationOptions struct {
	storage                  map[token.Kind]bool
	include                  []*regexp.Regexp
	exclude                  []*regexp.Regexp
	minimumDescriptionLength int
	requireParameters        bool
	requireReturn            bool
}

type functionDocumentation struct {
	description string
	parameters  map[string]int
	paramText   map[string]string
	returnCount int
	returnText  string
}

func (PublicDocumentation) Run(ctx *lint.Context) {
	options := configuredPublicDocumentationOptions(ctx)
	for _, kind := range []parser.Kind{parser.KindFunctionDeclaration, parser.KindFunctionDefinition} {
		for _, node := range ctx.Walk.OfKind(kind) {
			nameNode := node.Field("name")
			if node.HasError || nameNode == nil || !selectedDocumentedFunction(ctx, node, nameNode, options) {
				continue
			}
			name := ctx.Walk.Text(nameNode)
			lines, ok := publicDocumentationLines(ctx, node)
			if !ok {
				ctx.Report(diagnostic.Diagnostic{Message: fmt.Sprintf("function %q requires API documentation", name), Range: ctx.Walk.Range(nameNode)})
				continue
			}
			documentation := parseFunctionDocumentation(lines)
			if message := publicDocumentationViolation(ctx, node, documentation, options); message != "" {
				ctx.Report(diagnostic.Diagnostic{Message: fmt.Sprintf("function %q %s", name, message), Range: ctx.Walk.Range(nameNode)})
			}
		}
	}
}

func configuredPublicDocumentationOptions(ctx *lint.Context) publicDocumentationOptions {
	options := publicDocumentationOptions{
		storage:                  map[token.Kind]bool{token.KwPublic: true},
		minimumDescriptionLength: 1,
		requireParameters:        true,
	}
	if ctx.PerRule == nil || ctx.PerRule["public-documentation"] == nil {
		return options
	}
	values := ctx.PerRule["public-documentation"]
	if storage, ok := values["storage"].([]string); ok {
		options.storage = make(map[token.Kind]bool, len(storage))
		for _, value := range storage {
			options.storage[publicDocumentationStorageKind(value)] = true
		}
	}
	options.include = compilePublicDocumentationPatterns(values["include"])
	options.exclude = compilePublicDocumentationPatterns(values["exclude"])
	if length, ok := values["minimum-description-length"].(int64); ok {
		options.minimumDescriptionLength = int(length)
	}
	if value, ok := values["require-parameters"].(bool); ok {
		options.requireParameters = value
	}
	if value, ok := values["require-return"].(bool); ok {
		options.requireReturn = value
	}
	return options
}

func publicDocumentationStorageKind(value string) token.Kind {
	switch value {
	case "stock":
		return token.KwStock
	case "native":
		return token.KwNative
	case "forward":
		return token.KwForward
	default:
		return token.KwPublic
	}
}

func compilePublicDocumentationPatterns(value any) []*regexp.Regexp {
	patterns, _ := value.([]string)
	result := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			result = append(result, compiled)
		}
	}
	return result
}

func selectedDocumentedFunction(ctx *lint.Context, node, nameNode *parser.Node, options publicDocumentationOptions) bool {
	selected := false
	for kind := range options.storage {
		if walk.HasChildToken(node, kind) {
			selected = true
			break
		}
	}
	if !selected {
		return false
	}
	name := ctx.Walk.Text(nameNode)
	if len(options.include) != 0 && !matchesPublicDocumentationPattern(options.include, name) {
		return false
	}
	return !matchesPublicDocumentationPattern(options.exclude, name)
}

func matchesPublicDocumentationPattern(patterns []*regexp.Regexp, name string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

func publicDocumentationLines(ctx *lint.Context, node *parser.Node) ([]string, bool) {
	for index := len(node.Leading) - 1; index >= 0; index-- {
		trivia := node.Leading[index]
		if trivia.Kind != token.Comment {
			continue
		}
		if ctx.Walk.LineTable.Lookup(node.Start).Line-trivia.End.Line > 1 {
			return nil, false
		}
		text := trivia.Text(ctx.File.Source)
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "/**") {
			return blockDocumentationLines(trimmed), true
		}
		if !strings.HasPrefix(trimmed, "///") {
			return nil, false
		}
		comments := []string{text}
		previousLine := trivia.Start.Line
		for earlier := index - 1; earlier >= 0; earlier-- {
			candidate := node.Leading[earlier]
			if candidate.Kind != token.Comment {
				continue
			}
			candidateText := candidate.Text(ctx.File.Source)
			if previousLine-candidate.End.Line > 1 || !strings.HasPrefix(strings.TrimSpace(candidateText), "///") {
				break
			}
			comments = append(comments, candidateText)
			previousLine = candidate.Start.Line
		}
		lines := make([]string, 0, len(comments))
		for i := len(comments) - 1; i >= 0; i-- {
			line := strings.TrimSpace(comments[i])
			lines = append(lines, strings.TrimSpace(strings.TrimPrefix(line, "///")))
		}
		return lines, true
	}
	return nil, false
}

func blockDocumentationLines(text string) []string {
	text = strings.TrimPrefix(text, "/**")
	text = strings.TrimSuffix(text, "*/")
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		lines[index] = line
	}
	return lines
}

func parseFunctionDocumentation(lines []string) functionDocumentation {
	documentation := functionDocumentation{parameters: make(map[string]int), paramText: make(map[string]string)}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		brief, isBrief := documentationTagBody(line, "@brief", false)
		parameter, isParameter := documentationTagBody(line, "@param", true)
		returnValue, isReturn := documentationTagBody(line, "@return", false)
		switch {
		case isBrief:
			if documentation.description == "" {
				documentation.description = brief
			}
		case isParameter:
			name, description := parseDocumentationParameter(parameter)
			if name != "" {
				documentation.parameters[name]++
				documentation.paramText[name] = description
			}
		case isReturn:
			documentation.returnCount++
			documentation.returnText = returnValue
		case !strings.HasPrefix(line, "@") && documentation.description == "":
			documentation.description = line
		}
	}
	return documentation
}

func documentationTagBody(line, tag string, bracket bool) (string, bool) {
	if !strings.HasPrefix(line, tag) {
		return "", false
	}
	rest := line[len(tag):]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' && (!bracket || rest[0] != '[') {
		return "", false
	}
	return strings.TrimSpace(rest), true
}

func parseDocumentationParameter(rest string) (string, string) {
	if strings.HasPrefix(rest, "[") {
		if end := strings.IndexByte(rest, ']'); end >= 0 {
			rest = strings.TrimSpace(rest[end+1:])
		}
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", ""
	}
	name := fields[0]
	return name, strings.TrimSpace(rest[len(name):])
}

func publicDocumentationViolation(ctx *lint.Context, node *parser.Node, documentation functionDocumentation, options publicDocumentationOptions) string {
	if utf8.RuneCountInString(documentation.description) < options.minimumDescriptionLength {
		return fmt.Sprintf("requires a description of at least %d characters", options.minimumDescriptionLength)
	}
	parameters := publicDocumentationParameters(ctx, node)
	if options.requireParameters {
		for _, name := range parameters {
			if documentation.parameters[name] == 0 {
				return fmt.Sprintf("requires an @param entry for %q", name)
			}
			if documentation.parameters[name] > 1 {
				return fmt.Sprintf("documents parameter %q more than once", name)
			}
			if documentation.paramText[name] == "" {
				return fmt.Sprintf("requires a description for parameter %q", name)
			}
		}
		known := make(map[string]bool, len(parameters))
		for _, name := range parameters {
			known[name] = true
		}
		documented := make([]string, 0, len(documentation.parameters))
		for name := range documentation.parameters {
			documented = append(documented, name)
		}
		sort.Strings(documented)
		for _, name := range documented {
			if !known[name] {
				return fmt.Sprintf("documents unknown parameter %q", name)
			}
		}
	}
	if options.requireReturn {
		if documentation.returnCount == 0 {
			return "requires an @return entry"
		}
		if documentation.returnCount > 1 {
			return "documents its return value more than once"
		}
		if documentation.returnText == "" {
			return "requires an @return description"
		}
	}
	return ""
}

func publicDocumentationParameters(ctx *lint.Context, node *parser.Node) []string {
	list := node.Field("parameters")
	if list == nil {
		return nil
	}
	result := make([]string, 0, len(list.Children))
	for _, parameter := range list.Children {
		if name := parameter.Field("name"); name != nil {
			result = append(result, ctx.Walk.Text(name))
		}
	}
	return result
}
