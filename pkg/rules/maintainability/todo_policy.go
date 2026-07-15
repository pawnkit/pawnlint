package maintainability

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type TodoPolicy struct{}

const defaultTodoIssuePattern = `[A-Z][A-Z0-9]+-[0-9]+`

var defaultTodoTags = []string{"TODO", "FIXME"}
var compiledDefaultTodoIssuePattern = regexp.MustCompile(defaultTodoIssuePattern)

func (TodoPolicy) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "todo-policy",
		Name:            "TODO policy",
		Summary:         "Reports task comments that violate configured metadata policy",
		Explanation:     "Configured tags identify task comments at the start of comment lines. Metadata uses the form TAG(owner, YYYY-MM-DD, ISSUE-123): description. Policies can allow owners, require owners, dates, or issue references, validate issue formats, and limit task age.",
		Category:        diagnostic.CategoryRestriction,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"comments", "documentation", "policy", "todo"},
		Options: []lint.Option{
			{Name: "tags", Summary: "Task comment tags", Type: lint.OptionStringList, Default: defaultTodoTags, Validate: validateTodoTags},
			{Name: "allowed-owners", Summary: "Owners permitted in task metadata", Type: lint.OptionStringList, Default: []string{}, Validate: validateTodoNames},
			{Name: "require-owner", Summary: "Require an owner", Type: lint.OptionBoolean, Default: false},
			{Name: "require-date", Summary: "Require an ISO date", Type: lint.OptionBoolean, Default: false},
			{Name: "require-issue", Summary: "Require an issue reference", Type: lint.OptionBoolean, Default: false},
			{Name: "issue-pattern", Summary: "Issue reference regular expression", Type: lint.OptionString, Default: defaultTodoIssuePattern, Validate: validateTodoPattern},
			{Name: "maximum-age-days", Summary: "Maximum task age; zero disables age checks", Type: lint.OptionInteger, Default: int64(0), Minimum: 0, Maximum: 36500, HasMinimum: true, HasMaximum: true},
		},
	}
}

var todoName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
var todoDate = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

func validateTodoNames(value any) error {
	values, _ := value.([]string)
	for _, item := range values {
		if !todoName.MatchString(item) {
			return fmt.Errorf("entries must contain only letters, digits, underscores, or hyphens")
		}
	}
	return nil
}

func validateTodoTags(value any) error {
	values, _ := value.([]string)
	if len(values) == 0 {
		return fmt.Errorf("must contain at least one entry")
	}
	return validateTodoNames(value)
}

func validateTodoPattern(value any) error {
	pattern, _ := value.(string)
	if pattern == "" {
		return fmt.Errorf("must be non-empty")
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("must be a valid regular expression: %w", err)
	}
	return nil
}

type todoOptions struct {
	tags           []string
	owners         map[string]bool
	requireOwner   bool
	requireDate    bool
	requireIssue   bool
	issuePattern   *regexp.Regexp
	maximumAgeDays int
}

type todoAnnotation struct {
	tag         string
	owner       string
	dateText    string
	date        time.Time
	invalidDate bool
	issue       string
	rangeValue  source.Range
}

func (TodoPolicy) Run(ctx *lint.Context) {
	options := configuredTodoOptions(ctx)
	if len(options.owners) == 0 && !options.requireOwner && !options.requireDate && !options.requireIssue && options.maximumAgeDays == 0 {
		return
	}
	for _, annotation := range todoAnnotations(ctx, options) {
		message := todoViolation(annotation, options, time.Now().UTC())
		if message != "" {
			ctx.Report(diagnostic.Diagnostic{Message: message, Range: annotation.rangeValue})
		}
	}
}

func configuredTodoOptions(ctx *lint.Context) todoOptions {
	options := todoOptions{
		tags:         defaultTodoTags,
		issuePattern: compiledDefaultTodoIssuePattern,
	}
	if ctx.PerRule == nil || ctx.PerRule["todo-policy"] == nil {
		return options
	}
	values := ctx.PerRule["todo-policy"]
	if tags, ok := values["tags"].([]string); ok {
		options.tags = tags
	}
	if owners, ok := values["allowed-owners"].([]string); ok {
		if len(owners) != 0 {
			options.owners = make(map[string]bool, len(owners))
		}
		for _, owner := range owners {
			options.owners[owner] = true
		}
	}
	options.requireOwner, _ = values["require-owner"].(bool)
	options.requireDate, _ = values["require-date"].(bool)
	options.requireIssue, _ = values["require-issue"].(bool)
	if pattern, ok := values["issue-pattern"].(string); ok {
		if pattern == defaultTodoIssuePattern {
			options.issuePattern = compiledDefaultTodoIssuePattern
		} else if compiled, err := regexp.Compile(pattern); err == nil {
			options.issuePattern = compiled
		}
	}
	if age, ok := values["maximum-age-days"].(int64); ok {
		options.maximumAgeDays = int(age)
	}
	return options
}

