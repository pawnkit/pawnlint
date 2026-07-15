package externalrule

const ProtocolVersion = 1

type Request struct {
	ProtocolVersion  int            `json:"protocolVersion"`
	WorkingDirectory string         `json:"workingDirectory"`
	Build            string         `json:"build,omitempty"`
	Target           string         `json:"target,omitempty"`
	Defines          []string       `json:"defines,omitempty"`
	Configuration    map[string]any `json:"configuration,omitempty"`
	Targets          []string       `json:"targets"`
	Files            []File         `json:"files"`
}

type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type Response struct {
	ProtocolVersion int          `json:"protocolVersion"`
	Diagnostics     []Diagnostic `json:"diagnostics"`
}

type Diagnostic struct {
	RuleID      string            `json:"ruleId"`
	Code        string            `json:"code,omitempty"`
	Severity    string            `json:"severity"`
	Category    string            `json:"category"`
	Message     string            `json:"message"`
	Path        string            `json:"path"`
	StartOffset int               `json:"startOffset"`
	EndOffset   int               `json:"endOffset"`
	Related     []RelatedLocation `json:"related,omitempty"`
	Fix         *Fix              `json:"fix,omitempty"`
	Suggestions []Suggestion      `json:"suggestions,omitempty"`
}

type RelatedLocation struct {
	Path        string `json:"path"`
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
	Message     string `json:"message"`
}

type Edit struct {
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
	NewText     string `json:"newText"`
}

type Fix struct {
	Description string `json:"description"`
	Edits       []Edit `json:"edits"`
}

type Suggestion struct {
	Description string `json:"description"`
	Edits       []Edit `json:"edits,omitempty"`
}
