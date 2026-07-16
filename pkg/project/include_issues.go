package project

import "sort"

type IncludeIssue struct {
	Owner   *File
	File    *File
	Include *Include
}

func (m *Model) Includes() []IncludeIssue {
	if m == nil {
		return nil
	}
	return append([]IncludeIssue(nil), m.includeDirectives...)
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

func (m *Model) DuplicateIncludes() []IncludeIssue {
	if m == nil {
		return nil
	}
	return append([]IncludeIssue(nil), m.duplicateIncludes...)
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
	duplicateSeen := make(map[string]struct{})
	for _, file := range m.Files {
		owner := owners[file]
		if owner == nil {
			continue
		}
		resolved := make(map[string]struct{})
		for _, include := range file.Includes {
			if !include.Valid() || include.Path == "" || include.Uncertain {
				continue
			}
			key := file.canonical + ":" + offsetText(include)
			m.includeDirectives = append(m.includeDirectives, IncludeIssue{Owner: owner, File: file, Include: include})
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
			if include.Resolved != nil {
				target := include.Resolved.canonical
				if _, exists := resolved[target]; exists {
					if _, reported := duplicateSeen[key]; !reported {
						duplicateSeen[key] = struct{}{}
						m.duplicateIncludes = append(m.duplicateIncludes, IncludeIssue{Owner: owner, File: file, Include: include})
					}
				} else {
					resolved[target] = struct{}{}
				}
			}
		}
	}
	sortIncludeIssues(m.missingIncludes)
	sortIncludeIssues(m.ambiguousIncludes)
	sortIncludeIssues(m.duplicateIncludes)
	sortIncludeIssues(m.includeDirectives)
}

func sortIncludeIssues(issues []IncludeIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].File.canonical != issues[j].File.canonical {
			return issues[i].File.canonical < issues[j].File.canonical
		}
		return issues[i].Include.Start() < issues[j].Include.Start()
	})
}
