package suppress

import "sort"

type Matcher struct {
	Directives []Directive
}

func NewMatcher(directives []Directive) *Matcher {
	return &Matcher{Directives: directives}
}

type Diagnostic struct {
	RuleID string
	Line   int
}

func (m *Matcher) IsSuppressed(used []bool, ruleID string, diagLine int) bool {
	for i, d := range m.Directives {
		if d.Kind != KindDisableNextLine {
			continue
		}
		if !d.MatchesRule(ruleID) {
			continue
		}
		if d.Line+1 == diagLine {
			if used != nil {
				used[i] = true
			}
			return true
		}
	}
	return m.blockSuppresses(used, ruleID, diagLine)
}

func (m *Matcher) blockSuppresses(used []bool, ruleID string, diagLine int) bool {
	var activeAll []int
	activeIDs := make(map[string][]int)
	for i, d := range m.Directives {
		if d.Line > diagLine {
			break
		}
		switch d.Kind {
		case KindDisable:
			if d.All {
				activeAll = append(activeAll, i)
			}
			for _, id := range d.IDs {
				activeIDs[id] = append(activeIDs[id], i)
			}
		case KindEnable:
			if d.All && len(activeAll) > 0 {
				activeAll = activeAll[:len(activeAll)-1]
			}
			for _, id := range d.IDs {
				stack := activeIDs[id]
				if len(stack) > 0 {
					activeIDs[id] = stack[:len(stack)-1]
				}
			}
		}
	}
	active := activeIDs[ruleID]
	if len(activeAll) == 0 && len(active) == 0 {
		return false
	}
	if used != nil {
		for _, i := range activeAll {
			used[i] = true
		}
		for _, i := range active {
			used[i] = true
		}
	}
	return true
}

func SortDirectives(ds []Directive) {
	sort.SliceStable(ds, func(i, j int) bool {
		if ds[i].Line != ds[j].Line {
			return ds[i].Line < ds[j].Line
		}
		return ds[i].Offset < ds[j].Offset
	})
}
