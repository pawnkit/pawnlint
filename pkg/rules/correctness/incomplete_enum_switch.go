package correctness

import (
	"fmt"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type IncompleteEnumSwitch struct{}

func (IncompleteEnumSwitch) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "incomplete-enum-switch",
		Name:            "Incomplete enum switch",
		Summary:         "Reports enum switches that omit named values",
		Explanation:     "A switch over a resolved enum should cover every named value or provide a default clause. Enums with custom increments and switches with unknown cases, uncertain branches, ambiguous tags, or malformed syntax are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"switch", "enums", "coverage", "project"},
	}
}

type enumSwitchValue struct {
	name  string
	value int64
	node  *parser.Node
}

type enumSwitchDefinition struct {
	name   string
	file   *project.File
	node   *parser.Node
	values []enumSwitchValue
}

func (IncompleteEnumSwitch) Run(ctx *lint.Context) {
	if ctx.Project == nil || ctx.Semantic == nil {
		return
	}
	file := ctx.Project.File(ctx.File.Path)
	if !ctx.Project.InProgram(file) {
		return
	}
	for _, statement := range ctx.Walk.OfKind(parser.KindSwitchStatement) {
		condition := statement.Field("condition")
		if condition == nil || statement.HasError || ctx.Walk.Inactive(statement) || ctx.Walk.Uncertain(statement) || enumSwitchHasDefault(statement) {
			continue
		}
		tag, ok := ctx.ExpressionTag(condition)
		if !ok || tag == "" || tag == "_" || tag == "Float" || tag == "bool" {
			continue
		}
		definition, ok := resolvedEnumSwitchDefinition(ctx.Project, file, tag)
		if !ok || len(definition.values) == 0 {
			continue
		}
		covered, ok := coveredEnumSwitchValues(ctx, file, statement, definition)
		if !ok {
			continue
		}
		var missing []string
		for _, value := range definition.values {
			if !covered[value.value] {
				missing = append(missing, value.name)
			}
		}
		if len(missing) == 0 {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  incompleteEnumSwitchMessage(definition.name, missing),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(condition),
		})
	}
}

func enumSwitchHasDefault(statement *parser.Node) bool {
	for _, child := range statement.Children {
		if child.Kind == parser.KindDefaultClause {
			return true
		}
	}
	return false
}

func resolvedEnumSwitchDefinition(model *project.Model, from *project.File, tag string) (enumSwitchDefinition, bool) {
	seen := make(map[*parser.Node]enumSwitchDefinition)
	for _, unit := range model.Units {
		contains := false
		for _, file := range unit.Files {
			contains = contains || file == from
		}
		if !contains {
			continue
		}
		for _, file := range unit.Files {
			for _, declaration := range file.Walk.OfKind(parser.KindEnumDeclaration) {
				if enumSwitchTypeName(file, declaration) != tag || declaration.HasError || file.Walk.Inactive(declaration) || file.Walk.Uncertain(declaration) {
					continue
				}
				definition, ok := buildEnumSwitchDefinition(model, file, declaration, tag)
				if ok {
					seen[declaration] = definition
				}
			}
		}
	}
	if len(seen) != 1 {
		return enumSwitchDefinition{}, false
	}
	for _, definition := range seen {
		return definition, true
	}
	return enumSwitchDefinition{}, false
}

func enumSwitchTypeName(file *project.File, declaration *parser.Node) string {
	if tag := declaration.Field("tag"); tag != nil {
		if tag.Tok.Kind == token.Identifier {
			return tag.Tok.Text(file.Source)
		}
		for _, child := range tag.Children {
			if child.Kind == parser.KindIdentifier {
				return file.Walk.Text(child)
			}
		}
	}
	return file.Walk.Text(declaration.Field("name"))
}

