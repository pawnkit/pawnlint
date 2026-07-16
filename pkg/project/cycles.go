package project

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type IncludeEdge struct {
	From    *File
	Include *Include
	To      *File
}

type IncludeCycle struct {
	Owner *File
	Edges []IncludeEdge
}

func (m *Model) IncludeCycles() []IncludeCycle {
	if m == nil {
		return nil
	}
	return append([]IncludeCycle(nil), m.includeCycles...)
}

func (m *Model) buildIncludeCycles() []IncludeCycle {
	seen := make(map[string]struct{})
	var cycles []IncludeCycle
	for _, unit := range m.Units {
		state := make(map[*File]uint8, len(unit.Files))
		positions := make(map[*File]int, len(unit.Files))
		var files []*File
		var edges []IncludeEdge
		var visit func(*File)
		visit = func(file *File) {
			state[file] = 1
			positions[file] = len(files)
			files = append(files, file)
			for _, include := range file.Includes {
				if include == nil || include.Uncertain || include.Resolved == nil {
					continue
				}
				to := include.Resolved
				edge := IncludeEdge{From: file, Include: include, To: to}
				switch state[to] {
				case 0:
					edges = append(edges, edge)
					visit(to)
					edges = edges[:len(edges)-1]
				case 1:
					start := positions[to]
					cycleEdges := append([]IncludeEdge(nil), edges[start:]...)
					cycleEdges = append(cycleEdges, edge)
					key := includeCycleKey(cycleEdges)
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						cycles = append(cycles, IncludeCycle{Owner: unit.Root, Edges: cycleEdges})
					}
				}
			}
			files = files[:len(files)-1]
			delete(positions, file)
			state[file] = 2
		}
		visit(unit.Root)
	}
	sort.SliceStable(cycles, func(i, j int) bool {
		return includeCycleKey(cycles[i].Edges) < includeCycleKey(cycles[j].Edges)
	})
	return cycles
}

func includeCycleKey(edges []IncludeEdge) string {
	if len(edges) == 0 {
		return ""
	}
	parts := make([]string, len(edges))
	for i, edge := range edges {
		parts[i] = edge.From.canonical + ":" + offsetText(edge.Include) + "->" + edge.To.canonical
	}
	best := strings.Join(parts, "\x00")
	for i := 1; i < len(parts); i++ {
		rotated := append(append([]string(nil), parts[i:]...), parts[:i]...)
		candidate := strings.Join(rotated, "\x00")
		if candidate < best {
			best = candidate
		}
	}
	return best
}

func offsetText(include *Include) string {
	if !include.Valid() {
		return "0"
	}
	return filepath.ToSlash(include.Path) + ":" + strconv.Itoa(include.Start())
}
