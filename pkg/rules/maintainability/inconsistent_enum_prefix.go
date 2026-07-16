package maintainability

import (
	"fmt"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type InconsistentEnumPrefix struct{}

func (InconsistentEnumPrefix) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "inconsistent-enum-prefix",
		Name:            "Inconsistent enum prefix",
		Summary:         "Reports enum entries that omit a dominant member prefix",
		Explanation:     "A named enum with a dominant member prefix is easier to scan and less likely to contribute inconsistent global names. The rule uses the first underscore or case boundary and requires at least four definite entries. A prefix must appear on at least three entries and 75 percent of the enum.",
		Category:        diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"naming", "style", "enums"},
	}
}

func (InconsistentEnumPrefix) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	enums := enumPrefixEntries(ctx)
	for _, declaration := range ctx.Walk.OfKind(parser.KindEnumDeclaration) {
		members := enums[declaration]
		body := declaration.Field("body")
		if declaration.Field("name") == nil || declaration.HasError || body == nil || body.HasError || ctx.Walk.Uncertain(declaration) || ctx.Walk.Inactive(declaration) || members == nil || members.uncertain {
			continue
		}
		if len(members.entries) < 4 {
			continue
		}
		prefix := dominantEnumPrefix(members.entries)
		if prefix == "" {
			continue
		}
		var example *semantic.Symbol
		for _, member := range members.entries {
			if strings.HasPrefix(member.Name, prefix) {
				example = member
				break
			}
		}
		for _, member := range members.entries {
			if strings.HasPrefix(member.Name, prefix) {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("enum-entry name %q should use the dominant prefix %q", member.Name, prefix),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(member.NameNode),
				Notes: []diagnostic.RelatedLocation{{
					Range:   ctx.Walk.Range(example.NameNode),
					Message: "prefixed enum entry is here",
				}},
			})
		}
	}
}

type enumPrefixMembers struct {
	entries   []*semantic.Symbol
	uncertain bool
}

func enumPrefixEntries(ctx *lint.Context) map[*parser.Node]*enumPrefixMembers {
	symbols := make(map[*parser.Node]*semantic.Symbol)
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind == semantic.SymbolEnumEntry && symbol.Decl != nil {
			symbols[symbol.Decl] = symbol
		}
	}
	result := make(map[*parser.Node]*enumPrefixMembers)
	for _, entry := range ctx.Walk.OfKind(parser.KindEnumEntry) {
		if ctx.Walk.Inactive(entry) {
			continue
		}
		var declaration *parser.Node
		for ancestor := ctx.Walk.Parent(entry); ancestor != nil; ancestor = ctx.Walk.Parent(ancestor) {
			if ancestor.Kind == parser.KindEnumDeclaration {
				declaration = ancestor
				break
			}
		}
		if declaration == nil {
			continue
		}
		if result[declaration] == nil {
			result[declaration] = &enumPrefixMembers{}
		}
		group := result[declaration]
		symbol := symbols[entry]
		if entry.HasError || ctx.Walk.Uncertain(entry) || symbol == nil || symbol.Ambiguous || symbol.NameNode == nil {
			group.uncertain = true
			continue
		}
		group.entries = append(group.entries, symbol)
	}
	return result
}

func dominantEnumPrefix(entries []*semantic.Symbol) string {
	seen := make(map[string]bool)
	var candidates []string
	for _, entry := range entries {
		for _, prefix := range enumPrefixCandidates(entry.Name) {
			if !seen[prefix] {
				seen[prefix] = true
				candidates = append(candidates, prefix)
			}
		}
	}
	best := ""
	bestCount := 0
	for _, candidate := range candidates {
		count := 0
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name, candidate) {
				count++
			}
		}
		if count == len(entries) {
			return ""
		}
		if count < 3 || count*4 < len(entries)*3 {
			continue
		}
		if count > bestCount || count == bestCount && len(candidate) > len(best) {
			best = candidate
			bestCount = count
		}
	}
	return best
}

func enumPrefixCandidates(name string) []string {
	for index := 1; index < len(name); index++ {
		if name[index-1] == '_' || asciiUpper(name[index]) && !asciiUpper(name[index-1]) {
			return []string{name[:index]}
		}
	}
	return nil
}

func asciiUpper(value byte) bool {
	return value >= 'A' && value <= 'Z'
}
