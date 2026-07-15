package suppress

import (
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

type Kind uint8

const (
	KindDisableNextLine Kind = iota

	KindDisable

	KindEnable
)

type Directive struct {
	File      string
	Line      int
	Offset    int
	End       int
	Kind      Kind
	IDs       []string
	All       bool
	Reason    string
	Malformed bool
}

func (d Directive) MatchesRule(ruleID string) bool {
	if d.All {
		return true
	}
	for _, id := range d.IDs {
		if id == ruleID {
			return true
		}
	}
	return false
}

func FromFile(path string, src []byte, f *parser.File) []Directive {
	if f == nil {
		return nil
	}
	seen := make(map[int]bool)
	var out []Directive
	visit := func(tok token.Token) {
		for _, tr := range tok.LeadingTrivia {
			if tr.Kind != token.Comment || seen[tr.Start.Offset] {
				continue
			}
			seen[tr.Start.Offset] = true
			out = append(out, parseTrivia(path, src, tr)...)
		}
		for _, tr := range tok.TrailingTrivia {
			if tr.Kind != token.Comment || seen[tr.Start.Offset] {
				continue
			}
			seen[tr.Start.Offset] = true
			out = append(out, parseTrivia(path, src, tr)...)
		}
	}
	for _, tok := range f.Tokens {
		visit(tok)
	}
	if f.Root == nil {
		SortDirectives(out)
		return out
	}
	for _, tr := range f.Root.Leading {
		if tr.Kind != token.Comment || seen[tr.Start.Offset] {
			continue
		}
		seen[tr.Start.Offset] = true
		out = append(out, parseTrivia(path, src, tr)...)
	}
	for _, tr := range f.Root.Trailing {
		if tr.Kind != token.Comment || seen[tr.Start.Offset] {
			continue
		}
		seen[tr.Start.Offset] = true
		out = append(out, parseTrivia(path, src, tr)...)
	}
	SortDirectives(out)
	return out
}

const marker = "pawnlint-"

func parseTrivia(path string, src []byte, tr token.Trivia) []Directive {
	if tr.Kind != token.Comment {
		return nil
	}
	offs := tr.Start.Offset
	if offs < 0 || offs >= len(src) {
		return nil
	}
	line := tr.Start.Line
	end := tr.End.Offset
	if end > len(src) {
		end = len(src)
	}
	raw := string(src[offs:end])
	matches := extractFromComment(raw)
	var dirs []Directive
	for _, m := range matches {
		d, ok := buildDirective(path, line, offs, end, m)
		if ok {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

type match struct {
	action  string
	idsPart string
}

func extractFromComment(comment string) []match {
	var ms []match
	for _, line := range strings.Split(comment, "\n") {
		body := stripCommentSyntax(line)
		idx := strings.Index(body, marker)
		if idx < 0 {
			continue
		}
		rest := body[idx+len(marker):]
		var action, ids string
		if i := strings.IndexByte(rest, ' '); i >= 0 {
			action, ids = rest[:i], rest[i+1:]
		} else {
			action = rest
		}
		action = strings.TrimSpace(action)
		if !strings.HasPrefix(action, "disable") && !strings.HasPrefix(action, "enable") {
			continue
		}
		ms = append(ms, match{action: action, idsPart: ids})
	}
	return ms
}

func stripCommentSyntax(line string) string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "//")
	s = strings.TrimPrefix(s, "/*")
	s = strings.TrimSuffix(s, "*/")
	return strings.TrimSpace(s)
}

func buildDirective(path string, line, startOff, endOff int, m match) (Directive, bool) {
	var kind Kind
	switch {
	case m.action == "disable-next-line":
		kind = KindDisableNextLine
	case m.action == "disable":
		kind = KindDisable
	case m.action == "enable":
		kind = KindEnable
	default:
		return Directive{}, false
	}
	ids, all, reason, malformed := parseIDs(m.idsPart)
	d := Directive{
		File:      path,
		Line:      line,
		Offset:    startOff,
		End:       endOff,
		Kind:      kind,
		IDs:       ids,
		All:       all,
		Reason:    reason,
		Malformed: malformed,
	}
	return d, true
}

func parseIDs(rest string) (ids []string, all bool, reason string, malformed bool) {
	body := strings.TrimSpace(rest)
	if body == "" {
		return nil, false, "", true
	}
	if body == "--" {
		body = ""
	} else if strings.HasPrefix(body, "-- ") {
		reason = strings.TrimSpace(body[3:])
		body = ""
	} else if i := strings.Index(body, " -- "); i >= 0 {
		reason = strings.TrimSpace(body[i+4:])
		body = strings.TrimSpace(body[:i])
	}
	if body == "" {
		return nil, false, reason, true
	}
	for _, id := range strings.Split(body, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if id == "all" {
			all = true
			continue
		}
		ids = append(ids, id)
	}
	if !all && len(ids) == 0 {
		return nil, false, reason, true
	}
	return ids, all, reason, false
}

func IsMultiline(c string) bool {
	return strings.Count(c, "\n") > 0 || strings.HasPrefix(strings.TrimSpace(c), "/*")
}

var _ = IsMultiline
