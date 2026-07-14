package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func (m *Model) addFile(path string, source []byte, provided bool) (*File, error) {
	canonical, err := canonicalPath(path, m.options.WorkingDir)
	if err != nil {
		return nil, err
	}
	if existing := m.byCanonical[canonical]; existing != nil {
		existing.Provided = existing.Provided || provided
		return existing, nil
	}
	parsed := parser.Parse(source)
	if parsed == nil {
		return nil, fmt.Errorf("project: parse %s", path)
	}
	display := path
	if !provided {
		display = canonical
	}
	tree := walk.NewWithDefines(display, parsed, m.options.Defines)
	file := &File{Path: display, Source: source, Parsed: parsed, Walk: tree, Provided: provided, canonical: canonical}
	file.Semantic = semantic.Build(parsed, tree)
	m.Files = append(m.Files, file)
	m.byCanonical[canonical] = file
	return file, nil
}

func (m *Model) resolveFileIncludes(file *File) error {
	for _, kind := range []parser.Kind{parser.KindDirectiveInclude, parser.KindDirectiveTryInclude} {
		for _, node := range file.Walk.OfKind(kind) {
			if file.Walk.Inactive(node) {
				continue
			}
			path := includePath(file.Walk.Text(node.Field("path")))
			include := &Include{Node: node, Path: path, Optional: kind == parser.KindDirectiveTryInclude, Uncertain: file.Walk.Uncertain(node)}
			file.Includes = append(file.Includes, include)
			if path == "" || include.Uncertain {
				continue
			}
			resolved, err := m.resolveInclude(file, path)
			if err != nil {
				return err
			}
			include.Resolved = resolved
		}
	}
	sort.SliceStable(file.Includes, func(i, j int) bool { return file.Includes[i].Node.Start < file.Includes[j].Node.Start })
	return nil
}

func (m *Model) resolveInclude(from *File, path string) (*File, error) {
	path = filepath.FromSlash(strings.ReplaceAll(path, "\\", "/"))
	var bases []string
	if filepath.IsAbs(path) {
		bases = []string{""}
	} else {
		bases = append(bases, filepath.Dir(from.canonical))
		bases = append(bases, m.options.IncludePaths...)
		bases = append(bases, m.options.WorkingDir)
	}
	seen := make(map[string]struct{})
	for _, base := range bases {
		candidate := path
		if base != "" {
			candidate = filepath.Join(base, path)
		}
		for _, name := range includeCandidates(candidate) {
			canonical, err := canonicalPath(name, m.options.WorkingDir)
			if err != nil {
				continue
			}
			if _, tried := seen[canonical]; tried {
				continue
			}
			seen[canonical] = struct{}{}
			if existing := m.byCanonical[canonical]; existing != nil {
				return existing, nil
			}
			info, err := os.Stat(canonical)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			source, err := os.ReadFile(canonical)
			if err != nil {
				return nil, err
			}
			return m.addFile(canonical, source, false)
		}
	}
	return nil, nil
}

func includePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && raw[0] == '<' && raw[len(raw)-1] == '>' {
		return strings.TrimSpace(raw[1 : len(raw)-1])
	}
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		if value, err := strconv.Unquote(raw); err == nil {
			return value
		}
		return raw[1 : len(raw)-1]
	}
	return raw
}

func includeCandidates(path string) []string {
	if filepath.Ext(path) != "" {
		return []string{path}
	}
	return []string{path, path + ".inc"}
}
