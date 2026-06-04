// grok gen — scaffolds a new .grok file from an existing Go package.
// Extracts all exported types into Grok structural declarations (Zone 1),
// then generates the function index (Zone 2) and dependencies (Zone 3).
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
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: grok gen <package-dir>\n")
		os.Exit(1)
	}

	pkgDir := os.Args[1]
	if err := run(pkgDir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(pkgDir string) error {
	absDir, err := filepath.Abs(pkgDir)
	if err != nil {
		return err
	}

	goFiles, err := scanGoFiles(absDir)
	if err != nil {
		return err
	}
	if len(goFiles) == 0 {
		return fmt.Errorf("no .go files found in %s", pkgDir)
	}

	fset := token.NewFileSet()
	var allFiles []*goast.File
	var fileNames []string
	for _, name := range goFiles {
		f, err := goparser.ParseFile(fset, filepath.Join(absDir, name), nil, goparser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", name, err)
			continue
		}
		allFiles = append(allFiles, f)
		fileNames = append(fileNames, name)
	}

	if len(allFiles) == 0 {
		return fmt.Errorf("no parseable .go files")
	}

	pkgName := allFiles[0].Name.Name

	// Extract types and functions
	var structs []structInfo
	var interfaces []ifaceInfo
	var typeAliases []aliasInfo
	var funcs []funcEntry
	importSet := make(map[string]bool)

	for i, f := range allFiles {
		baseName := fileNames[i]

		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *goast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *goast.TypeSpec:
						if !isExported(s.Name.Name) {
							continue
						}
						switch t := s.Type.(type) {
						case *goast.StructType:
							structs = append(structs, extractStruct(s.Name.Name, t, s.TypeParams))
						case *goast.InterfaceType:
							interfaces = append(interfaces, extractInterface(s.Name.Name, t, s.TypeParams))
						default:
							// Type alias or named type
							typeAliases = append(typeAliases, aliasInfo{
								Name: s.Name.Name,
								Type: goTypeToGrok(typeString(s.Type)),
							})
						}
					}
				}
			case *goast.FuncDecl:
				entry := funcEntry{
					File: baseName,
					Line: fset.Position(d.Pos()).Line,
				}
				if d.Doc != nil && len(d.Doc.List) > 0 {
					entry.DocLine = strings.TrimPrefix(d.Doc.List[0].Text, "// ")
					entry.DocLine = strings.TrimPrefix(entry.DocLine, "/* ")
				}
				entry.Signature = buildSignature(d, fset)
				funcs = append(funcs, entry)
			}
		}

		for _, imp := range f.Imports {
			importSet[strings.Trim(imp.Path.Value, `"`)] = true
		}
	}

	// Sort functions by file then line
	sort.Slice(funcs, func(i, j int) bool {
		if funcs[i].File != funcs[j].File {
			return funcs[i].File < funcs[j].File
		}
		return funcs[i].Line < funcs[j].Line
	})

	// Build output
	var b strings.Builder

	// Zone 1 — scaffold
	blockName := exportName(pkgName)
	b.WriteString(fmt.Sprintf("// %s.grok — structural understanding of the %s package\n\n", pkgName, pkgName))
	b.WriteString(fmt.Sprintf("grok %s {\n", blockName))
	b.WriteString("  why: \"\"\n\n")

	// Structs
	for _, s := range structs {
		b.WriteString(fmt.Sprintf("  struct %s", s.Name))
		if s.TypeParams != "" {
			b.WriteString(fmt.Sprintf("<%s>", s.TypeParams))
		}
		b.WriteString(" {\n")
		for _, f := range s.Fields {
			b.WriteString(fmt.Sprintf("    %s: %s\n", f.Name, f.Type))
		}
		b.WriteString("  }\n\n")
	}

	// Interfaces
	for _, iface := range interfaces {
		b.WriteString(fmt.Sprintf("  interface %s", iface.Name))
		if iface.TypeParams != "" {
			b.WriteString(fmt.Sprintf("<%s>", iface.TypeParams))
		}
		b.WriteString(" {\n")
		for _, m := range iface.Methods {
			b.WriteString(fmt.Sprintf("    %s\n", m))
		}
		b.WriteString("  }\n\n")
	}

	// Type aliases
	for _, a := range typeAliases {
		b.WriteString(fmt.Sprintf("  type %s = %s\n\n", a.Name, a.Type))
	}

	// Source annotation
	b.WriteString(fmt.Sprintf("  source: [%s]\n", formatSourceList(fileNames)))
	b.WriteString("}\n")

	// Zone 2
	b.WriteString(buildFunctionIndex(funcs, fileNames))

	// Zone 3
	modPath := findModulePath(absDir)
	b.WriteString(buildDependencies(importSet, modPath))

	// Write file
	outPath := filepath.Join(absDir, pkgName+".grok")
	if _, err := os.Stat(outPath); err == nil {
		return fmt.Errorf("%s already exists — use grok update instead", outPath)
	}

	fmt.Printf("Generated %s\n", outPath)
	return os.WriteFile(outPath, []byte(b.String()), 0644)
}

