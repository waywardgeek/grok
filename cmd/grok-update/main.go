// grok update — regenerates Zone 2 (function index) and Zone 3 (dependencies)
// of a .grok file while preserving Zone 1 (human-curated) verbatim.
package main

import (
	"fmt"
	goast "go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/waywardgeek/grok/pkg/parser"
)

// funcEntry holds extracted info about a single Go function or method.
type funcEntry struct {
	File      string // source filename (basename)
	Line      int
	Signature string // full Go signature line
	DocLine   string // first line of doc comment, empty if none
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: grok update <file.grok>\n")
		os.Exit(1)
	}

	grokPath := os.Args[1]
	if err := run(grokPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(grokPath string) error {
	raw, err := os.ReadFile(grokPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", grokPath, err)
	}

	// Split at Zone 2 marker
	zone1, _ := splitAtMarker(string(raw), "// --- index ---")

	// Parse Zone 1 to extract source: paths
	sources, err := extractSourcePaths(grokPath)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		// Fallback: scan .grok file's directory for .go files
		sources, err = scanGoFiles(filepath.Dir(grokPath))
		if err != nil {
			return err
		}
	}

	grokDir := filepath.Dir(grokPath)

	// Find module path for dependency filtering
	modPath := findModulePath(grokDir)

	// Extract functions and imports from all source files
	var allFuncs []funcEntry
	importSet := make(map[string]bool)

	for _, src := range sources {
		fullPath := filepath.Join(grokDir, src)
		funcs, imports, err := extractFromGoFile(fullPath, src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", src, err)
			continue
		}
		allFuncs = append(allFuncs, funcs...)
		for _, imp := range imports {
			importSet[imp] = true
		}
	}

	// Build Zone 2
	zone2 := buildFunctionIndex(allFuncs, sources)

	// Build Zone 3
	zone3 := buildDependencies(importSet, modPath)

	// Write output
	output := zone1 + zone2 + zone3
	return os.WriteFile(grokPath, []byte(output), 0644)
}

// splitAtMarker splits text at the first occurrence of marker.
// Returns (before, after). If marker not found, returns (text, "").
func splitAtMarker(text, marker string) (string, string) {
	idx := strings.Index(text, marker)
	if idx < 0 {
		return text, ""
	}
	return text[:idx], text[idx:]
}

// extractSourcePaths parses the .grok file and collects all source: annotations.
func extractSourcePaths(grokPath string) ([]string, error) {
	src, err := os.ReadFile(grokPath)
	if err != nil {
		return nil, err
	}
	// Only parse Zone 1
	zone1, _ := splitAtMarker(string(src), "// --- index ---")
	f, err := parser.ParseString(zone1)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", grokPath, err)
	}

	var sources []string
	seen := make(map[string]bool)
	for _, block := range f.Blocks {
		for _, s := range block.Source {
			if !seen[s] {
				sources = append(sources, s)
				seen[s] = true
			}
		}
	}
	return sources, nil
}

// scanGoFiles returns all .go files (excluding _test.go) in a directory.
func scanGoFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files, nil
}

// extractFromGoFile parses a Go file and returns function entries and import paths.
func extractFromGoFile(fullPath, baseName string) ([]funcEntry, []string, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, fullPath, nil, goparser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	var funcs []funcEntry
	for _, decl := range f.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok {
			continue
		}

		entry := funcEntry{
			File: baseName,
			Line: fset.Position(fn.Pos()).Line,
		}

		// Extract first line of doc comment
		if fn.Doc != nil && len(fn.Doc.List) > 0 {
			entry.DocLine = strings.TrimPrefix(fn.Doc.List[0].Text, "// ")
			entry.DocLine = strings.TrimPrefix(entry.DocLine, "/* ")
		}

		// Build signature
		entry.Signature = buildSignature(fn, fset)
		funcs = append(funcs, entry)
	}

	// Sort by line number
	sort.Slice(funcs, func(i, j int) bool { return funcs[i].Line < funcs[j].Line })

	// Collect imports
	var imports []string
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, path)
	}

	return funcs, imports, nil
}

// buildSignature constructs a Go function signature string from AST.
func buildSignature(fn *goast.FuncDecl, fset *token.FileSet) string {
	var b strings.Builder
	b.WriteString("func ")

	// Receiver
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := fn.Recv.List[0]
		b.WriteString("(")
		if len(recv.Names) > 0 {
			b.WriteString(recv.Names[0].Name)
			b.WriteString(" ")
		}
		b.WriteString(typeString(recv.Type))
		b.WriteString(") ")
	}

	// Name
	b.WriteString(fn.Name.Name)

	// Type params
	if fn.Type.TypeParams != nil && len(fn.Type.TypeParams.List) > 0 {
		b.WriteString("[")
		b.WriteString(fieldListString(fn.Type.TypeParams))
		b.WriteString("]")
	}

	// Params
	b.WriteString("(")
	if fn.Type.Params != nil {
		b.WriteString(fieldListString(fn.Type.Params))
	}
	b.WriteString(")")

	// Returns
	if fn.Type.Results != nil {
		results := fn.Type.Results.List
		if len(results) == 1 && len(results[0].Names) == 0 {
			b.WriteString(" ")
			b.WriteString(typeString(results[0].Type))
		} else if len(results) > 0 {
			b.WriteString(" (")
			b.WriteString(fieldListString(fn.Type.Results))
			b.WriteString(")")
		}
	}

	return b.String()
}

