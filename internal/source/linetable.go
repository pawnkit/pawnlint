package source

import "strings"

type LineTable struct {
	source    []byte
	lineStart []int
}

func NewLineTable(source []byte) *LineTable {
	lt := &LineTable{source: source, lineStart: []int{0}}
	for i, b := range source {
		if b == '\n' {
			lt.lineStart = append(lt.lineStart, i+1)
		}
	}
	return lt
}

func (lt *LineTable) LineCount() int {
	return len(lt.lineStart)
}

func (lt *LineTable) Lookup(offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(lt.source) {
		offset = len(lt.source)
	}
	line := searchLine(lt.lineStart, offset)
	lineStart := lt.lineStart[line]
	col := offset - lineStart + 1
	return Position{Offset: offset, Line: line + 1, Col: col}
}

func (lt *LineTable) Range(start, end int) Range {
	if start < 0 {
		start = 0
	}
	if end > len(lt.source) {
		end = len(lt.source)
	}
	if end < start {
		end = start
	}
	return Range{Start: lt.Lookup(start), End: lt.Lookup(end)}
}

func (lt *LineTable) LineText(lineIdx int) string {
	if lineIdx < 1 || lineIdx > len(lt.lineStart) {
		return ""
	}
	start := lt.lineStart[lineIdx-1]
	end := len(lt.source)
	if lineIdx < len(lt.lineStart) {
		end = lt.lineStart[lineIdx] - 1
		if end > 0 && lt.source[end-1] == '\r' {
			end--
		}
	}
	return string(lt.source[start:end])
}

func (lt *LineTable) DisplayWidth(start, end int) int {
	if start < 0 {
		start = 0
	}
	if end < start {
		return 0
	}
	pos := lt.Lookup(start)
	lineStart := lt.lineStart[pos.Line-1]
	lineEnd := len(lt.source)
	if pos.Line < len(lt.lineStart) {
		lineEnd = lt.lineStart[pos.Line] - 1
	}
	if end > lineEnd {
		end = lineEnd
	}
	if end <= lineStart {
		return 0
	}
	return end - start
}

func searchLine(lineStart []int, offset int) int {
	lo, hi := 0, len(lineStart)-1
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if lineStart[mid] <= offset {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo > 0 && lineStart[lo] > offset {
		lo--
	}
	lo = clamp(lo, 0, len(lineStart)-1)
	return lo
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func SplitLines(source string) []string {
	lines := strings.SplitAfter(source, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSuffix(l, "\n")
		l = strings.TrimSuffix(l, "\r")
		out = append(out, l)
	}
	return out
}