// --- Types ---

type fieldInfo struct {
	Name string
	Type string
}

type structInfo struct {
	Name       string
	TypeParams string
	Fields     []fieldInfo
}

type ifaceInfo struct {
	Name       string
	TypeParams string
	Methods    []string // formatted method signatures in Grok syntax
}

type aliasInfo struct {
	Name string
	Type string
}

type funcEntry struct {
	File      string
	Line      int
	Signature string
	DocLine   string
}

// --- Extraction ---

func extractStruct(name string, st *goast.StructType, typeParams *goast.FieldList) structInfo {
	s := structInfo{Name: name}
	if typeParams != nil {
		s.TypeParams = grokTypeParams(typeParams)
	}
	if st.Fields != nil {
		for _, f := range st.Fields.List {
			grokType := goTypeToGrok(typeString(f.Type))
			if len(f.Names) == 0 {
				// Embedded field
				s.Fields = append(s.Fields, fieldInfo{
					Name: typeString(f.Type),
					Type: grokType,
				})
			} else {
				for _, n := range f.Names {
					if !isExported(n.Name) {
						continue
					}
					s.Fields = append(s.Fields, fieldInfo{
						Name: n.Name,
						Type: grokType,
					})
				}
			}
		}
	}
	return s
}

func extractInterface(name string, it *goast.InterfaceType, typeParams *goast.FieldList) ifaceInfo {
	iface := ifaceInfo{Name: name}
	if typeParams != nil {
		iface.TypeParams = grokTypeParams(typeParams)
	}
	if it.Methods != nil {
		for _, m := range it.Methods.List {
			if len(m.Names) == 0 {
				// Embedded interface
				iface.Methods = append(iface.Methods, typeString(m.Type))
				continue
			}
			ft, ok := m.Type.(*goast.FuncType)
			if !ok {
				continue
			}
			sig := formatGrokMethod(m.Names[0].Name, ft)
			iface.Methods = append(iface.Methods, sig)
		}
	}
	return iface
}

func formatGrokMethod(name string, ft *goast.FuncType) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("func %s(", name))

	if ft.Params != nil {
		var params []string
		for _, p := range ft.Params.List {
			grokType := goTypeToGrok(typeString(p.Type))
			if len(p.Names) == 0 {
				params = append(params, grokType)
			} else {
				for _, n := range p.Names {
					params = append(params, fmt.Sprintf("%s: %s", n.Name, grokType))
				}
			}
		}
		b.WriteString(strings.Join(params, ", "))
	}
	b.WriteString(")")

	if ft.Results != nil && len(ft.Results.List) > 0 {
		results := ft.Results.List
		if len(results) == 1 && len(results[0].Names) == 0 {
			b.WriteString(" -> ")
			b.WriteString(goTypeToGrok(typeString(results[0].Type)))
		} else {
			b.WriteString(" -> (")
			var rets []string
			for _, r := range results {
				rets = append(rets, goTypeToGrok(typeString(r.Type)))
			}
			b.WriteString(strings.Join(rets, ", "))
			b.WriteString(")")
		}
	}

	return b.String()
}

