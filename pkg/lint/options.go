package lint

import (
	"fmt"
	"math"
	"sort"
)

type OptionType uint8

const (
	OptionBoolean OptionType = iota
	OptionInteger
	OptionString
	OptionStringList
	OptionObjectList
)

func (t OptionType) String() string {
	switch t {
	case OptionBoolean:
		return "boolean"
	case OptionInteger:
		return "integer"
	case OptionString:
		return "string"
	case OptionStringList:
		return "string-list"
	case OptionObjectList:
		return "object-list"
	default:
		return "unknown"
	}
}

type Option struct {
	Name       string
	Summary    string
	Type       OptionType
	Default    any
	Minimum    int64
	Maximum    int64
	HasMinimum bool
	HasMaximum bool
	Choices    []string
	Fields     []Option
	Required   bool
	Validate   func(any) error
}

func validateOptions(ruleID string, options []Option) error {
	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		if option.Name == "" || option.Name == "severity" {
			return fmt.Errorf("lint: rule %q has invalid option name %q", ruleID, option.Name)
		}
		if _, exists := seen[option.Name]; exists {
			return fmt.Errorf("lint: rule %q has duplicate option %q", ruleID, option.Name)
		}
		seen[option.Name] = struct{}{}
		if option.Type > OptionObjectList {
			return fmt.Errorf("lint: rule %q option %q has invalid type", ruleID, option.Name)
		}
		if option.HasMinimum && option.HasMaximum && option.Minimum > option.Maximum {
			return fmt.Errorf("lint: rule %q option %q has invalid range", ruleID, option.Name)
		}
		if option.Type != OptionInteger && (option.HasMinimum || option.HasMaximum) {
			return fmt.Errorf("lint: rule %q option %q has numeric constraints on a non-integer", ruleID, option.Name)
		}
		if option.Type != OptionString && option.Type != OptionStringList && len(option.Choices) != 0 {
			return fmt.Errorf("lint: rule %q option %q has choices on a non-string", ruleID, option.Name)
		}
		if option.Type != OptionObjectList && len(option.Fields) != 0 {
			return fmt.Errorf("lint: rule %q option %q has fields on a non-object list", ruleID, option.Name)
		}
		if option.Type == OptionObjectList {
			if err := validateObjectFields(ruleID, option.Name, option.Fields); err != nil {
				return err
			}
		}
		if option.Default != nil {
			if _, err := NormalizeOption(option, option.Default); err != nil {
				return fmt.Errorf("lint: rule %q option %q has invalid default: %w", ruleID, option.Name, err)
			}
		}
	}
	return nil
}

func NormalizeOption(option Option, value any) (any, error) {
	result, err := normalizeOptionValue(option, value)
	if err != nil {
		return nil, err
	}
	if option.Validate != nil {
		if err := option.Validate(result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func normalizeOptionValue(option Option, value any) (any, error) {
	switch option.Type {
	case OptionBoolean:
		result, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("must be a boolean")
		}
		return result, nil
	case OptionInteger:
		result, ok := integerOption(value)
		if !ok {
			return nil, fmt.Errorf("must be an integer")
		}
		if option.HasMinimum && result < option.Minimum {
			return nil, fmt.Errorf("must be at least %d", option.Minimum)
		}
		if option.HasMaximum && result > option.Maximum {
			return nil, fmt.Errorf("must be at most %d", option.Maximum)
		}
		return result, nil
	case OptionString:
		result, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("must be a string")
		}
		if len(option.Choices) != 0 && !containsOptionChoice(option.Choices, result) {
			return nil, fmt.Errorf("must be one of %v", option.Choices)
		}
		return result, nil
	case OptionStringList:
		result, ok := stringListOption(value)
		if !ok {
			return nil, fmt.Errorf("must be a list of strings")
		}
		if len(option.Choices) != 0 {
			for _, item := range result {
				if !containsOptionChoice(option.Choices, item) {
					return nil, fmt.Errorf("entries must be one of %v", option.Choices)
				}
			}
		}
		return result, nil
	case OptionObjectList:
		return objectListOption(option, value)
	default:
		return nil, fmt.Errorf("has an invalid type")
	}
}

func validateObjectFields(ruleID, owner string, fields []Option) error {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field.Name == "" || field.Name == "severity" {
			return fmt.Errorf("lint: rule %q option %q has invalid field name %q", ruleID, owner, field.Name)
		}
		if _, exists := seen[field.Name]; exists {
			return fmt.Errorf("lint: rule %q option %q has duplicate field %q", ruleID, owner, field.Name)
		}
		seen[field.Name] = struct{}{}
		if field.Type > OptionStringList || len(field.Fields) != 0 {
			return fmt.Errorf("lint: rule %q option %q field %q has invalid type", ruleID, owner, field.Name)
		}
		if field.HasMinimum && field.HasMaximum && field.Minimum > field.Maximum {
			return fmt.Errorf("lint: rule %q option %q field %q has invalid range", ruleID, owner, field.Name)
		}
		if field.Type != OptionInteger && (field.HasMinimum || field.HasMaximum) {
			return fmt.Errorf("lint: rule %q option %q field %q has numeric constraints on a non-integer", ruleID, owner, field.Name)
		}
		if field.Type != OptionString && field.Type != OptionStringList && len(field.Choices) != 0 {
			return fmt.Errorf("lint: rule %q option %q field %q has choices on a non-string", ruleID, owner, field.Name)
		}
		if field.Default != nil {
			if _, err := NormalizeOption(field, field.Default); err != nil {
				return fmt.Errorf("lint: rule %q option %q field %q has invalid default: %w", ruleID, owner, field.Name, err)
			}
		}
	}
	return nil
}