func todoAnnotations(ctx *lint.Context, options todoOptions) []todoAnnotation {
	seen := make(map[int]bool)
	var result []todoAnnotation
	visit := func(trivia token.Trivia) {
		if trivia.Kind != token.Comment || seen[trivia.Start.Offset] {
			return
		}
		seen[trivia.Start.Offset] = true
		result = append(result, parseTodoTrivia(ctx, trivia, options)...)
	}
	for _, item := range ctx.File.Parsed.Tokens {
		for _, trivia := range item.LeadingTrivia {
			visit(trivia)
		}
		for _, trivia := range item.TrailingTrivia {
			visit(trivia)
		}
	}
	if ctx.File.Parsed.Root != nil {
		for _, trivia := range ctx.File.Parsed.Root.Leading {
			visit(trivia)
		}
		for _, trivia := range ctx.File.Parsed.Root.Trailing {
			visit(trivia)
		}
	}
	return result
}

func parseTodoTrivia(ctx *lint.Context, trivia token.Trivia, options todoOptions) []todoAnnotation {
	start := trivia.Start.Offset
	end := trivia.End.Offset
	if start < 0 || start >= len(ctx.File.Source) || end <= start || end > len(ctx.File.Source) {
		return nil
	}
	raw := string(ctx.File.Source[start:end])
	var result []todoAnnotation
	lineOffset := 0
	for _, line := range strings.Split(raw, "\n") {
		body, bodyOffset := todoCommentBody(line)
		for _, tag := range options.tags {
			if !todoTagPrefix(body, tag) {
				continue
			}
			offset := start + lineOffset + bodyOffset
			annotation := parseTodoAnnotation(body, tag, options)
			annotation.rangeValue = ctx.Walk.LineTable.Range(offset, offset+len(tag))
			result = append(result, annotation)
			break
		}
		lineOffset += len(line) + 1
	}
	return result
}

func todoCommentBody(line string) (string, int) {
	offset := len(line) - len(strings.TrimLeft(line, " \t\r"))
	line = line[offset:]
	for _, marker := range []string{"//", "/*", "*"} {
		if strings.HasPrefix(line, marker) {
			line = line[len(marker):]
			offset += len(marker)
			break
		}
	}
	spaces := len(line) - len(strings.TrimLeft(line, " \t"))
	return line[spaces:], offset + spaces
}

func todoTagPrefix(body, tag string) bool {
	if !strings.HasPrefix(body, tag) {
		return false
	}
	if len(body) == len(tag) {
		return true
	}
	switch body[len(tag)] {
	case '(', ':', ' ', '\t':
		return true
	default:
		return false
	}
}

func parseTodoAnnotation(body, tag string, options todoOptions) todoAnnotation {
	annotation := todoAnnotation{tag: tag}
	rest := strings.TrimSpace(body[len(tag):])
	if !strings.HasPrefix(rest, "(") {
		return annotation
	}
	close := strings.IndexByte(rest, ')')
	if close < 0 {
		return annotation
	}
	for _, raw := range strings.Split(rest[1:close], ",") {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		switch {
		case todoDate.MatchString(value):
			annotation.dateText = value
			parsed, err := time.Parse(time.DateOnly, value)
			if err != nil {
				annotation.invalidDate = true
			} else {
				annotation.date = parsed
			}
		case options.issuePattern.MatchString(value):
			annotation.issue = value
		case annotation.owner == "":
			annotation.owner = value
		}
	}
	return annotation
}

func todoViolation(annotation todoAnnotation, options todoOptions, now time.Time) string {
	if (options.requireOwner || len(options.owners) != 0) && annotation.owner == "" {
		return annotation.tag + " must include an owner"
	}
	if annotation.owner != "" && len(options.owners) != 0 && !options.owners[annotation.owner] {
		return fmt.Sprintf("%s owner %q is not allowed", annotation.tag, annotation.owner)
	}
	if annotation.invalidDate {
		return fmt.Sprintf("%s has invalid date %q", annotation.tag, annotation.dateText)
	}
	if (options.requireDate || options.maximumAgeDays > 0) && annotation.dateText == "" {
		return annotation.tag + " must include an ISO date"
	}
	if options.requireIssue && annotation.issue == "" {
		return fmt.Sprintf("%s must include an issue matching %q", annotation.tag, options.issuePattern.String())
	}
	if options.maximumAgeDays > 0 && !annotation.date.IsZero() {
		age := int(now.Sub(annotation.date).Hours() / 24)
		if age > options.maximumAgeDays {
			return fmt.Sprintf("%s date %q exceeds the maximum age of %d days", annotation.tag, annotation.dateText, options.maximumAgeDays)
		}
	}
	return ""
}
