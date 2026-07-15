package baseline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

const Version = 1

type File struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

type Entry struct {
	Fingerprint string `json:"fingerprint"`
	RuleID      string `json:"ruleId"`
	Path        string `json:"path"`
	Message     string `json:"message"`
	Line        int    `json:"line"`
}

type Match struct {
	Remaining []diagnostic.Diagnostic
	Current   File
	Stale     int
}

func Load(path string) (File, error) {
	input, err := os.Open(path)
	if err != nil {
		return File{}, fmt.Errorf("baseline: %w", err)
	}
	defer func() { _ = input.Close() }()
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	var file File
	if err := decoder.Decode(&file); err != nil {
		return File{}, fmt.Errorf("baseline %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return File{}, fmt.Errorf("baseline %s: multiple JSON values", path)
		}
		return File{}, fmt.Errorf("baseline %s: %w", path, err)
	}
	if err := Validate(file); err != nil {
		return File{}, fmt.Errorf("baseline %s: %w", path, err)
	}
	return file, nil
}

func Validate(file File) error {
	if file.Version != Version {
		return fmt.Errorf("unsupported version %d", file.Version)
	}
	seen := make(map[string]struct{}, len(file.Entries))
	for index, entry := range file.Entries {
		position := fmt.Sprintf("entry %d", index+1)
		if !validFingerprint(entry.Fingerprint) {
			return fmt.Errorf("%s has invalid fingerprint", position)
		}
		if _, duplicate := seen[entry.Fingerprint]; duplicate {
			return fmt.Errorf("%s repeats fingerprint %s", position, entry.Fingerprint)
		}
		seen[entry.Fingerprint] = struct{}{}
		if strings.TrimSpace(entry.RuleID) == "" {
			return fmt.Errorf("%s has empty ruleId", position)
		}
		if entry.Path == "" || filepath.IsAbs(entry.Path) || filepath.ToSlash(filepath.Clean(entry.Path)) != entry.Path {
			return fmt.Errorf("%s has invalid project-relative path", position)
		}
		if strings.TrimSpace(entry.Message) == "" {
			return fmt.Errorf("%s has empty message", position)
		}
		if entry.Line < 1 {
			return fmt.Errorf("%s has invalid line", position)
		}
	}
	return nil
}

func Generate(diagnostics []diagnostic.Diagnostic, sources map[string][]byte, projectDir string) File {
	entries, _ := fingerprintDiagnostics(diagnostics, sources, projectDir)
	result := File{Version: Version, Entries: make([]Entry, 0, len(entries))}
	for _, entry := range entries {
		if entry.Fingerprint != "" {
			result.Entries = append(result.Entries, entry)
		}
	}
	sortEntries(result.Entries)
	return result
}

func Apply(file File, diagnostics []diagnostic.Diagnostic, sources map[string][]byte, projectDir string) Match {
	entries, indexed := fingerprintDiagnostics(diagnostics, sources, projectDir)
	existing := make(map[string]struct{}, len(file.Entries))
	for _, entry := range file.Entries {
		existing[entry.Fingerprint] = struct{}{}
	}
	matched := make(map[string]Entry)
	remaining := make([]diagnostic.Diagnostic, 0, len(diagnostics))
	for index, item := range diagnostics {
		entry := entries[indexed[index]]
		if entry.Fingerprint == "" {
			remaining = append(remaining, item)
			continue
		}
		if _, found := existing[entry.Fingerprint]; !found {
			remaining = append(remaining, item)
			continue
		}
		matched[entry.Fingerprint] = entry
	}
	current := File{Version: Version, Entries: make([]Entry, 0, len(matched))}
	for _, entry := range matched {
		current.Entries = append(current.Entries, entry)
	}
	sortEntries(current.Entries)
	return Match{Remaining: remaining, Current: current, Stale: len(file.Entries) - len(current.Entries)}
}

func Write(path string, file File) error {
	if err := Validate(file); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(directory, ".pawnlint-baseline-*")
	if err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	temporaryPath := temporary.Name()
	remove := true
	defer func() {
		_ = temporary.Close()
		if remove {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	remove = false
	return nil
}

func fingerprintDiagnostics(diagnostics []diagnostic.Diagnostic, sources map[string][]byte, projectDir string) ([]Entry, map[int]int) {
	type item struct {
		index int
		diag  diagnostic.Diagnostic
	}
	items := make([]item, len(diagnostics))
	for index, diag := range diagnostics {
		items[index] = item{index: index, diag: diag}
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i].diag, items[j].diag
		if left.Filename != right.Filename {
			return left.Filename < right.Filename
		}
		if left.Range.Start.Offset != right.Range.Start.Offset {
			return left.Range.Start.Offset < right.Range.Start.Offset
		}
		if left.RuleID != right.RuleID {
			return left.RuleID < right.RuleID
		}
		return left.Message < right.Message
	})
	entries := make([]Entry, len(items))
	indexed := make(map[int]int, len(items))
	occurrences := make(map[string]int)
	for position, item := range items {
		indexed[item.index] = position
		diag := item.diag
		if !eligible(diag) {
			continue
		}
		path := relativePath(projectDir, diag.Filename)
		source := sourceFor(sources, diag.Filename)
		excerpt := diagnosticExcerpt(source, diag)
		base := strings.Join([]string{path, diag.RuleID, diag.Code, diag.Message, excerpt}, "\x00")
		occurrence := occurrences[base]
		occurrences[base] = occurrence + 1
		sum := sha256.Sum256([]byte(base + "\x00" + strconv.Itoa(occurrence)))
		entries[position] = Entry{
			Fingerprint: hex.EncodeToString(sum[:]),
			RuleID:      diag.RuleID,
			Path:        path,
			Message:     diag.Message,
			Line:        max(diag.Range.Start.Line, 1),
		}
	}
	return entries, indexed
}

func eligible(diag diagnostic.Diagnostic) bool {
	return diag.RuleID != "" && diag.RuleID != "parse-error" && diag.RuleID != "internal-error" && diag.Message != ""
}

func relativePath(projectDir, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectDir, path)
	}
	relative, err := filepath.Rel(projectDir, path)
	if err != nil {
		return filepath.ToSlash(filepath.Clean(path))
	}
	return filepath.ToSlash(filepath.Clean(relative))
}

func sourceFor(sources map[string][]byte, path string) []byte {
	if source, found := sources[path]; found {
		return source
	}
	clean := filepath.Clean(path)
	for candidate, source := range sources {
		if filepath.Clean(candidate) == clean {
			return source
		}
	}
	return nil
}

func diagnosticExcerpt(source []byte, diag diagnostic.Diagnostic) string {
	start := diag.Range.Start.Offset
	end := diag.Range.End.Offset
	if start < 0 || end < start || end > len(source) {
		return ""
	}
	return strings.Join(strings.Fields(string(source[start:end])), " ")
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Path != entries[j].Path {
			return entries[i].Path < entries[j].Path
		}
		if entries[i].RuleID != entries[j].RuleID {
			return entries[i].RuleID < entries[j].RuleID
		}
		return entries[i].Fingerprint < entries[j].Fingerprint
	})
}

func validFingerprint(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}
