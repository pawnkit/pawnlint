package lint_test

import (
	"reflect"
	"testing"

	"github.com/pawnkit/pawnlint/pkg/lint"
)

func TestNormalizeOptions(t *testing.T) {
	tests := []struct {
		option lint.Option
		input  any
		want   any
	}{
		{lint.Option{Type: lint.OptionBoolean}, true, true},
		{lint.Option{Type: lint.OptionInteger, Minimum: 1, HasMinimum: true}, 20, int64(20)},
		{lint.Option{Type: lint.OptionString, Choices: []string{"one", "two"}}, "two", "two"},
		{lint.Option{Type: lint.OptionStringList, Choices: []string{"one", "two"}}, []any{"one", "two"}, []string{"one", "two"}},
		{lint.Option{Type: lint.OptionObjectList, Fields: []lint.Option{{Name: "kind", Type: lint.OptionString, Choices: []string{"one", "two"}}, {Name: "enabled", Type: lint.OptionBoolean, Default: true}}}, []any{map[string]any{"kind": "two"}}, []map[string]any{{"kind": "two", "enabled": true}}},
	}
	for _, test := range tests {
		got, err := lint.NormalizeOption(test.option, test.input)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("normalized option = %#v, want %#v", got, test.want)
		}
	}
}

func TestNormalizeOptionsRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		option lint.Option
		input  any
	}{
		{lint.Option{Type: lint.OptionBoolean}, "true"},
		{lint.Option{Type: lint.OptionInteger}, 1.5},
		{lint.Option{Type: lint.OptionInteger, Maximum: 5, HasMaximum: true}, 6},
		{lint.Option{Type: lint.OptionString, Choices: []string{"one"}}, "two"},
		{lint.Option{Type: lint.OptionStringList}, []any{"one", 2}},
		{lint.Option{Type: lint.OptionObjectList, Fields: []lint.Option{{Name: "kind", Type: lint.OptionString}}}, []any{map[string]any{"unknown": "one"}}},
	}
	for _, test := range tests {
		if _, err := lint.NormalizeOption(test.option, test.input); err == nil {
			t.Fatalf("invalid option accepted: %#v", test.input)
		}
	}
}
