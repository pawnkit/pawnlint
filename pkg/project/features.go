package project

type Feature uint16

type Features uint16

const (
	FeatureDefinedNames Feature = 1 << iota
	FeatureDuplicates
	FeatureConflicts
	FeatureIncludeCycles
	FeatureIncludeIssues
	FeatureReferences
	FeatureUnusedIncludes
	FeatureCallGraph
	FeatureRuntimeCalls
	FeatureFunctionEffects
	FeatureTrivia
)

func AllFeatures() Features {
	return Features(FeatureDefinedNames | FeatureDuplicates | FeatureConflicts | FeatureIncludeCycles | FeatureIncludeIssues | FeatureReferences | FeatureUnusedIncludes | FeatureCallGraph | FeatureRuntimeCalls | FeatureFunctionEffects | FeatureTrivia)
}

func NewFeatures(features ...Feature) Features {
	var result Features
	for _, feature := range features {
		result |= Features(feature)
	}
	return result.withDependencies()
}

func NewFeaturesFromSet(features Features) Features {
	return features.withDependencies()
}

func (f Features) Has(feature Feature) bool {
	return f&Features(feature) != 0
}

func (f Features) withDependencies() Features {
	if f.Has(FeatureUnusedIncludes) {
		f |= Features(FeatureIncludeIssues | FeatureReferences)
	}
	if f.Has(FeatureFunctionEffects) {
		f |= Features(FeatureCallGraph)
	}
	if f.Has(FeatureRuntimeCalls) {
		f |= Features(FeatureCallGraph)
	}
	if f.Has(FeatureCallGraph) {
		f |= Features(FeatureReferences)
	}
	return f
}
