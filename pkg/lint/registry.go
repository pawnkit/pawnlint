package lint

import (
	"fmt"
	"sort"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

type Profile string

const (
	ProfileRecommended Profile = "recommended"
	ProfileStrict      Profile = "strict"
	ProfileAll         Profile = "all"
)

func AllProfiles() []string {
	return []string{string(ProfileRecommended), string(ProfileStrict), string(ProfileAll)}
}

func AllowedProfile(p string) bool {
	switch Profile(p) {
	case ProfileRecommended, ProfileStrict, ProfileAll:
		return true
	default:
		return false
	}
}

var curate = map[Profile]map[string]struct{}{
	ProfileStrict: {
		"unused-local": {}, "unused-parameter": {}, "unused-function": {}, "unused-global": {},
		"shadowed-variable": {}, "unused-label": {}, "constant-condition": {}, "duplicate-condition": {},
		"redundant-boolean-comparison": {}, "identical-branches": {}, "dead-write": {}, "redundant-initialization": {}, "possibly-uninitialized": {},
		"discarded-resource-handle": {}, "mismatched-resource-handle": {}, "unreleased-resource-handle": {},
		"overwritten-resource-handle": {}, "recursive-call": {}, "large-local-array": {}, "repeated-strlen-in-loop": {},
		"callback-signature": {}, "misspelled-callback": {}, "unimplemented-function": {}, "deprecated-function": {},
		"legacy-include": {}, "native-argument-count": {}, "deprecated-native": {}, "format-argument-count": {},
		"buffer-size": {}, "target-native-availability": {}, "target-constant-availability": {},
		"float-equality": {}, "non-public-callback": {}, "invalid-sentinel-comparison": {},
		"unescaped-sql-format": {}, "discarded-repeating-timer": {}, "raw-tick-subtraction": {},
		"sscanf-format-argument-count": {}, "settimerex-format-argument-count": {},
		"confusable-identifier":    {},
		"inconsistent-enum-prefix": {}, "cyclomatic-complexity": {},
		"boolean-complexity":        {},
		"maximum-nesting":           {},
		"too-many-parameters":       {},
		"prefer-const":              {},
		"redundant-forward":         {},
		"redundant-tag":             {},
		"redundant-else":            {},
		"incomplete-enum-switch":    {},
		"narrowing-conversion":      {},
		"signedness-mismatch":       {},
		"macro-repeated-parameter":  {},
		"statement-macro-hazard":    {},
		"loop-invariant-call":       {},
		"overwritten-copy":          {},
		"repeated-format-work":      {},
		"string-concatenation-loop": {},
	},
}

func profileEnables(p Profile, id string, m Metadata) bool {
	if m.Stability == StabilityPreview {
		return false
	}
	switch p {
	case ProfileAll:
		return true
	case ProfileStrict:
		if _, ok := curate[ProfileStrict][id]; ok {
			return true
		}
		return profileEnables(ProfileRecommended, id, m)
	default:
		return m.DefaultEnabled
	}
}

func EnableUnderProfile(p Profile, id string) {
	if curate[p] == nil {
		return
	}
	curate[p][id] = struct{}{}
}

type entry struct {
	rule Rule
	meta Metadata
}

type Registrar struct {
	entries []entry
	byID    map[string]int
	aliases map[string]string
}

type RuleAlias struct {
	Deprecated  string
	Replacement string
}

type Rule interface {
	Metadata() Metadata
	Run(ctx *Context)
}

func NewRegistrar() *Registrar {
	return &Registrar{byID: make(map[string]int), aliases: make(map[string]string)}
}

func (reg *Registrar) Register(r Rule) error {
	if r == nil {
		return fmt.Errorf("lint: nil rule")
	}
	meta := r.Metadata()
	if meta.ID == "" {
		return fmt.Errorf("lint: rule missing ID")
	}
	if meta.Stability != StabilityStable && meta.Stability != StabilityPreview {
		return fmt.Errorf("lint: rule %q has invalid stability", meta.ID)
	}
	if err := validateOptions(meta.ID, meta.Options); err != nil {
		return err
	}
	if _, ok := reg.byID[meta.ID]; ok {
		return fmt.Errorf("lint: duplicate rule ID %q", meta.ID)
	}
	if _, ok := reg.aliases[meta.ID]; ok {
		return fmt.Errorf("lint: rule ID %q conflicts with an alias", meta.ID)
	}
	reg.byID[meta.ID] = len(reg.entries)
	reg.entries = append(reg.entries, entry{rule: r, meta: meta})
	return nil
}

func (reg *Registrar) RegisterAlias(deprecated, replacement string) error {
	if deprecated == "" || replacement == "" || deprecated == replacement {
		return fmt.Errorf("lint: invalid rule alias %q -> %q", deprecated, replacement)
	}
	if _, exists := reg.byID[deprecated]; exists {
		return fmt.Errorf("lint: alias %q conflicts with a rule", deprecated)
	}
	if _, exists := reg.aliases[deprecated]; exists {
		return fmt.Errorf("lint: duplicate rule alias %q", deprecated)
	}
	if _, exists := reg.byID[replacement]; !exists {
		return fmt.Errorf("lint: alias %q targets unknown rule %q", deprecated, replacement)
	}
	reg.aliases[deprecated] = replacement
	return nil
}

func (reg *Registrar) MustRegisterAlias(deprecated, replacement string) {
	if err := reg.RegisterAlias(deprecated, replacement); err != nil {
		panic(err)
	}
}

func (reg *Registrar) ResolveID(id string) (string, bool, bool) {
	if _, exists := reg.byID[id]; exists {
		return id, false, true
	}
	replacement, exists := reg.aliases[id]
	if !exists {
		return "", false, false
	}
	return replacement, true, true
}

func (reg *Registrar) Aliases() []RuleAlias {
	result := make([]RuleAlias, 0, len(reg.aliases))
	for deprecated, replacement := range reg.aliases {
		result = append(result, RuleAlias{Deprecated: deprecated, Replacement: replacement})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Deprecated < result[j].Deprecated })
	return result
}

func (reg *Registrar) MustRegister(r Rule) {
	if err := reg.Register(r); err != nil {
		panic(err)
	}
}

func (reg *Registrar) IDs() []string {
	out := make([]string, len(reg.entries))
	for i, e := range reg.entries {
		out[i] = e.meta.ID
	}
	return out
}

func (reg *Registrar) All() []Metadata {
	out := make([]Metadata, len(reg.entries))
	for i, e := range reg.entries {
		out[i] = e.meta
	}
	return out
}

func (reg *Registrar) Lookup(id string) (Metadata, bool) {
	id, _, ok := reg.ResolveID(id)
	if !ok {
		return Metadata{}, false
	}
	i, ok := reg.byID[id]
	if !ok {
		return Metadata{}, false
	}
	return reg.entries[i].meta, true
}

func (reg *Registrar) Rule(id string) (Rule, bool) {
	id, _, ok := reg.ResolveID(id)
	if !ok {
		return nil, false
	}
	i, ok := reg.byID[id]
	if !ok {
		return nil, false
	}
	return reg.entries[i].rule, true
}

func (reg *Registrar) Sorted() []Metadata {
	out := reg.All()
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (reg *Registrar) EnabledForProfile(p Profile) (enabled map[string]diagnostic.Severity) {
	enabled = make(map[string]diagnostic.Severity, len(reg.entries))
	for _, e := range reg.entries {
		if profileEnables(p, e.meta.ID, e.meta) {
			enabled[e.meta.ID] = e.meta.DefaultSeverity
		}
	}
	return enabled
}
