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
		for configuredID, value := range r.Source.Rules {
			id, deprecated, known := reg.ResolveID(configuredID)
			if !known {
				continue
			}
			if deprecated {
				r.RuleMigrations = appendRuleMigrations(r.RuleMigrations, RuleMigration{Deprecated: configuredID, Replacement: id})
			}
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
		for i := range r.Source.Builds {
			r.Source.Builds[i].Target = target
		}
		r.API = metadata
	}
	for _, configuredID := range enable {
		id, deprecated, ok := reg.ResolveID(configuredID)
		if !ok {
			return fmt.Errorf("unknown rule ID %q in --enable", configuredID)
		}
		if deprecated {
			r.RuleMigrations = appendRuleMigrations(r.RuleMigrations, RuleMigration{Deprecated: configuredID, Replacement: id})
		}
		m, _ := reg.Lookup(id)
		r.Enabled[id] = m.DefaultSeverity
	}
	for _, configuredID := range disable {
		id, deprecated, ok := reg.ResolveID(configuredID)
		if !ok {
			return fmt.Errorf("unknown rule ID %q in --disable", configuredID)
		}
		if deprecated {
			r.RuleMigrations = appendRuleMigrations(r.RuleMigrations, RuleMigration{Deprecated: configuredID, Replacement: id})
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
