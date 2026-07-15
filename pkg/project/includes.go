package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/preprocess"
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
		var parsed *parser.File
		if m.options.ObserveTiming == nil {
			parsed = parser.Parse(source)
		} else {
			started := time.Now()
			parsed = parser.Parse(source)
			m.observe(TimingEvent{Stage: TimingParse, Duration: time.Since(started)})
		}
		physical = &physicalFile{source: source, parsed: parsed}
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
	file := &File{Path: display, Source: physical.source, Parsed: parsed, Walk: tree, Provided: provided, canonical: canonical, instance: instance, defines: append([]string(nil), defines...), complete: m.options.DefinesComplete, sourceID: uint32(len(m.Files) + 1)}
	m.Files = append(m.Files, file)
	m.sourceFiles[file.sourceID] = file
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
		resolved, candidates, err := m.resolveInclude(file, path, defines)
		if err != nil {
			return err
		}
		include.Resolved = resolved
		include.Candidates = candidates
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
	if m.options.ObserveTiming == nil {
		file.Semantic = semantic.Build(file.Parsed, file.Walk)
	} else {
		started := time.Now()
		file.Semantic = semantic.Build(file.Parsed, file.Walk)
		m.observe(TimingEvent{Stage: TimingSemantic, Duration: time.Since(started)})
	}
	started := time.Now()
	imports := make(map[int]*preprocess.State)
	for _, include := range file.Includes {
		if include.Resolved != nil && !include.Uncertain && include.Resolved.expansionState != nil {
			imports[include.Node.Start] = include.Resolved.expansionState
		}
	}
	expanded, expansionState := preprocess.ExpandWithState(file.Parsed, file.Walk, file.sourceID, nil, imports)
	file.expansionState = expansionState
	file.ExpansionComplete = expanded.Complete
	for _, include := range file.Includes {
		if include.Uncertain || include.Resolved != nil && !include.Resolved.ExpansionComplete {
			file.ExpansionComplete = false
		}
	}
	if !expanded.Changed {
		file.ExpandedSource = file.Source
		file.ExpandedParsed = file.Parsed
		file.ExpandedWalk = file.Walk
		file.ExpandedSemantic = file.Semantic
	} else {
		file.ExpandedSource = expanded.Source
		file.ExpandedParsed = expanded.Parsed
		file.ExpandedWalk = walk.NewWithDefineContext(file.Path, expanded.Parsed, file.defines, nil, file.complete)
		if !m.options.ReleaseExpanded {
			file.ExpandedSemantic = semantic.Build(expanded.Parsed, file.ExpandedWalk)
		}
	}
	m.captureRuntimeCalls(file)
	if m.options.ReleaseExpanded {
		file.ExpandedSource = nil
		file.ExpandedParsed = nil
		file.ExpandedWalk = nil
		file.ExpandedSemantic = nil
	}
	if m.options.ObserveTiming != nil {
		m.observe(TimingEvent{Stage: TimingPreprocess, Duration: time.Since(started)})
	}
	file.resolved = true
	return nil
}

func (m *Model) observe(event TimingEvent) {
	if m.options.ObserveTiming != nil {
		m.options.ObserveTiming(event)
	}
}

func (m *Model) resolveInclude(from *File, path string, defines []string) (*File, []string, error) {
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
	var candidates []string
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
			if m.physical[canonical] != nil {
				candidates = append(candidates, canonical)
				continue
			}
			info, err := os.Stat(canonical)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, nil, err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			candidates = append(candidates, canonical)
		}
	}
	if len(candidates) == 0 {
		return nil, nil, nil
	}
	chosen := candidates[0]
	if existing := m.byContext[contextKey(chosen, defines)]; existing != nil {
		return existing, candidates, nil
	}
	var source []byte
	if physical := m.physical[chosen]; physical != nil {
		source = physical.source
	} else {
		var err error
		source, err = os.ReadFile(chosen)
		if err != nil {
			return nil, nil, err
		}
	}
	resolved, err := m.addFile(chosen, source, false, defines)
	return resolved, candidates, err
}

func (f *File) rebuildWalk(snapshots []walk.DefineSnapshot) {
	f.Walk = walk.NewWithDefineContext(f.Path, f.Parsed, f.defines, snapshots, f.complete)
}

func contextKey(canonical string, defines []string) string {
	return canonical + "\x00" + strings.Join(defines, "\x00")
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
	extension := strings.ToLower(filepath.Ext(path))
	if extension == ".inc" || extension == ".pwn" {
		return []string{path}
	}
	return []string{path, path + ".inc"}
}
