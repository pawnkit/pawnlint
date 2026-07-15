package project

import "sort"

type IncludeIssue struct {
	Owner   *File
	File    *File
	Include *Include
}

func (m *Model) MissingIncludes() []IncludeIssue {
	if m == nil {
		return nil
	}
	return append([]IncludeIssue(nil), m.missingIncludes...)
}

func (m *Model) AmbiguousIncludes() []IncludeIssue {
	if m == nil {
		return nil
	}
	return append([]IncludeIssue(nil), m.ambiguousIncludes...)
}

func (m *Model) buildIncludeIssues() {
	owners := make(map[*File]*File, len(m.Files))
	var assign func(*File, *File)
	assign = func(file, owner *File) {
		if file == nil || owners[file] != nil {
			return
		}
		owners[file] = owner
		for _, include := range file.Includes {
			if include != nil && !include.Uncertain {
				assign(include.Resolved, owner)
			}
		}
	}
	for _, unit := range m.Units {
		assign(unit.Root, unit.Root)
	}
	for _, file := range m.Files {
		if file.Provided {
			assign(file, file)
		}
	}
	missingSeen := make(map[string]struct{})
	ambiguousSeen := make(map[string]struct{})
	for _, file := range m.Files {
		owner := owners[file]
		if owner == nil {
			continue
		}
		for _, include := range file.Includes {
			if include == nil || include.Node == nil || include.Path == "" || include.Uncertain {
				continue
			}
			key := file.canonical + ":" + offsetText(include)
			if include.Resolved == nil && !include.Optional {
				if _, exists := missingSeen[key]; !exists {
					missingSeen[key] = struct{}{}
					m.missingIncludes = append(m.missingIncludes, IncludeIssue{Owner: owner, File: file, Include: include})
				}
			}
			if len(include.Candidates) > 1 {
				if _, exists := ambiguousSeen[key]; !exists {
					ambiguousSeen[key] = struct{}{}
					m.ambiguousIncludes = append(m.ambiguousIncludes, IncludeIssue{Owner: owner, File: file, Include: include})
				}
			}
		}
	}
	sortIncludeIssues(m.missingIncludes)
	sortIncludeIssues(m.ambiguousIncludes)
}

func sortIncludeIssues(issues []IncludeIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].File.canonical != issues[j].File.canonical {
			return issues[i].File.canonical < issues[j].File.canonical
		}
		return issues[i].Include.Node.Start < issues[j].Include.Node.Start
	})
}
