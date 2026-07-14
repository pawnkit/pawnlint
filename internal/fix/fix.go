package fix

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

type Change struct {
	Path   string
	Before []byte
	After  []byte
}

type Plan struct {
	Changes []Change
}

type plannedEdit struct {
	start int
	end   int
	text  string
}

func Build(sources map[string][]byte, diagnostics []diagnostic.Diagnostic) (Plan, error) {
	byPath := make(map[string][]plannedEdit)
	for _, finding := range diagnostics {
		if finding.Fix == nil {
			continue
		}
		if _, ok := sources[finding.Filename]; !ok {
			return Plan{}, fmt.Errorf("fix source %q is unavailable", finding.Filename)
		}
		for _, edit := range finding.Fix.Edits {
			byPath[finding.Filename] = append(byPath[finding.Filename], plannedEdit{start: edit.Range.Start.Offset, end: edit.Range.End.Offset, text: edit.NewText})
		}
	}
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	plan := Plan{Changes: make([]Change, 0, len(paths))}
	for _, path := range paths {
		source := sources[path]
		edits, err := validateEdits(path, len(source), byPath[path])
		if err != nil {
			return Plan{}, err
		}
		if len(edits) == 0 {
			continue
		}
		updated := append([]byte(nil), source...)
		for i := len(edits) - 1; i >= 0; i-- {
			edit := edits[i]
			replacement := []byte(edit.text)
			next := make([]byte, 0, len(updated)-(edit.end-edit.start)+len(replacement))
			next = append(next, updated[:edit.start]...)
			next = append(next, replacement...)
			next = append(next, updated[edit.end:]...)
			updated = next
		}
		if !bytes.Equal(source, updated) {
			plan.Changes = append(plan.Changes, Change{Path: path, Before: append([]byte(nil), source...), After: updated})
		}
	}
	return plan, nil
}

func validateEdits(path string, size int, edits []plannedEdit) ([]plannedEdit, error) {
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].start != edits[j].start {
			return edits[i].start < edits[j].start
		}
		if edits[i].end != edits[j].end {
			return edits[i].end < edits[j].end
		}
		return edits[i].text < edits[j].text
	})
	result := make([]plannedEdit, 0, len(edits))
	for _, edit := range edits {
		if edit.start < 0 || edit.end < edit.start || edit.end > size {
			return nil, fmt.Errorf("invalid fix range %d:%d for %q", edit.start, edit.end, path)
		}
		if len(result) != 0 {
			previous := result[len(result)-1]
			if edit == previous {
				continue
			}
			if edit.start < previous.end || edit.start == previous.start {
				return nil, fmt.Errorf("overlapping fixes for %q at offsets %d and %d", path, previous.start, edit.start)
			}
		}
		result = append(result, edit)
	}
	return result, nil
}

func Write(plan Plan) error {
	type preparedChange struct {
		change Change
		mode   os.FileMode
	}
	prepared := make([]preparedChange, 0, len(plan.Changes))
	seen := make(map[string]struct{}, len(plan.Changes))
	for _, change := range plan.Changes {
		if _, ok := seen[change.Path]; ok {
			return fmt.Errorf("duplicate fix target %q", change.Path)
		}
		seen[change.Path] = struct{}{}
		current, err := os.ReadFile(change.Path)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, change.Before) {
			return fmt.Errorf("source %q changed before fixes were written", change.Path)
		}
		info, err := os.Stat(change.Path)
		if err != nil {
			return err
		}
		prepared = append(prepared, preparedChange{change: change, mode: info.Mode().Perm()})
	}
	for _, item := range prepared {
		change := item.change
		dir := filepath.Dir(change.Path)
		temporary, err := os.CreateTemp(dir, ".pawnlint-*")
		if err != nil {
			return err
		}
		name := temporary.Name()
		ok := false
		defer func() {
			if !ok {
				_ = os.Remove(name)
			}
		}()
		if err := temporary.Chmod(item.mode); err != nil {
			_ = temporary.Close()
			return err
		}
		if _, err := temporary.Write(change.After); err != nil {
			_ = temporary.Close()
			return err
		}
		if err := temporary.Sync(); err != nil {
			_ = temporary.Close()
			return err
		}
		if err := temporary.Close(); err != nil {
			return err
		}
		if err := os.Rename(name, change.Path); err != nil {
			return err
		}
		ok = true
	}
	return nil
}

func Diff(plan Plan) string {
	var output strings.Builder
	for _, change := range plan.Changes {
		before := splitLines(change.Before)
		after := splitLines(change.After)
		fmt.Fprintf(&output, "--- %s\n+++ %s\n@@ -%d,%d +%d,%d @@\n", change.Path, change.Path, diffStart(before), len(before), diffStart(after), len(after))
		for _, line := range before {
			writeDiffLine(&output, '-', line)
		}
		for _, line := range after {
			writeDiffLine(&output, '+', line)
		}
	}
	return output.String()
}

func diffStart(lines []diffLine) int {
	if len(lines) == 0 {
		return 0
	}
	return 1
}

type diffLine struct {
	text       string
	terminated bool
}

func splitLines(source []byte) []diffLine {
	if len(source) == 0 {
		return nil
	}
	parts := strings.SplitAfter(string(source), "\n")
	lines := make([]diffLine, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		terminated := strings.HasSuffix(part, "\n")
		lines = append(lines, diffLine{text: strings.TrimSuffix(part, "\n"), terminated: terminated})
	}
	return lines
}

func writeDiffLine(output *strings.Builder, prefix byte, line diffLine) {
	output.WriteByte(prefix)
	output.WriteString(line.text)
	output.WriteByte('\n')
	if !line.terminated {
		output.WriteString("\\ No newline at end of file\n")
	}
}