func grokTypeParams(fl *goast.FieldList) string {
	var parts []string
	for _, f := range fl.List {
		constraint := typeString(f.Type)
		for _, n := range f.Names {
			if constraint != "" && constraint != "any" {
				parts = append(parts, fmt.Sprintf("%s: %s", n.Name, constraint))
			} else {
				parts = append(parts, n.Name)
			}
		}
	}
	return strings.Join(parts, ", ")
}

// --- Go → Grok type conversion ---

func goTypeToGrok(goType string) string {
	// Reverse of grokNameToGo
	switch goType {
	case "int8":
		return "i8"
	case "int16":
		return "i16"
	case "int32":
		return "i32"
	case "int64":
		return "i64"
	case "uint8":
		return "u8"
	case "uint16":
		return "u16"
	case "uint32":
		return "u32"
	case "uint64":
		return "u64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	case "interface{}":
		return "any"
	case "string", "bool", "int", "error", "any":
		return goType
	}

	// Pointer → optional
	if strings.HasPrefix(goType, "*") {
		return goTypeToGrok(goType[1:]) + "?"
	}
	// Slice → list
	if strings.HasPrefix(goType, "[]") {
		return "[" + goTypeToGrok(goType[2:]) + "]"
	}
	// Map
	if strings.HasPrefix(goType, "map[") {
		// Find the closing bracket
		depth := 0
		for i, c := range goType {
			if c == '[' {
				depth++
			} else if c == ']' {
				depth--
				if depth == 0 {
					key := goType[4:i]
					val := goType[i+1:]
					return fmt.Sprintf("map[%s]%s", goTypeToGrok(key), goTypeToGrok(val))
				}
			}
		}
	}

	return goType
}

// --- Shared utilities (duplicated from grok-update for now) ---

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

func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

func exportName(name string) string {
	if len(name) == 0 {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func formatSourceList(files []string) string {
	var quoted []string
	for _, f := range files {
		quoted = append(quoted, fmt.Sprintf("%q", f))
	}
	return strings.Join(quoted, ", ")
}

func buildSignature(fn *goast.FuncDecl, fset *token.FileSet) string {
	var b strings.Builder
	b.WriteString("func ")

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

	b.WriteString(fn.Name.Name)

	if fn.Type.TypeParams != nil && len(fn.Type.TypeParams.List) > 0 {
		b.WriteString("[")
		b.WriteString(fieldListString(fn.Type.TypeParams))
		b.WriteString("]")
	}

	b.WriteString("(")
	if fn.Type.Params != nil {
		b.WriteString(fieldListString(fn.Type.Params))
	}
	b.WriteString(")")

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

func buildFunctionIndex(funcs []funcEntry, sourceFiles []string) string {
	var b strings.Builder
	b.WriteString("// --- index ---\n")
	b.WriteString("//\n")
	b.WriteString("// Auto-generated function/method index with signatures in Go syntax.\n")
	b.WriteString("// DO NOT EDIT below this line — regenerated by `grok update`.\n")

	byFile := make(map[string][]funcEntry)
	for _, f := range funcs {
		byFile[f.File] = append(byFile[f.File], f)
	}

	for _, file := range sourceFiles {
		entries := byFile[file]
		b.WriteString("//\n")
		b.WriteString(fmt.Sprintf("// file: %s\n", file))
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

func buildDependencies(imports map[string]bool, modPath string) string {
	if modPath == "" {
		return ""
	}
	var internal []string
	for imp := range imports {
		if strings.HasPrefix(imp, modPath+"/") {
			internal = append(internal, strings.TrimPrefix(imp, modPath+"/"))
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

func findModulePath(dir string) string {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		data, err := os.ReadFile(filepath.Join(absDir, "go.mod"))
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
