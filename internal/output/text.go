package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

func writeText(w io.Writer, diags []diagnostic.Diagnostic, sources SourceSet, useColor bool) error {
	c := noColor
	if useColor {
		c = realColor
	}
	for _, d := range diags {
		if err := writeOneText(w, d, sources, c); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func writeOneText(w io.Writer, d diagnostic.Diagnostic, sources SourceSet, c colorizer) error {
	lt := sources.lineTableFor(d.Filename)
	pos := d.Range.Start
	if pos.Offset == 0 && pos.Line == 0 {
		pos = lt.Lookup(d.Range.Start.Offset)
	}
	header := fmt.Sprintf("%s:%d:%d: %s[%s]:",
		d.Filename, pos.Line, pos.Col, c.severity(d.Severity), d.RuleID)
	if _, err := fmt.Fprintln(w, header); err != nil {
		return err
	}
	if d.Message != "" {
		if _, err := fmt.Fprintln(w, d.Message); err != nil {
			return err
		}
	}
	if pos.Line > 0 {
		lineText := lt.LineText(pos.Line)
		if lineText != "" || d.Range.Start.Offset < d.Range.End.Offset {
			prefix := fmt.Sprintf("%d | ", pos.Line)
			if _, err := fmt.Fprintln(w, prefix+lineText); err != nil {
				return err
			}
			caret := buildCaret(lt, d.Range, len(prefix))
			if _, err := fmt.Fprintln(w, caret); err != nil {
				return err
			}
		}
	}
	for _, note := range d.Notes {
		npos := note.Range.Start
		if npos.Line == 0 {
			npos = lt.Lookup(note.Range.Start.Offset)
		}
		if _, err := fmt.Fprintf(w, "note: %s:%d:%d: %s\n", d.Filename, npos.Line, npos.Col, note.Message); err != nil {
			return err
		}
	}
	if d.Suggested != "" {
		if _, err := fmt.Fprintln(w, "help: "+d.Suggested); err != nil {
			return err
		}
	}
	if d.Fix != nil {
		if _, err := fmt.Fprintln(w, "fix: "+d.Fix.Description); err != nil {
			return err
		}
	}
	return nil
}

func buildCaret(lt *source.LineTable, r source.Range, prefixLen int) string {
	startLine := lt.Lookup(r.Start.Offset)
	width := lt.DisplayWidth(r.Start.Offset, r.End.Offset)
	if width <= 0 {
		width = 1
	}
	indent := startLine.Col - 1
	lineText := lt.LineText(startLine.Line)
	if indent > len(lineText) {
		indent = len(lineText)
	}
	pad := strings.Repeat(" ", prefixLen) + mirrorIndent(lineText[:indent])
	carets := strings.Repeat("^", width)
	return pad + carets
}

func mirrorIndent(prefix string) string {
	out := make([]byte, len(prefix))
	for i := 0; i < len(prefix); i++ {
		if prefix[i] == '\t' {
			out[i] = '\t'
		} else {
			out[i] = ' '
		}
	}
	return string(out)
}

type colorizer struct {
	severity func(s diagnostic.Severity) string
}

var noColor = colorizer{severity: func(s diagnostic.Severity) string { return s.String() }}

var realColor = colorizer{severity: func(s diagnostic.Severity) string {
	switch s {
	case diagnostic.SeverityError:
		return "\x1b[31m" + s.String() + "\x1b[0m"
	case diagnostic.SeverityWarning:
		return "\x1b[33m" + s.String() + "\x1b[0m"
	case diagnostic.SeverityInfo:
		return "\x1b[36m" + s.String() + "\x1b[0m"
	case diagnostic.SeverityHint:
		return "\x1b[35m" + s.String() + "\x1b[0m"
	default:
		return s.String()
	}
}}
