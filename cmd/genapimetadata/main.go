package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func main() {
	target := flag.String("target", "", "metadata target")
	source := flag.String("source", "", "stdlib directory")
	out := flag.String("out", "", "output Go file")
	flag.Parse()
	if *target == "" || *source == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "target, source, and out are required")
		os.Exit(2)
	}
	metadata, err := load(*target, *source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	generated, err := render(*target, metadata)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, generated, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type metadata struct {
	Callbacks           map[string]callback
	Natives             map[string]native
	Constants           map[string]constant
	Unsupported         map[string]unsupported
	DeprecatedFunctions map[string]deprecatedFunction
}

func load(target, root string) (metadata, error) {
	if !validTarget(target) {
		return metadata{}, fmt.Errorf("unknown target %q", target)
	}
	paths, err := filepath.Glob(filepath.Join(root, "*.inc"))
	if err != nil {
		return metadata{}, err
	}
	result := metadata{Callbacks: make(map[string]callback), Natives: make(map[string]native), Constants: make(map[string]constant), Unsupported: make(map[string]unsupported), DeprecatedFunctions: make(map[string]deprecatedFunction)}
	for _, path := range paths {
		base := filepath.Base(path)
		if !includeFile(target, base) {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return metadata{}, err
		}
		file := parser.Parse(src)
		if file == nil || file.Root == nil {
			return metadata{}, fmt.Errorf("parse %s", path)
		}
		if err := collectFile(file, path, &result); err != nil {
			return metadata{}, err
		}
	}
	if len(result.Callbacks) == 0 || len(result.Natives) == 0 {
		return metadata{}, fmt.Errorf("incomplete %s metadata in %s", target, root)
	}
	return result, nil
}

func includeFile(target, base string) bool {
	if target == "samp" {
		return strings.HasPrefix(base, "a_") && base != "a_npc.inc"
	}
	if strings.HasPrefix(base, "omp_") {
		return true
	}
	switch base {
	case "_open_mp.inc", "args.inc", "console.inc", "core.inc", "file.inc", "float.inc", "string.inc", "time.inc":
		return true
	default:
		return false
	}
}

func collectFile(file *parser.File, path string, result *metadata) error {
	base := filepath.Base(path)
	openMPOnly := strings.HasPrefix(base, "omp_") || base == "_open_mp.inc"
	var collect func([]*parser.Node, bool) error
	collect = func(nodes []*parser.Node, inFunction bool) error {
		deprecated := ""
		for _, node := range nodes {
			if !inFunction {
				collectConstants(file, node, openMPOnly, result.Constants)
			}
			if message, ok := deprecatedMessage(file, node); ok {
				deprecated = message
				continue
			}
			if node.Kind == parser.KindFunctionDeclaration {
				entry := readFunction(file, node)
				if openMPOnly && hasToken(node, token.KwForward) && entry.Name != "" && !strings.HasPrefix(entry.Name, "On") {
					result.Unsupported[entry.Name] = unsupported{Name: entry.Name, Suggested: deprecated}
				}
				if hasToken(node, token.KwForward) && strings.HasPrefix(entry.Name, "On") {
					callback := callback{Name: entry.Name, ReturnTag: entry.ReturnTag, Parameters: entry.Parameters}
					if existing, ok := result.Callbacks[entry.Name]; ok && !sameCallback(existing, callback) {
						return fmt.Errorf("conflicting callback %s in %s", entry.Name, path)
					}
					result.Callbacks[entry.Name] = callback
				}
				if hasToken(node, token.KwNative) && entry.Name != "" {
					entry.Deprecated = deprecated
					entry.OpenMPOnly = openMPOnly
					if existing, ok := result.Natives[entry.Name]; ok {
						if !sameNative(existing, entry) {
							return fmt.Errorf("conflicting native %s in %s", entry.Name, path)
						}
						if existing.Deprecated == "" {
							existing.Deprecated = entry.Deprecated
						}
						existing.OpenMPOnly = existing.OpenMPOnly && entry.OpenMPOnly
						result.Natives[entry.Name] = existing
					} else {
						result.Natives[entry.Name] = entry
					}
				}
				deprecated = ""
				continue
			}
			if node.Kind == parser.KindFunctionDefinition {
				entry := readFunction(file, node)
				if deprecated != "" && entry.Name != "" {
					result.DeprecatedFunctions[entry.Name] = deprecatedFunction{Name: entry.Name, Suggested: deprecated}
				}
				deprecated = ""
				continue
			}
			deprecated = ""
			if err := collect(node.Children, inFunction || node.Kind == parser.KindFunctionDefinition); err != nil {
				return err
			}
		}
		return nil
	}
	return collect(file.Root.Children, false)
}

type constant struct {
	Name       string
	OpenMPOnly bool
}

type unsupported struct {
	Name      string
	Suggested string
}

type deprecatedFunction struct {
	Name      string
	Suggested string
}

func collectConstants(file *parser.File, node *parser.Node, openMPOnly bool, constants map[string]constant) {
	var names []string
	switch node.Kind {
	case parser.KindDirectiveDefine:
		if node.Field("parameters") == nil && node.Field("value") != nil {
			names = append(names, text(file, node.Field("name")))
		}
	case parser.KindVariableDeclaration:
		if hasToken(node, token.KwConst) {
			for _, child := range node.Children {
				if child.Kind == parser.KindVariableDeclarator {
					names = append(names, text(file, child.Field("name")))
				}
			}
		}
	case parser.KindEnumEntry:
		names = append(names, text(file, node.Field("name")))
	}
	for _, name := range names {
		if name == "" {
			continue
		}
		entry := constant{Name: name, OpenMPOnly: openMPOnly}
		if existing, ok := constants[name]; ok {
			entry.OpenMPOnly = existing.OpenMPOnly && entry.OpenMPOnly
		}
		constants[name] = entry
	}
}

type callback struct {
	Name       string
	ReturnTag  string
	Parameters []parameter
}

type native struct {
	Name       string
	ReturnTag  string
	Parameters []parameter
	Deprecated string
	Format     int
	Buffers    []buffer
	OpenMPOnly bool
	Release    string
}

type buffer struct {
	Parameter     int
	SizeParameter int
}

type parameter struct {
	Name      string
	Tag       string
	ArrayRank int
	Const     bool
	Reference bool
	Variadic  bool
	Default   bool
}

func readFunction(file *parser.File, node *parser.Node) native {
	entry := native{Name: text(file, node.Field("name")), ReturnTag: tag(file, node)}
	entry.Release = resourceRelease(entry.Name)
	list := node.Field("parameters")
	if list == nil {
		return entry
	}
	for _, child := range list.Children {
		if child.Kind != parser.KindParameter {
			continue
		}
		param := parameter{
			Name:      text(file, child.Field("name")),
			Tag:       tag(file, child),
			ArrayRank: countKind(child, parser.KindDimension),
			Const:     hasToken(child, token.KwConst),
			Reference: hasTokenBefore(file, child, child.Field("name"), token.Amp),
			Variadic:  child.Field("name") == nil && hasTokenWithin(file, child, token.Ellipsis),
			Default:   child.Field("default_value") != nil,
		}
		entry.Parameters = append(entry.Parameters, param)
	}
	entry.Buffers = readBuffers(file, list)
	return entry
}

func resourceRelease(name string) string {
	switch name {
	case "fopen", "ftemp":
		return "fclose"
	case "DB_Open":
		return "DB_Close"
	case "db_open":
		return "db_close"
	case "DB_ExecuteQuery":
		return "DB_FreeResultSet"
	case "db_query":
		return "db_free_result"
	default:
		return ""
	}
}

func readBuffers(file *parser.File, list *parser.Node) []buffer {
	var parameters []*parser.Node
	for _, child := range list.Children {
		if child.Kind == parser.KindParameter {
			parameters = append(parameters, child)
		}
	}
	var result []buffer
	for i, parameter := range parameters {
		name := text(file, parameter.Field("name"))
		if name == "" || countKind(parameter, parser.KindDimension) != 1 || hasToken(parameter, token.KwConst) {
			continue
		}
		for j, candidate := range parameters {
			if j <= i {
				continue
			}
			value := candidate.Field("default_value")
			if value == nil || value.Kind != parser.KindSizeofExpression {
				continue
			}
			expression := value.Field("expression")
			if expression != nil && expression.Kind == parser.KindIdentifier && text(file, expression) == name {
				result = append(result, buffer{Parameter: i + 1, SizeParameter: j + 1})
				break
			}
		}
	}
	return result
}

func render(target string, data metadata) ([]byte, error) {
	if !validTarget(target) {
		return nil, fmt.Errorf("unknown target %q", target)
	}
	name := "openMPCallbacks"
	if target == "samp" {
		name = "sampCallbacks"
	}
	names := make([]string, 0, len(data.Callbacks))
	for callbackName := range data.Callbacks {
		names = append(names, callbackName)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("package api\n\nvar " + name + " = map[string]Callback{\n")
	for _, callbackName := range names {
		entry := data.Callbacks[callbackName]
		fmt.Fprintf(&b, "%s: {Name: %s, ReturnTag: %s", strconv.Quote(callbackName), strconv.Quote(entry.Name), strconv.Quote(entry.ReturnTag))
		if len(entry.Parameters) > 0 {
			b.WriteString(", Parameters: []Parameter{")
			for _, param := range entry.Parameters {
				writeParameter(&b, param)
			}
			b.WriteString("}")
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n")
	deprecatedMap := "openMPDeprecatedFunctions"
	if target == "samp" {
		deprecatedMap = "sampDeprecatedFunctions"
	}
	names = names[:0]
	for functionName := range data.DeprecatedFunctions {
		names = append(names, functionName)
	}
	sort.Strings(names)
	b.WriteString("\nvar " + deprecatedMap + " = map[string]DeprecatedFunction{\n")
	for _, functionName := range names {
		entry := data.DeprecatedFunctions[functionName]
		fmt.Fprintf(&b, "%s: {Name: %s, Suggested: %s},\n", strconv.Quote(functionName), strconv.Quote(entry.Name), strconv.Quote(entry.Suggested))
	}
	b.WriteString("}\n")
	unsupportedMap := "openMPUnsupportedFunctions"
	if target == "samp" {
		unsupportedMap = "sampUnsupportedFunctions"
	}
	names = names[:0]
	for functionName := range data.Unsupported {
		names = append(names, functionName)
	}
	sort.Strings(names)
	b.WriteString("\nvar " + unsupportedMap + " = map[string]UnsupportedFunction{\n")
	for _, functionName := range names {
		entry := data.Unsupported[functionName]
		fmt.Fprintf(&b, "%s: {Name: %s", strconv.Quote(functionName), strconv.Quote(entry.Name))
		if entry.Suggested != "" {
			fmt.Fprintf(&b, ", Suggested: %s", strconv.Quote(entry.Suggested))
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n")
	constantMap := "openMPConstants"
	if target == "samp" {
		constantMap = "sampConstants"
	}
	names = names[:0]
	for constantName := range data.Constants {
		names = append(names, constantName)
	}
	sort.Strings(names)
	b.WriteString("\nvar " + constantMap + " = map[string]Constant{\n")
	for _, constantName := range names {
		entry := data.Constants[constantName]
		fmt.Fprintf(&b, "%s: {Name: %s", strconv.Quote(constantName), strconv.Quote(entry.Name))
		if entry.OpenMPOnly {
			b.WriteString(", OpenMPOnly: true")
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n")
	nativeMap := "openMPNatives"
	if target == "samp" {
		nativeMap = "sampNatives"
	}
	names = names[:0]
	for nativeName := range data.Natives {
		names = append(names, nativeName)
	}
	sort.Strings(names)
	b.WriteString("\nvar " + nativeMap + " = map[string]Native{\n")
	for _, nativeName := range names {
		entry := data.Natives[nativeName]
		fmt.Fprintf(&b, "%s: {Name: %s, ReturnTag: %s", strconv.Quote(nativeName), strconv.Quote(entry.Name), strconv.Quote(entry.ReturnTag))
		if len(entry.Parameters) > 0 {
			b.WriteString(", Parameters: []Parameter{")
			for _, param := range entry.Parameters {
				writeParameter(&b, param)
			}
			b.WriteString("}")
		}
		if entry.Deprecated != "" {
			fmt.Fprintf(&b, ", Deprecated: %s", strconv.Quote(entry.Deprecated))
		}
		if format := formatParameter(entry); format != 0 {
			fmt.Fprintf(&b, ", FormatParameter: %d", format)
		}
		if len(entry.Buffers) != 0 {
			b.WriteString(", Buffers: []Buffer{")
			for _, relation := range entry.Buffers {
				fmt.Fprintf(&b, "{Parameter: %d, SizeParameter: %d},", relation.Parameter, relation.SizeParameter)
			}
			b.WriteString("}")
		}
		if entry.OpenMPOnly {
			b.WriteString(", OpenMPOnly: true")
		}
		if entry.Release != "" {
			fmt.Fprintf(&b, ", Release: %s", strconv.Quote(entry.Release))
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n")
	return format.Source([]byte(b.String()))
}

func formatParameter(entry native) int {
	switch entry.Name {
	case "CallLocalFunction", "CallRemoteFunction", "SetTimerEx":
		return 0
	}
	variadic := false
	for _, parameter := range entry.Parameters {
		variadic = variadic || parameter.Variadic
	}
	if !variadic {
		return 0
	}
	for i, parameter := range entry.Parameters {
		if parameter.Name == "format" {
			return i + 1
		}
	}
	return 0
}

func writeParameter(b *strings.Builder, param parameter) {
	fmt.Fprintf(b, "{Name: %s, Tag: %s, ArrayRank: %d, Const: %t, Reference: %t, Variadic: %t, Default: %t},", strconv.Quote(param.Name), strconv.Quote(param.Tag), param.ArrayRank, param.Const, param.Reference, param.Variadic, param.Default)
}

func validTarget(target string) bool {
	return target == "openmp" || target == "samp"
}

func sameCallback(left, right callback) bool {
	if left.Name != right.Name || left.ReturnTag != right.ReturnTag || len(left.Parameters) != len(right.Parameters) {
		return false
	}
	for i := range left.Parameters {
		if left.Parameters[i] != right.Parameters[i] {
			return false
		}
	}
	return true
}

func sameNative(left, right native) bool {
	if left.Name != right.Name || left.ReturnTag != right.ReturnTag || len(left.Parameters) != len(right.Parameters) {
		return false
	}
	for i := range left.Parameters {
		a, b := left.Parameters[i], right.Parameters[i]
		if a.Tag != b.Tag || a.ArrayRank != b.ArrayRank || a.Const != b.Const || a.Reference != b.Reference || a.Variadic != b.Variadic || a.Default != b.Default {
			return false
		}
	}
	return true
}

func deprecatedMessage(file *parser.File, node *parser.Node) (string, bool) {
	if node.Kind != parser.KindDirectivePragma {
		return "", false
	}
	value := strings.TrimSpace(node.Text(file.Source))
	const prefix = "#pragma deprecated"
	if !strings.HasPrefix(value, prefix) {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix)), true
}

func text(file *parser.File, node *parser.Node) string {
	if node == nil {
		return ""
	}
	return node.Text(file.Source)
}

func tag(file *parser.File, node *parser.Node) string {
	tagNode := node.Field("tag")
	if tagNode == nil || len(tagNode.Children) != 1 {
		return ""
	}
	return text(file, tagNode.Children[0])
}

func countKind(node *parser.Node, kind parser.Kind) int {
	count := 0
	for _, child := range node.Children {
		if child.Kind == kind {
			count++
		}
	}
	return count
}

func hasToken(node *parser.Node, kind token.Kind) bool {
	for _, child := range node.Children {
		if child.Tok.Kind == kind {
			return true
		}
	}
	return false
}

func hasTokenBefore(file *parser.File, node, endNode *parser.Node, kind token.Kind) bool {
	end := node.End
	if endNode != nil {
		end = endNode.Start
	}
	for _, tok := range file.Tokens {
		if tok.Start.Offset >= node.Start && tok.End.Offset <= end && tok.Kind == kind {
			return true
		}
	}
	return false
}

func hasTokenWithin(file *parser.File, node *parser.Node, kind token.Kind) bool {
	for _, tok := range file.Tokens {
		if tok.Start.Offset >= node.Start && tok.End.Offset <= node.End && tok.Kind == kind {
			return true
		}
	}
	return false
}
