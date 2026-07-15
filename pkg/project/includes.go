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

func (m *Model) addFile(path string, source []byte, provided bool, defines []string) (*File, error) {
	canonical, err := canonicalPath(path, m.options.WorkingDir)
	if err != nil {
		return nil, err
	}
	instance := contextKey(canonical, defines)
	if existing := m.byContext[instance]; existing != nil {
		existing.Provided = existing.Provided || provided
		if provided {
			m.byCanonical[canonical] = existing
		}
		return existing, nil
	}
	physical := m.physical[canonical]
	if physical == nil {
		physical = &physicalFile{source: source, parsed: parser.Parse(source)}
		m.physical[canonical] = physical
	}
	parsed := physical.parsed
	if parsed == nil {
		return nil, fmt.Errorf("project: parse %s", path)
	}
	display := path
	if !provided {
		display = canonical
	}
	tree := walk.NewWithDefineContext(display, parsed, defines, nil, m.options.DefinesComplete)
	file := &File{Path: display, Source: physical.source, Parsed: parsed, Walk: tree, Provided: provided, canonical: canonical, instance: instance, defines: normalizeDefines(defines), complete: m.options.DefinesComplete}
	m.Files = append(m.Files, file)
	m.byContext[instance] = file
	if m.byCanonical[canonical] == nil || provided {
		m.byCanonical[canonical] = file
	}
	return file, nil
}

func (m *Model) resolveFileIncludes(file *File) error {
	if file == nil || file.resolved || file.resolving {
		return nil
	}
	file.resolving = true
	defer func() { file.resolving = false }()
	var nodes []*parser.Node
	for _, kind := range []parser.Kind{parser.KindDirectiveInclude, parser.KindDirectiveTryInclude} {
		nodes = append(nodes, file.Walk.OfKind(kind)...)
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].Start < nodes[j].Start })
	var snapshots []walk.DefineSnapshot
	dirty := false
	for _, node := range nodes {
		if dirty {
			file.rebuildWalk(snapshots)
			dirty = false
		}
		if file.Walk.Inactive(node) {
			continue
		}
		path := includePath(file.Walk.Text(node.Field("path")))
		include := &Include{Node: node, Path: path, Optional: node.Kind == parser.KindDirectiveTryInclude, Uncertain: file.Walk.Uncertain(node)}
		file.Includes = append(file.Includes, include)
		if path == "" || include.Uncertain {
			continue
		}
		defines := file.Walk.KnownDefinesAt(node.Start)
		resolved, err := m.resolveInclude(file, path, defines)
		if err != nil {
			return err
		}
		include.Resolved = resolved
		if resolved == nil {
			continue
		}
		if err := m.resolveFileIncludes(resolved); err != nil {
			return err
		}
		if len(resolved.final) > 0 && !sameDefines(defines, resolved.final) {
			snapshots = append(snapshots, walk.DefineSnapshot{Offset: node.End, Defines: resolved.final})
			dirty = true
		}
	}
	if dirty {
		file.rebuildWalk(snapshots)
	}
	file.final = file.Walk.KnownDefinesAt(len(file.Source) + 1)
	file.Semantic = semantic.Build(file.Parsed, file.Walk)
	file.resolved = true
	return nil
}

func (m *Model) resolveInclude(from *File, path string, defines []string) (*File, error) {
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
			if existing := m.byContext[contextKey(canonical, defines)]; existing != nil {
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
			return m.addFile(canonical, source, false, defines)
		}
	}
	return nil, nil
}

func (f *File) rebuildWalk(snapshots []walk.DefineSnapshot) {
	f.Walk = walk.NewWithDefineContext(f.Path, f.Parsed, f.defines, snapshots, f.complete)
}

func contextKey(canonical string, defines []string) string {
	return canonical + "\x00" + strings.Join(normalizeDefines(defines), "\x00")
}

func normalizeDefines(defines []string) []string {
	seen := make(map[string]struct{}, len(defines))
	normalized := make([]string, 0, len(defines))
	for _, define := range defines {
		if define == "" {
			continue
		}
		if _, exists := seen[define]; exists {
			continue
		}
		seen[define] = struct{}{}
		normalized = append(normalized, define)
	}
	sort.Strings(normalized)
	return normalized
}

func sameDefines(left, right []string) bool {
	left = normalizeDefines(left)
	right = normalizeDefines(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
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
