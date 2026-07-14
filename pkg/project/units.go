package project

import (
	"path/filepath"
	"strings"
)

func (m *Model) buildUnits() {
	var roots []*File
	for _, file := range m.Files {
		if file.Provided && strings.EqualFold(filepath.Ext(file.Path), ".pwn") {
			roots = append(roots, file)
		}
	}
	if len(roots) == 0 {
		for _, file := range m.Files {
			if file.Provided {
				roots = append(roots, file)
			}
		}
	}
	for _, root := range roots {
		visited := make(map[*File]bool)
		var files []*File
		var visit func(*File)
		visit = func(file *File) {
			if file == nil || visited[file] {
				return
			}
			visited[file] = true
			files = append(files, file)
			for _, include := range file.Includes {
				if include.Uncertain {
					continue
				}
				visit(include.Resolved)
			}
		}
		visit(root)
		members := make(map[*File]struct{}, len(files))
		for _, file := range files {
			members[file] = struct{}{}
		}
		m.Units = append(m.Units, &Unit{Root: root, Files: files, members: members})
	}
}
