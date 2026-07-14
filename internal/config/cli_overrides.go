package config

import (
	"fmt"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func (r *Resolved) ApplyCLIOverrides(profile, target string, enable, disable []string, reg *lint.Registrar) error {
	if profile != "" {
		if !lint.AllowedProfile(profile) {
			return fmt.Errorf("unknown profile %q", profile)
		}
		r.Profile = profile
		r.Source.Profile = profile
		enabled := reg.EnabledForProfile(lint.Profile(profile))
		for id, value := range r.Source.Rules {
			sev, ok := configuredSeverity(value)
			if !ok {
				continue
			}
			if sev == diagnostic.SeverityOff {
				delete(enabled, id)
			} else {
				enabled[id] = sev
			}
		}
		r.Enabled = enabled
	}
	if target != "" {
		if !allowedTarget(target) {
			return fmt.Errorf("unknown target %q", target)
		}
		metadata, err := loadAPIMetadata(r.Source.APIMetadata, r.SourcePath, target)
		if err != nil {
			return err
		}
		r.Target = Target(target)
		r.Source.Target = target
		r.API = metadata
	}
	for _, id := range enable {
		if _, known := r.AllKnownRuleIDs[id]; !known {
			return fmt.Errorf("unknown rule ID %q in --enable", id)
		}
		m, ok := reg.Lookup(id)
		if !ok {
			return fmt.Errorf("unknown rule ID %q in --enable", id)
		}
		r.Enabled[id] = m.DefaultSeverity
	}
	for _, id := range disable {
		if _, known := r.AllKnownRuleIDs[id]; !known {
			return fmt.Errorf("unknown rule ID %q in --disable", id)
		}
		delete(r.Enabled, id)
	}
	return nil
}

func configuredSeverity(value any) (diagnostic.Severity, bool) {
	switch v := value.(type) {
	case string:
		sev, ok := diagnostic.ParseSeverity(v)
		return sev, ok
	case map[string]any:
		raw, ok := v["severity"]
		if !ok {
			return diagnostic.SeverityOff, false
		}
		name, ok := raw.(string)
		if !ok {
			return diagnostic.SeverityOff, false
		}
		sev, ok := diagnostic.ParseSeverity(name)
		return sev, ok
	default:
		return diagnostic.SeverityOff, false
	}
}