// fieldListString formats a field list as comma-separated params.
func fieldListString(fl *goast.FieldList) string {
	var parts []string
	for _, f := range fl.List {
		ts := typeString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, ts)
		} else {
			var names []string
			for _, n := range f.Names {
				names = append(names, n.Name)
			}
			parts = append(parts, strings.Join(names, ", ")+" "+ts)
		}
	}
	return strings.Join(parts, ", ")
}

// typeString converts a Go AST type expression to a string.
func typeString(expr goast.Expr) string {
	switch t := expr.(type) {
	case *goast.Ident:
		return t.Name
	case *goast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *goast.StarExpr:
		return "*" + typeString(t.X)
	case *goast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(t.Elt)
		}
		return "[" + exprString(t.Len) + "]" + typeString(t.Elt)
	case *goast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *goast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{...}"
	case *goast.FuncType:
		var b strings.Builder
		b.WriteString("func(")
		if t.Params != nil {
			b.WriteString(fieldListString(t.Params))
		}
		b.WriteString(")")
		if t.Results != nil {
			results := t.Results.List
			if len(results) == 1 && len(results[0].Names) == 0 {
				b.WriteString(" ")
				b.WriteString(typeString(results[0].Type))
			} else if len(results) > 0 {
				b.WriteString(" (")
				b.WriteString(fieldListString(t.Results))
				b.WriteString(")")
			}
		}
		return b.String()
	case *goast.Ellipsis:
		return "..." + typeString(t.Elt)
	case *goast.ChanType:
		switch t.Dir {
		case goast.SEND:
			return "chan<- " + typeString(t.Value)
		case goast.RECV:
			return "<-chan " + typeString(t.Value)
		default:
			return "chan " + typeString(t.Value)
		}
	case *goast.IndexExpr:
		return typeString(t.X) + "[" + typeString(t.Index) + "]"
	case *goast.IndexListExpr:
		var indices []string
		for _, idx := range t.Indices {
			indices = append(indices, typeString(idx))
		}
		return typeString(t.X) + "[" + strings.Join(indices, ", ") + "]"
	case *goast.StructType:
		return "struct{...}"
	case *goast.ParenExpr:
		return "(" + typeString(t.X) + ")"
	default:
		return fmt.Sprintf("<%T>", expr)
	}
}

// exprString formats a basic expression (used for array lengths).
func exprString(expr goast.Expr) string {
	switch e := expr.(type) {
	case *goast.BasicLit:
		return e.Value
	case *goast.Ident:
		return e.Name
	default:
		return "..."
	}
}

// buildFunctionIndex formats Zone 2 output.
func buildFunctionIndex(funcs []funcEntry, sourceFiles []string) string {
	var b strings.Builder
	b.WriteString("// --- index ---\n")
	b.WriteString("//\n")
	b.WriteString("// Auto-generated function/method index with signatures in Go syntax.\n")
	b.WriteString("// DO NOT EDIT below this line — regenerated by `grok update`.\n")

	// Group by file, maintaining source order
	fileOrder := sourceFiles
	byFile := make(map[string][]funcEntry)
	for _, f := range funcs {
		byFile[f.File] = append(byFile[f.File], f)
	}

	for _, file := range fileOrder {
		entries := byFile[file]
		b.WriteString("//\n")
		b.WriteString(fmt.Sprintf("// file: %s\n", file))

		if len(entries) == 0 {
			continue
		}

		for _, e := range entries {
			b.WriteString("//\n")
			if e.DocLine != "" {
				b.WriteString(fmt.Sprintf("// %s\n", e.DocLine))
			}
			b.WriteString(fmt.Sprintf("// %s    :%d\n", e.Signature, e.Line))
		}
	}

	b.WriteString("\n")
	return b.String()
}

// buildDependencies formats Zone 3 output.
func buildDependencies(imports map[string]bool, modPath string) string {
	if modPath == "" {
		return ""
	}

	var internal []string
	for imp := range imports {
		if strings.HasPrefix(imp, modPath+"/") {
			// Strip module prefix
			rel := strings.TrimPrefix(imp, modPath+"/")
			internal = append(internal, rel)
		}
	}

	if len(internal) == 0 {
		return ""
	}

	sort.Strings(internal)

	var b strings.Builder
	b.WriteString("// --- dependencies ---\n")
	b.WriteString("//\n")
	b.WriteString("// Auto-generated internal package dependencies.\n")
	b.WriteString("// DO NOT EDIT below this line — regenerated by `grok update`.\n")
	b.WriteString("//\n")
	for _, dep := range internal {
		b.WriteString(fmt.Sprintf("// %s\n", dep))
	}

	return b.String()
}

// findModulePath walks up from dir looking for go.mod and extracts the module path.
func findModulePath(dir string) string {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}

	for {
		modFile := filepath.Join(absDir, "go.mod")
		data, err := os.ReadFile(modFile)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "module "))
				}
			}
		}
		parent := filepath.Dir(absDir)
		if parent == absDir {
			return ""
		}
		absDir = parent
	}
}