func buildEnumSwitchDefinition(model *project.Model, file *project.File, declaration *parser.Node, name string) (enumSwitchDefinition, bool) {
	if declaration.Field("increment") != nil {
		return enumSwitchDefinition{}, false
	}
	body := declaration.Field("body")
	if body == nil {
		return enumSwitchDefinition{}, false
	}
	definition := enumSwitchDefinition{name: name, file: file, node: declaration}
	current := int64(0)
	for _, entry := range body.Children {
		if entry.Kind != parser.KindEnumEntry {
			continue
		}
		if entry.HasError || file.Walk.Inactive(entry) || file.Walk.Uncertain(entry) {
			return enumSwitchDefinition{}, false
		}
		if explicit := entry.Field("value"); explicit != nil {
			value, ok := model.Eval(file, explicit)
			if !ok {
				return enumSwitchDefinition{}, false
			}
			current = value
		}
		entryName := file.Walk.Text(entry.Field("name"))
		if entryName == "" {
			return enumSwitchDefinition{}, false
		}
		definition.values = append(definition.values, enumSwitchValue{name: entryName, value: current, node: entry})
		width := int64(1)
		for _, child := range entry.Children {
			if child.Kind != parser.KindDimension {
				continue
			}
			size, ok := model.Eval(file, child.Field("size"))
			if !ok || size <= 0 {
				return enumSwitchDefinition{}, false
			}
			width = int64(int32(uint32(width * size)))
			if width <= 0 {
				return enumSwitchDefinition{}, false
			}
		}
		current = int64(int32(uint32(current + width)))
	}
	return definition, len(definition.values) != 0
}

func coveredEnumSwitchValues(ctx *lint.Context, file *project.File, statement *parser.Node, definition enumSwitchDefinition) (map[int64]bool, bool) {
	covered := make(map[int64]bool)
	entryValues := make(map[*parser.Node]int64, len(definition.values))
	for _, value := range definition.values {
		entryValues[value.node] = value.value
	}
	for _, clause := range statement.Children {
		if clause.Kind != parser.KindCaseClause {
			continue
		}
		if clause.HasError || ctx.Walk.Inactive(clause) || ctx.Walk.Uncertain(clause) {
			return nil, false
		}
		values := clause.Field("values")
		if values == nil {
			return nil, false
		}
		for _, valueNode := range values.Children {
			if valueNode.Kind == parser.KindCaseRange {
				start, startOK := enumSwitchCaseValue(ctx, file, valueNode.Field("start"), entryValues)
				end, endOK := enumSwitchCaseValue(ctx, file, valueNode.Field("end"), entryValues)
				if !startOK || !endOK || start > end {
					return nil, false
				}
				for _, expected := range definition.values {
					if expected.value >= start && expected.value <= end {
						covered[expected.value] = true
					}
				}
				continue
			}
			value, ok := enumSwitchCaseValue(ctx, file, valueNode, entryValues)
			if !ok {
				return nil, false
			}
			covered[value] = true
		}
	}
	return covered, true
}

func enumSwitchCaseValue(ctx *lint.Context, file *project.File, node *parser.Node, entries map[*parser.Node]int64) (int64, bool) {
	if node == nil || node.HasError || ctx.Walk.Uncertain(node) {
		return 0, false
	}
	if value, ok := ctx.Constant(node); ok {
		return value, true
	}
	switch node.Kind {
	case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
		return enumSwitchCaseValue(ctx, file, node.Field("expression"), entries)
	case parser.KindIdentifier:
		declaration, ok := ctx.Project.Resolve(file, node)
		if !ok {
			return 0, false
		}
		value, ok := entries[declaration.Node]
		return value, ok
	default:
		return 0, false
	}
}

func incompleteEnumSwitchMessage(name string, missing []string) string {
	shown := missing
	suffix := ""
	if len(shown) > 5 {
		shown = shown[:5]
		suffix = fmt.Sprintf(", and %d more", len(missing)-len(shown))
	}
	return fmt.Sprintf("switch on enum %q does not cover %s%s", name, strings.Join(shown, ", "), suffix)
}
