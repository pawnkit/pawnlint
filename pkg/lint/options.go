package lint

import (
	"fmt"
	"math"
)

type OptionType uint8

const (
	OptionBoolean OptionType = iota
	OptionInteger
	OptionString
	OptionStringList
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
		if option.Type > OptionStringList {
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
		if option.Default != nil {
			if _, err := NormalizeOption(option, option.Default); err != nil {
				return fmt.Errorf("lint: rule %q option %q has invalid default: %w", ruleID, option.Name, err)
			}
		}
	}
	return nil
}

func NormalizeOption(option Option, value any) (any, error) {
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
	default:
		return nil, fmt.Errorf("has an invalid type")
	}
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