func objectListOption(option Option, value any) ([]map[string]any, error) {
	items, ok := value.([]any)
	if !ok {
		if typed, valid := value.([]map[string]any); valid {
			items = make([]any, len(typed))
			for index := range typed {
				items[index] = typed[index]
			}
		} else {
			return nil, fmt.Errorf("must be a list of objects")
		}
	}
	fields := make(map[string]Option, len(option.Fields))
	for _, field := range option.Fields {
		fields[field.Name] = field
	}
	result := make([]map[string]any, len(items))
	for index, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("entry %d must be an object", index+1)
		}
		normalized := make(map[string]any, len(option.Fields))
		names := make([]string, 0, len(object))
		for name := range object {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			field, known := fields[name]
			if !known {
				return nil, fmt.Errorf("entry %d has unknown field %q", index+1, name)
			}
			value, err := NormalizeOption(field, object[name])
			if err != nil {
				return nil, fmt.Errorf("entry %d field %q: %w", index+1, name, err)
			}
			normalized[name] = value
		}
		for _, field := range option.Fields {
			if _, exists := normalized[field.Name]; exists {
				continue
			}
			if field.Required {
				return nil, fmt.Errorf("entry %d requires field %q", index+1, field.Name)
			}
			if field.Default != nil {
				value, err := NormalizeOption(field, field.Default)
				if err != nil {
					return nil, fmt.Errorf("entry %d field %q default: %w", index+1, field.Name, err)
				}
				normalized[field.Name] = value
			}
		}
		result[index] = normalized
	}
	return result, nil
}

func integerOption(value any) (int64, bool) {
	switch item := value.(type) {
	case int:
		return int64(item), true
	case int8:
		return int64(item), true
	case int16:
		return int64(item), true
	case int32:
		return int64(item), true
	case int64:
		return item, true
	case uint:
		if uint64(item) <= math.MaxInt64 {
			return int64(item), true
		}
	case uint8:
		return int64(item), true
	case uint16:
		return int64(item), true
	case uint32:
		return int64(item), true
	case uint64:
		if item <= math.MaxInt64 {
			return int64(item), true
		}
	case float64:
		result := int64(item)
		if item == float64(result) {
			return result, true
		}
	}
	return 0, false
}

func stringListOption(value any) ([]string, bool) {
	if items, ok := value.([]string); ok {
		return append([]string(nil), items...), true
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, len(items))
	for i, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		result[i] = text
	}
	return result, true
}

func containsOptionChoice(choices []string, value string) bool {
	for _, choice := range choices {
		if choice == value {
			return true
		}
	}
	return false
}
