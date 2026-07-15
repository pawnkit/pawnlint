package lint

import (
	"fmt"

	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint/suppress"
)

func (e *Engine) normalizeSuppressionAliases(directives []suppress.Directive, lines *source.LineTable) ([]suppress.Directive, []diagnostic.Diagnostic) {
	if e.Reg == nil || len(e.Reg.aliases) == 0 {
		return directives, nil
	}
	result := append([]suppress.Directive(nil), directives...)
	var migrations []diagnostic.Diagnostic
	for index := range result {
		ids := make([]string, 0, len(result[index].IDs))
		seen := make(map[string]struct{}, len(result[index].IDs))
		for _, configuredID := range result[index].IDs {
			id, deprecated, known := e.Reg.ResolveID(configuredID)
			if !known {
				id = configuredID
			}
			if _, duplicate := seen[id]; !duplicate {
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
			if deprecated {
				migrations = append(migrations, diagnostic.Diagnostic{
					RuleID:   DeprecatedRuleID,
					Severity: diagnostic.SeverityWarning,
					Category: diagnostic.CategoryMaintainability,
					Message:  fmt.Sprintf("suppression rule ID %q is deprecated; use %q", configuredID, id),
					Filename: result[index].File,
					Range:    lines.Range(result[index].Offset, result[index].End),
				})
			}
		}
		result[index].IDs = ids
	}
	return result, migrations
}
