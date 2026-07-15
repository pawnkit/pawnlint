package docgen

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	b.WriteString("Each rule page documents its behavior, its configuration, and a good/bad\n")
	b.WriteString("example together. Rules are stable unless their page marks them as preview.\n\n")
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

	writeConfiguration(&b, m)
	writeExamples(&b, m.ID)

	return os.WriteFile(filepath.Join(dir, m.ID+".md"), []byte(b.String()), 0o644)
}

func writeConfiguration(b *strings.Builder, m lint.Metadata) {
	b.WriteString("\n## Configuration\n\n")
	b.WriteString(fmt.Sprintf("```toml\n[rules]\n%s = %q\n```\n", m.ID, m.DefaultSeverity.String()))

	if len(m.Options) > 0 {
		b.WriteString("\nSet options under `[rules." + m.ID + "]`.\n\n")
		b.WriteString("| Name | Type | Default | Constraint | Description |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for _, option := range m.Options {
			b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
				option.Name, option.Type, optionDefault(option), optionConstraint(option), escapeMD(option.Summary)))
		}
		for _, option := range m.Options {
			if len(option.Fields) == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("\n`%s` entry fields:\n\n", option.Name))
			b.WriteString("| Name | Type | Default | Constraint | Description |\n")
			b.WriteString("| --- | --- | --- | --- | --- |\n")
			for _, field := range option.Fields {
				b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s |\n",
					field.Name, field.Type, optionDefault(field), optionConstraint(field), escapeMD(field.Summary)))
			}
		}
	}

	if m.ConfigExample != "" {
		b.WriteString("\n### Example\n\n")
		b.WriteString("```toml\n")
		b.WriteString(strings.Trim(m.ConfigExample, "\n"))
		b.WriteString("\n```\n")
	}
}

func writeExamples(b *strings.Builder, id string) {
	bad, hasBad := readExample(id, "invalid")
	good, hasGood := readExample(id, "valid")
	if !hasBad && !hasGood {
		return
	}
	b.WriteString("\n## Examples\n\n")
	if hasBad {
		b.WriteString("### Bad\n\n")
		writeExampleSupport(b, id, "invalid")
		b.WriteString("```pawn\n" + bad + "\n```\n\n")
	}
	if hasGood {
		b.WriteString("### Good\n\n")
		writeExampleSupport(b, id, "valid")
		b.WriteString("```pawn\n" + good + "\n```\n")
	}
}

func writeExampleSupport(b *strings.Builder, id, name string) {
	content, ok := readExampleSupport(id, name)
	if !ok {
		return
	}
	b.WriteString("`example-" + name + ".inc`:\n\n")
	b.WriteString("```pawn\n" + content + "\n```\n\n")
	b.WriteString("`example-" + name + ".pwn`:\n\n")
}

const maxExampleLines = 30

func readExample(id, name string) (string, bool) {
	var content []byte
	for _, filename := range []string{"example-" + name + ".pwn", name + ".pwn"} {
		path := filepath.Join(testdataRulesDir(), id, filename)
		current, err := os.ReadFile(path)
		if err == nil {
			content = current
			break
		}
	}
	if content == nil {
		return "", false
	}
	text := strings.Trim(string(content), "\n")
	if text == "" {
		return "", false
	}
	lines := strings.Split(text, "\n")
	if len(lines) > maxExampleLines {
		cut := maxExampleLines
		for i := maxExampleLines - 1; i > 0; i-- {
			if lines[i] == "}" {
				cut = i + 1
				break
			}
		}
		lines = append(lines[:cut], "// …")
	}
	return strings.Join(lines, "\n"), true
}

func readExampleSupport(id, name string) (string, bool) {
	content, err := os.ReadFile(filepath.Join(testdataRulesDir(), id, "example-"+name+".inc"))
	if err != nil {
		return "", false
	}
	text := strings.Trim(string(content), "\n")
	return text, text != ""
}

func testdataRulesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "rules")
}

func optionDefault(option lint.Option) string {
	if option.Default == nil {
		return "—"
	}
	return fmt.Sprintf("`%v`", option.Default)
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
