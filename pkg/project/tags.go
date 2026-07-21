package project

import "strings"

func (f *File) buildTagAliases() {
	if f == nil || f.expansionState == nil {
		return
	}
	aliases := make(map[string][]string)
	add := func(tags []string) {
		for _, tag := range tags {
			if tag == "" {
				continue
			}
			expanded, ok := f.expansionState.ExpandIdentifier(tag)
			if ok {
				aliases[tag] = expandedTags(expanded)
			}
		}
	}
	if f.Semantic != nil {
		for _, symbol := range f.Semantic.Symbols {
			add(symbol.Tags)
		}
	} else if f.CompactSemantic != nil {
		for _, symbol := range f.CompactSemantic.Symbols {
			add(symbol.Tags)
		}
	}
	if len(aliases) != 0 {
		f.tagAliases = aliases
	}
}

func (f *File) normalizeTags(tags []string) []string {
	if f == nil {
		return append([]string(nil), tags...)
	}
	var result []string
	for _, tag := range tags {
		if aliases := f.tagAliases[tag]; len(aliases) != 0 {
			result = append(result, aliases...)
		} else {
			result = append(result, tag)
		}
	}
	return result
}

func expandedTags(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		value = value[1 : len(value)-1]
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if tag := strings.TrimSpace(part); tag != "" {
			result = append(result, tag)
		}
	}
	return result
}
