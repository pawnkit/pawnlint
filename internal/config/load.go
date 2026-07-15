package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatTOML Format = "toml"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

var candidateNames = []string{"pawnlint.toml", "pawnlint.yaml", "pawnlint.yml", "pawnlint.json"}

func formatFor(path string) (Format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		return FormatTOML, nil
	case ".json":
		return FormatJSON, nil
	case ".yaml", ".yml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("config: unrecognized config extension %q (allowed: .toml, .json, .yaml, .yml)", filepath.Ext(path))
	}
}

func Load(path string) (File, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return File{}, fmt.Errorf("config: %w", err)
	}
	return loadPresets(filepath.Clean(path), nil)
}

func loadFile(path string) (File, error) {
	format, err := formatFor(path)
	if err != nil {
		return File{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("config: %w", err)
	}
	return decode(b, format)
}

func decode(b []byte, format Format) (File, error) {
	switch format {
	case FormatJSON:
		return decodeJSON(b)
	case FormatYAML:
		return decodeYAML(b)
	default:
		return decodeTOML(b)
	}
}

func DecodeBytes(b []byte) (File, error) {
	f, err := decodeTOML(b)
	return rejectUnresolvedPresets(f, err)
}

func DecodeJSON(b []byte) (File, error) {
	f, err := decodeJSON(b)
	return rejectUnresolvedPresets(f, err)
}

func DecodeYAML(b []byte) (File, error) {
	f, err := decodeYAML(b)
	return rejectUnresolvedPresets(f, err)
}

func decodeTOML(b []byte) (File, error) {
	var f File
	meta, err := toml.Decode(string(b), &f)
	if err != nil {
		return File{}, fmt.Errorf("config: %w", err)
	}
	if undecoded := fixedUndecodedKeys(meta.Undecoded()); len(undecoded) > 0 {
		return f, &UnknownFieldsError{Fields: keysAsStrings(undecoded)}
	}
	f.presence.warningsAsErrors = meta.IsDefined("lint", "warnings-as-errors")
	f.presence.maxDiagnostics = meta.IsDefined("lint", "max-diagnostics")
	return withDefaultRules(f), nil
}

func fixedUndecodedKeys(keys []toml.Key) []toml.Key {
	var result []toml.Key
	for _, key := range keys {
		dynamic := len(key) >= 2 && key[0] == "rules"
		for index := 1; !dynamic && index < len(key); index++ {
			dynamic = key[index-1] == "overrides" && key[index] == "rules"
		}
		if !dynamic {
			result = append(result, key)
		}
	}
	return result
}

func decodeJSON(b []byte) (File, error) {
	var f File
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		if fields := jsonUnknownFields(err); len(fields) > 0 {
			return File{}, &UnknownFieldsError{Fields: fields}
		}
		return File{}, fmt.Errorf("config: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err == nil {
		if lintRaw := raw["lint"]; len(lintRaw) != 0 {
			var section map[string]json.RawMessage
			if json.Unmarshal(lintRaw, &section) == nil {
				_, f.presence.warningsAsErrors = section["warnings-as-errors"]
				_, f.presence.maxDiagnostics = section["max-diagnostics"]
			}
		}
	}
	return withDefaultRules(f), nil
}

func decodeYAML(b []byte) (File, error) {
	var f File
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		if fields := yamlUnknownFields(err); len(fields) > 0 {
			return File{}, &UnknownFieldsError{Fields: fields}
		}
		return File{}, fmt.Errorf("config: %w", err)
	}
	var raw map[string]any
	if yaml.Unmarshal(b, &raw) == nil {
		if section, ok := raw["lint"].(map[string]any); ok {
			_, f.presence.warningsAsErrors = section["warnings-as-errors"]
			_, f.presence.maxDiagnostics = section["max-diagnostics"]
		}
	}
	return withDefaultRules(f), nil
}

func rejectUnresolvedPresets(f File, err error) (File, error) {
	if err != nil {
		return File{}, err
	}
	if len(f.Presets) != 0 {
		return File{}, fmt.Errorf("config: presets require loading from a file")
	}
	return f, nil
}

func withDefaultRules(f File) File {
	if f.Rules == nil {
		f.Rules = map[string]any{}
	}
	return f
}

var jsonUnknownFieldPattern = regexp.MustCompile(`unknown field "([^"]+)"`)

func jsonUnknownFields(err error) []string {
	m := jsonUnknownFieldPattern.FindStringSubmatch(err.Error())
	if m == nil {
		return nil
	}
	return []string{m[1]}
}

var yamlUnknownFieldPattern = regexp.MustCompile(`field (\S+) not found`)

func yamlUnknownFields(err error) []string {
	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		return nil
	}
	var fields []string
	for _, e := range typeErr.Errors {
		if m := yamlUnknownFieldPattern.FindStringSubmatch(e); m != nil {
			fields = append(fields, m[1])
		}
	}
	sort.Strings(fields)
	return fields
}

type UnknownFieldsError struct {
	Fields []string
}

func (e *UnknownFieldsError) Error() string {
	return "unknown config fields: " + strings.Join(e.Fields, ", ")
}

func keysAsStrings(ks []toml.Key) []string {
	out := make([]string, 0, len(ks))
	for _, k := range ks {
		out = append(out, strings.Join(k, "."))
	}
	sort.Strings(out)
	return out
}

func Discover(startDir string) (string, File, error) {
	dir := startDir
	for i := 0; i < 64; i++ {
		for _, name := range candidateNames {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				f, err := Load(candidate)
				if err != nil {
					return candidate, File{}, err
				}
				return candidate, f, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", Defaults(), nil
}
