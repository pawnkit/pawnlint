package docgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/rules"
)

func Generate(outDir string) error {
	reg := rules.Default()
	metas := reg.Sorted()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := writeIndex(outDir, metas); err != nil {
		return err
	}
	for _, m := range metas {
		if err := writeRulePage(outDir, m); err != nil {
			return err
		}
	}
	return nil
}

func writeIndex(dir string, metas []lint.Metadata) error {
	var b strings.Builder
	b.WriteString("# Rule index\n\n")
	b.WriteString("Generated from rule metadata. Do not edit by hand.\n\n")
	b.WriteString("Rules are stable unless their page marks them as preview.\n\n")
	b.WriteString("| ID | Category | Severity | Default | Fixable | Summary |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, m := range metas {
		def := "off"
		if m.DefaultEnabled {
			def = "on"
		}
		fix := "no"
		if m.Fixable {
			fix = "yes"
		}
		b.WriteString(fmt.Sprintf("| [`%s`](%s.md) | %s | %s | %s | %s | %s |\n",
			m.ID, m.ID, m.Category, m.DefaultSeverity, def, fix, escapeMD(m.Summary)))
	}
	return os.WriteFile(filepath.Join(dir, "index.md"), []byte(b.String()), 0o644)
}

func writeRulePage(dir string, m lint.Metadata) error {
	var b strings.Builder
	b.WriteString("# " + m.ID + "\n\n")
	b.WriteString(m.Summary + "\n\n")
	b.WriteString("| | |\n| --- | --- |\n")
	b.WriteString(fmt.Sprintf("| Category | %s |\n", m.Category))
	b.WriteString(fmt.Sprintf("| Severity | %s |\n", m.DefaultSeverity))
	b.WriteString(fmt.Sprintf("| Analysis | %s |\n", analysisName(m.AnalysisLevel)))
	if m.Stability == lint.StabilityPreview {
		b.WriteString("| Stability | preview |\n")
	}
	b.WriteString(fmt.Sprintf("| Default | %s |\n", onOff(m.DefaultEnabled)))
	b.WriteString(fmt.Sprintf("| Fixable | %s |\n", yesNo(m.Fixable)))
	if len(m.Tags) > 0 {
		b.WriteString(fmt.Sprintf("| Tags | %s |\n", strings.Join(m.Tags, ", ")))
	}
	b.WriteString("\n## Details\n\n")
	b.WriteString(m.Explanation + "\n")
	if len(m.Options) > 0 {
		b.WriteString("\n## Options\n\n")
		b.WriteString("| Name | Type | Default | Constraint | Description |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for _, option := range m.Options {
			b.WriteString(fmt.Sprintf("| `%s` | %s | `%v` | %s | %s |\n",
				option.Name, option.Type, option.Default, optionConstraint(option), escapeMD(option.Summary)))
		}
	}
	return os.WriteFile(filepath.Join(dir, m.ID+".md"), []byte(b.String()), 0o644)
}

func optionConstraint(option lint.Option) string {
	var parts []string
	if option.HasMinimum {
		parts = append(parts, fmt.Sprintf("minimum %d", option.Minimum))
	}
	if option.HasMaximum {
		parts = append(parts, fmt.Sprintf("maximum %d", option.Maximum))
	}
	if len(option.Choices) > 0 {
		parts = append(parts, strings.Join(option.Choices, ", "))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "; ")
}

func analysisName(l lint.AnalysisLevel) string {
	switch l {
	case lint.SyntaxAnalysis:
		return "syntax"
	case lint.SemanticAnalysis:
		return "semantic"
	case lint.ControlFlowAnalysis:
		return "control-flow"
	case lint.ProjectAnalysis:
		return "project"
	default:
		return "unknown"
	}
}

func onOff(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func escapeMD(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

func Run(args []string) int {
	out := "docs/rules"
	if len(args) > 1 && args[0] == "-out" {
		out = args[1]
	}
	if err := Generate(out); err != nil {
		fmt.Fprintln(os.Stderr, "docgen:", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "wrote", out)
	return 0
}
