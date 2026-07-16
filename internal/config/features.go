package config

import (
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

func (r *Resolved) ProjectFeatures(_ *lint.Registrar) project.Features {
	if r == nil || len(r.Source.ExternalRules) != 0 {
		return project.AllFeatures()
	}
	enabled := make(map[string]struct{}, len(r.Enabled))
	for id, severity := range r.Enabled {
		if severity != diagnostic.SeverityOff {
			enabled[id] = struct{}{}
		}
	}
	for _, override := range r.Overrides {
		for id, severity := range override.Enabled {
			if severity != diagnostic.SeverityOff {
				enabled[id] = struct{}{}
			}
		}
	}
	for id, severity := range r.CLIForced {
		if severity != diagnostic.SeverityOff {
			enabled[id] = struct{}{}
		}
	}
	var features project.Features
	for id := range enabled {
		features |= ruleProjectFeatures(id)
	}
	return project.NewFeaturesFromSet(features)
}

func ruleProjectFeatures(id string) project.Features {
	switch id {
	case "duplicate-function-definition", "duplicate-global-definition":
		return project.NewFeatures(project.FeatureDuplicates)
	case "conflicting-include-symbol":
		return project.NewFeatures(project.FeatureConflicts)
	case "include-cycle":
		return project.NewFeatures(project.FeatureIncludeCycles)
	case "missing-include", "ambiguous-include", "duplicate-include", "forbidden-include", "include-layering":
		return project.NewFeatures(project.FeatureIncludeIssues)
	case "unused-include":
		return project.NewFeatures(project.FeatureUnusedIncludes)
	case "unconditional-recursion", "recursive-call", "unused-function", "unused-global", "tainted-data-to-sink":
		return project.NewFeatures(project.FeatureCallGraph)
	case "argument-tag-mismatch", "incomplete-enum-switch", "unsafe-string-termination", "restricted-syntax", "target-native-availability", "overwritten-copy", "repeated-format-work", "repeated-strlen", "string-concatenation-loop":
		return project.NewFeatures(project.FeatureReferences)
	case "possibly-uninitialized":
		return project.NewFeatures(project.FeatureReferences, project.FeatureFunctionEffects)
	case "redundant-initialization", "duplicate-condition", "resource-shared":
		return project.NewFeatures(project.FeatureFunctionEffects)
	case "deprecated-function", "unimplemented-function":
		return project.NewFeatures(project.FeatureReferences, project.FeatureDefinedNames)
	case "format-argument-count", "deprecated-native", "discarded-repeating-timer", "format-argument-tag", "required-call-order", "native-argument-count", "argument-value-range", "settimerex-format-argument-count", "swapped-arguments", "buffer-size", "ignored-return-value":
		return project.NewFeatures(project.FeatureDefinedNames)
	default:
		return 0
	}
}
