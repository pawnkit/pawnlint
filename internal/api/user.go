package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type Metadata struct {
	Callbacks map[string]Callback `json:"callbacks,omitempty"`
	Natives   map[string]Native   `json:"natives,omitempty"`
	Constants map[string]Constant `json:"constants,omitempty"`
}

func Builtin(target string) *Metadata {
	return &Metadata{
		Callbacks: copyMap(Callbacks(target)),
		Natives:   copyMap(Natives(target)),
		Constants: copyMap(Constants(target)),
	}
}

func Load(path string) (*Metadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	metadata := &Metadata{}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(metadata); err != nil {
		return nil, fmt.Errorf("API metadata %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("API metadata %s: multiple JSON values", path)
		}
		return nil, fmt.Errorf("API metadata %s: %w", path, err)
	}
	metadata.initialize()
	if err := metadata.validate(path); err != nil {
		return nil, err
	}
	return metadata, nil
}

func Merge(target string, custom ...*Metadata) (*Metadata, error) {
	result := Builtin(target)
	for _, metadata := range custom {
		if metadata == nil {
			continue
		}
		for name, callback := range metadata.Callbacks {
			callback.Name = name
			result.Callbacks[name] = callback
		}
		for name, native := range metadata.Natives {
			native.Name = name
			result.Natives[name] = native
		}
		for name, constant := range metadata.Constants {
			constant.Name = name
			result.Constants[name] = constant
		}
	}
	for name, native := range result.Natives {
		if native.Release != "" {
			if _, ok := result.Natives[native.Release]; !ok {
				return nil, fmt.Errorf("API native %q names unknown releaser %q", name, native.Release)
			}
		}
	}
	return result, nil
}

func (m *Metadata) initialize() {
	if m.Callbacks == nil {
		m.Callbacks = make(map[string]Callback)
	}
	if m.Natives == nil {
		m.Natives = make(map[string]Native)
	}
	if m.Constants == nil {
		m.Constants = make(map[string]Constant)
	}
}

func (m *Metadata) validate(path string) error {
	var problems []string
	for name, callback := range m.Callbacks {
		if strings.TrimSpace(name) == "" {
			problems = append(problems, "callback name is empty")
			continue
		}
		callback.Name = name
		m.Callbacks[name] = callback
		validateParameters("callback "+name, callback.Parameters, &problems)
	}
	for name, native := range m.Natives {
		if strings.TrimSpace(name) == "" {
			problems = append(problems, "native name is empty")
			continue
		}
		native.Name = name
		m.Natives[name] = native
		validateParameters("native "+name, native.Parameters, &problems)
		if native.FormatParameter < 0 || native.FormatParameter > len(native.Parameters) {
			problems = append(problems, fmt.Sprintf("native %s has invalid formatParameter %d", name, native.FormatParameter))
		}
		for _, buffer := range native.Buffers {
			if buffer.Parameter < 1 || buffer.Parameter > len(native.Parameters) || buffer.SizeParameter < 1 || buffer.SizeParameter > len(native.Parameters) {
				problems = append(problems, fmt.Sprintf("native %s has invalid buffer relation %d:%d", name, buffer.Parameter, buffer.SizeParameter))
			}
		}
	}
	for name, constant := range m.Constants {
		if strings.TrimSpace(name) == "" {
			problems = append(problems, "constant name is empty")
			continue
		}
		constant.Name = name
		m.Constants[name] = constant
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("API metadata %s: %s", path, strings.Join(problems, "; "))
}

func validateParameters(owner string, parameters []Parameter, problems *[]string) {
	variadic := false
	for index, parameter := range parameters {
		if parameter.ArrayRank < 0 {
			*problems = append(*problems, fmt.Sprintf("%s parameter %d has negative arrayRank", owner, index+1))
		}
		if variadic {
			*problems = append(*problems, fmt.Sprintf("%s has a parameter after its variadic parameter", owner))
		}
		variadic = variadic || parameter.Variadic
	}
}

func copyMap[K comparable, V any](source map[K]V) map[K]V {
	result := make(map[K]V, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
