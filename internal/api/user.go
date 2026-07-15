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
	Functions map[string]Function `json:"functions,omitempty"`
	Constants map[string]Constant `json:"constants,omitempty"`
}

func Builtin(target string) *Metadata {
	return &Metadata{
		Callbacks: copyMap(Callbacks(target)),
		Natives:   copyMap(Natives(target)),
		Functions: make(map[string]Function),
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
		for name, function := range metadata.Functions {
			function.Name = name
			result.Functions[name] = function
		}
		for name, constant := range metadata.Constants {
			constant.Name = name
			result.Constants[name] = constant
		}
	}
	if err := validateRelations(result); err != nil {
		return nil, err
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
	if m.Functions == nil {
		m.Functions = make(map[string]Function)
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
		if native.Pure && (native.Release != "" || len(native.RequiresBefore) != 0 || len(native.Buffers) != 0 || mutableParameters(native.Parameters)) {
			problems = append(problems, fmt.Sprintf("native %s has effects incompatible with pure", name))
		}
		seenRequirements := make(map[string]struct{}, len(native.RequiresBefore))
		for _, requirement := range native.RequiresBefore {
			if strings.TrimSpace(requirement) == "" {
				problems = append(problems, fmt.Sprintf("native %s has an empty call prerequisite", name))
				continue
			}
			if requirement == name {
				problems = append(problems, fmt.Sprintf("native %s requires itself", name))
			}
			if _, exists := seenRequirements[requirement]; exists {
				problems = append(problems, fmt.Sprintf("native %s repeats call prerequisite %s", name, requirement))
			}
			seenRequirements[requirement] = struct{}{}
		}
	}
	for name, function := range m.Functions {
		if strings.TrimSpace(name) == "" {
			problems = append(problems, "function name is empty")
			continue
		}
		function.Name = name
		m.Functions[name] = function
		validateParameters("function "+name, function.Parameters, &problems)
		if function.Pure && (function.Release != "" || mutableParameters(function.Parameters)) {
			problems = append(problems, fmt.Sprintf("function %s has effects incompatible with pure", name))
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

func mutableParameters(parameters []Parameter) bool {
	for _, parameter := range parameters {
		if parameter.Output || parameter.Reference || parameter.ArrayRank > 0 && !parameter.Const {
			return true
		}
	}
	return false
}

func validateNativeRelations(natives map[string]Native) error {
	return validateRelations(&Metadata{Natives: natives, Functions: make(map[string]Function)})
}

func validateRelations(metadata *Metadata) error {
	natives := metadata.Natives
	names := make([]string, 0, len(natives))
	for name := range natives {
		names = append(names, name)
	}
	sort.Strings(names)
	var problems []string
	for _, name := range names {
		native := natives[name]
		if native.Release != "" {
			_, nativeRelease := natives[native.Release]
			_, functionRelease := metadata.Functions[native.Release]
			if !nativeRelease && !functionRelease {
				problems = append(problems, fmt.Sprintf("native %q names unknown releaser %q", name, native.Release))
			}
			if native.Release == name {
				problems = append(problems, fmt.Sprintf("native %q releases itself", name))
			}
		}
		for _, requirement := range native.RequiresBefore {
			if _, ok := natives[requirement]; !ok {
				problems = append(problems, fmt.Sprintf("native %q names unknown call prerequisite %q", name, requirement))
			}
			if requirement == name {
				problems = append(problems, fmt.Sprintf("native %q requires itself", name))
			}
		}
	}
	functionNames := make([]string, 0, len(metadata.Functions))
	for name := range metadata.Functions {
		functionNames = append(functionNames, name)
	}
	sort.Strings(functionNames)
	for _, name := range functionNames {
		function := metadata.Functions[name]
		if function.Release == "" {
			continue
		}
		_, nativeRelease := natives[function.Release]
		_, functionRelease := metadata.Functions[function.Release]
		if !nativeRelease && !functionRelease {
			problems = append(problems, fmt.Sprintf("function %q names unknown releaser %q", name, function.Release))
		}
		if function.Release == name {
			problems = append(problems, fmt.Sprintf("function %q releases itself", name))
		}
	}
	if cycle := nativeRequirementCycle(natives, names); len(cycle) != 0 {
		problems = append(problems, "native call prerequisites contain cycle "+strings.Join(cycle, " -> "))
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("API %s", strings.Join(problems, "; "))
}

func nativeRequirementCycle(natives map[string]Native, names []string) []string {
	state := make(map[string]uint8, len(natives))
	var stack []string
	var visit func(string) []string
	visit = func(name string) []string {
		state[name] = 1
		stack = append(stack, name)
		requirements := append([]string(nil), natives[name].RequiresBefore...)
		sort.Strings(requirements)
		for _, requirement := range requirements {
			if _, exists := natives[requirement]; !exists {
				continue
			}
			if state[requirement] == 1 {
				for index, item := range stack {
					if item == requirement {
						return append(append([]string(nil), stack[index:]...), requirement)
					}
				}
			}
			if state[requirement] == 0 {
				if cycle := visit(requirement); len(cycle) != 0 {
					return cycle
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = 2
		return nil
	}
	for _, name := range names {
		if state[name] == 0 {
			if cycle := visit(name); len(cycle) != 0 {
				return cycle
			}
		}
	}
	return nil
}

func validateParameters(owner string, parameters []Parameter, problems *[]string) {
	variadic := false
	for index, parameter := range parameters {
		position := fmt.Sprintf("%s parameter %d", owner, index+1)
		if parameter.ArrayRank < 0 {
			*problems = append(*problems, fmt.Sprintf("%s parameter %d has negative arrayRank", owner, index+1))
		}
		if parameter.Minimum != nil || parameter.Maximum != nil {
			if parameter.ArrayRank != 0 || parameter.Output || parameter.Variadic {
				*problems = append(*problems, position+" has value bounds on a non-scalar or non-input parameter")
			}
			if parameter.Minimum != nil && parameter.Maximum != nil && *parameter.Minimum > *parameter.Maximum {
				*problems = append(*problems, position+" has minimum greater than maximum")
			}
			minimumOutside := parameter.Minimum != nil && (*parameter.Minimum < -2147483648 || *parameter.Minimum > 2147483647)
			maximumOutside := parameter.Maximum != nil && (*parameter.Maximum < -2147483648 || *parameter.Maximum > 2147483647)
			if minimumOutside || maximumOutside {
				*problems = append(*problems, position+" has value bounds outside the Pawn cell range")
			}
		}
		if parameter.Ownership != "" && parameter.Ownership != "borrowed" && parameter.Ownership != "transferred" {
			*problems = append(*problems, position+" has invalid ownership "+parameter.Ownership)
		}
		if parameter.Ownership != "" && (parameter.ArrayRank != 0 || parameter.Reference || parameter.Output || parameter.Variadic) {
			*problems = append(*problems, position+" has ownership on a non-scalar input parameter")
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
