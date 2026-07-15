package api

type Callback struct {
	Name       string      `json:"name,omitempty"`
	ReturnTag  string      `json:"returnTag,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

type Parameter struct {
	Name      string `json:"name,omitempty"`
	Tag       string `json:"tag,omitempty"`
	ArrayRank int    `json:"arrayRank,omitempty"`
	Const     bool   `json:"const,omitempty"`
	Reference bool   `json:"reference,omitempty"`
	Output    bool   `json:"output,omitempty"`
	Variadic  bool   `json:"variadic,omitempty"`
	Default   bool   `json:"default,omitempty"`
}

type Native struct {
	Name            string      `json:"name,omitempty"`
	ReturnTag       string      `json:"returnTag,omitempty"`
	Parameters      []Parameter `json:"parameters,omitempty"`
	Deprecated      string      `json:"deprecated,omitempty"`
	FormatParameter int         `json:"formatParameter,omitempty"`
	Buffers         []Buffer    `json:"buffers,omitempty"`
	OpenMPOnly      bool        `json:"openMPOnly,omitempty"`
	Release         string      `json:"release,omitempty"`
	MustUse         bool        `json:"mustUse,omitempty"`
}

type Constant struct {
	Name       string `json:"name,omitempty"`
	OpenMPOnly bool   `json:"openMPOnly,omitempty"`
}

type UnsupportedFunction struct {
	Name      string
	Suggested string
}

type DeprecatedFunction struct {
	Name      string
	Suggested string
}

type Buffer struct {
	Parameter     int `json:"parameter"`
	SizeParameter int `json:"sizeParameter"`
}

func Callbacks(target string) map[string]Callback {
	if target == "samp" {
		return sampCallbacks
	}
	return openMPCallbacks
}

func Natives(target string) map[string]Native {
	if target == "samp" {
		return sampAPINatives
	}
	return openMPNatives
}

var sampAPINatives = func() map[string]Native {
	result := make(map[string]Native, len(sampNatives))
	for name, native := range sampNatives {
		result[name] = native
	}
	for name, native := range openMPNatives {
		if !native.OpenMPOnly {
			result[name] = native
		}
	}
	return result
}()

func Constants(target string) map[string]Constant {
	if target == "samp" {
		return sampConstants
	}
	return openMPConstants
}

func UnsupportedFunctions(target string) map[string]UnsupportedFunction {
	if target == "samp" {
		return sampUnsupportedFunctions
	}
	return openMPUnsupportedFunctions
}

func DeprecatedFunctions(target string) map[string]DeprecatedFunction {
	if target == "samp" {
		return sampDeprecatedFunctions
	}
	return openMPDeprecatedFunctions
}

func LegacyIncludes() map[string]string {
	return map[string]string{
		"a_actor":    "open.mp",
		"a_http":     "open.mp",
		"a_objects":  "open.mp",
		"a_players":  "open.mp",
		"a_samp":     "open.mp",
		"a_sampdb":   "open.mp",
		"a_vehicles": "open.mp",
	}
}
