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

	grokast "github.com/waywardgeek/grok/pkg/ast"
	"github.com/waywardgeek/grok/pkg/parser"
)

// runUpdate parses a .grok file, auto-updates Zone 1 with missing exported
// symbols from Go source, then regenerates Zone 2 (function index) and
// Zone 3 (dependencies).
func runUpdate(grokPath string, prune bool) error {
	raw, err := os.ReadFile(grokPath)
	if err != nil {
		return err
	}

	zone1, _ := splitAtMarker(string(raw), "// --- index ---")

	// Parse Zone 1
	grokFile, err := parser.ParseFile(zone1, grokPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", grokPath, err)
	}

	if len(grokFile.Blocks) == 0 {
		return fmt.Errorf("%s: no grok blocks found", grokPath)
	}

	grokDir := filepath.Dir(grokPath)

	// Collect source paths from all blocks
	var sources []string
	seen := make(map[string]bool)
	for _, block := range grokFile.Blocks {
		for _, s := range block.Source {
			if !seen[s] {
				sources = append(sources, s)
				seen[s] = true
			}
		}
	}
	if len(sources) == 0 {
		sources, err = scanGoFiles(grokDir)
		if err != nil {
			return err
		}
	}

	// Parse Go source files
	fset := token.NewFileSet()
	var goFiles []*goast.File
	for _, src := range sources {
		fullPath := filepath.Join(grokDir, src)
		f, parseErr := goparser.ParseFile(fset, fullPath, nil, goparser.ParseComments)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", src, parseErr)
			continue
		}
		goFiles = append(goFiles, f)
	}

	// Extract Go type info
	goInfo := extractGoInfo(goFiles, fset)

	// Auto-update Zone 1: add missing exported symbols to the first block
	block := &grokFile.Blocks[0]
	addMissingSymbols(block, goInfo)

	// Optionally remove declarations not found in Go source
	if prune {
		pruneStaleSymbols(block, goInfo)
	}

	// Re-emit Zone 1 via formatter
	var b strings.Builder
	commentIdx := 0
	comments := grokFile.Comments

	for bi := range grokFile.Blocks {
		blk := &grokFile.Blocks[bi]
		commentIdx = emitCommentsBefore(&b, comments, commentIdx, blk.Span.Start.Line, "")
		fmtBlock(&b, blk, comments, &commentIdx)
		if bi < len(grokFile.Blocks)-1 {
			b.WriteString("\n")
		}
	}
	for commentIdx < len(comments) {
		b.WriteString(comments[commentIdx].Text)
		b.WriteString("\n")
		commentIdx++
	}

	// Build Zone 2 (function index)
	var allFuncs []funcEntry
	importSet := make(map[string]bool)
	for _, src := range sources {
		fullPath := filepath.Join(grokDir, src)
		funcs, imports, extractErr := extractFuncsFromGoFile(fullPath, filepath.Base(src))
		if extractErr != nil {
			continue
		}
		allFuncs = append(allFuncs, funcs...)
		for _, imp := range imports {
			importSet[imp] = true
		}
	}

	zone2 := buildFunctionIndex(allFuncs, sources)

	// Build Zone 3 (dependencies)
	modPath := findModulePath(grokDir)
	zone3 := buildDependencies(importSet, modPath)

	return os.WriteFile(grokPath, []byte(b.String()+zone2+zone3), 0644)
}

// --- Go type extraction ---

type goInfo struct {
	Structs    map[string]*goStructDef
	Interfaces map[string]*goIfaceDef
	Functions  map[string]*goFuncDef
	TypeDefs   map[string]string // name → underlying Go type string
}

type goStructDef struct {
	Fields  map[string]string          // field name → Go type string
	Methods map[string]*goFuncDef
}

type goIfaceDef struct {
	Methods    map[string]*goFuncDef
	Embeds     []string // embedded interface names
}

type goFuncDef struct {
	Params  []goParamDef
	Returns []string // Go type strings
}

type goParamDef struct {
	Name string
	Type string
}

func extractGoInfo(files []*goast.File, fset *token.FileSet) *goInfo {
	info := &goInfo{
		Structs:    make(map[string]*goStructDef),
		Interfaces: make(map[string]*goIfaceDef),
		Functions:  make(map[string]*goFuncDef),
		TypeDefs:   make(map[string]string),
	}

	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *goast.GenDecl:
				for _, spec := range d.Specs {
					ts, ok := spec.(*goast.TypeSpec)
					if !ok || !isExported(ts.Name.Name) {
						continue
					}
					switch t := ts.Type.(type) {
					case *goast.StructType:
						sd := &goStructDef{
							Fields:  make(map[string]string),
							Methods: make(map[string]*goFuncDef),
						}
						if t.Fields != nil {
							for _, field := range t.Fields.List {
								typStr := typeString(field.Type)
								for _, name := range field.Names {
									if isExported(name.Name) {
										sd.Fields[name.Name] = typStr
									}
								}
							}
						}
						if existing, ok := info.Structs[ts.Name.Name]; ok {
							// Merge fields
							for k, v := range sd.Fields {
								existing.Fields[k] = v
							}
						} else {
							info.Structs[ts.Name.Name] = sd
						}

					case *goast.InterfaceType:
						id := &goIfaceDef{
							Methods: make(map[string]*goFuncDef),
						}
						if t.Methods != nil {
							for _, m := range t.Methods.List {
								if len(m.Names) == 0 {
									// Embedded interface
									id.Embeds = append(id.Embeds, typeString(m.Type))
									continue
								}
								if ft, ok := m.Type.(*goast.FuncType); ok {
									for _, name := range m.Names {
										id.Methods[name.Name] = extractGoFuncType(ft)
									}
								}
							}
						}
						info.Interfaces[ts.Name.Name] = id

					default:
						info.TypeDefs[ts.Name.Name] = typeString(ts.Type)
					}
				}

			case *goast.FuncDecl:
				if !isExported(d.Name.Name) {
					continue
				}
				fd := extractGoFuncType(d.Type)
				if d.Recv != nil && len(d.Recv.List) > 0 {
					recvName := receiverTypeName(d.Recv.List[0].Type)
					if recvName != "" {
						sd, ok := info.Structs[recvName]
						if !ok {
							sd = &goStructDef{
								Fields:  make(map[string]string),
								Methods: make(map[string]*goFuncDef),
							}
							info.Structs[recvName] = sd
						}
						sd.Methods[d.Name.Name] = fd
					}
				} else {
					info.Functions[d.Name.Name] = fd
				}
			}
		}
	}
	return info
}

func extractGoFuncType(ft *goast.FuncType) *goFuncDef {
	fd := &goFuncDef{}
	if ft.Params != nil {
		for _, p := range ft.Params.List {
			typStr := typeString(p.Type)
			if len(p.Names) == 0 {
				fd.Params = append(fd.Params, goParamDef{Type: typStr})
			} else {
				for _, name := range p.Names {
					fd.Params = append(fd.Params, goParamDef{Name: name.Name, Type: typStr})
				}
			}
		}
	}
	if ft.Results != nil {
		for _, r := range ft.Results.List {
			typStr := typeString(r.Type)
			count := len(r.Names)
			if count == 0 {
				count = 1
			}
			for i := 0; i < count; i++ {
				fd.Returns = append(fd.Returns, typStr)
			}
		}
	}
	return fd
}

func receiverTypeName(expr goast.Expr) string {
	switch t := expr.(type) {
	case *goast.Ident:
		return t.Name
	case *goast.StarExpr:
		return receiverTypeName(t.X)
	case *goast.IndexExpr:
		return receiverTypeName(t.X)
	case *goast.IndexListExpr:
		return receiverTypeName(t.X)
	default:
		return ""
	}
}

// --- Auto-update logic ---

// addMissingSymbols compares Go info against the .grok block and adds
// missing exported structs, interfaces, functions, fields, and methods.
func addMissingSymbols(block *grokast.GrokBlock, info *goInfo) {
	// Build sets of what's already declared in .grok
	grokStructs := make(map[string]int)    // name → index
	grokClasses := make(map[string]int)    // name → index
	grokInterfaces := make(map[string]int) // name → index
	grokFuncs := make(map[string]bool)
	grokTypes := make(map[string]bool)
	grokEnums := make(map[string]bool)

	for i, s := range block.Structs {
		grokStructs[s.Name] = i
	}
	for i, c := range block.Classes {
		grokClasses[c.Name] = i
	}
	for i, iface := range block.Interfaces {
		grokInterfaces[iface.Name] = i
	}
	for _, e := range block.Enums {
		grokEnums[e.Name] = true
	}
	for _, f := range block.Functions {
		grokFuncs[f.Name] = true
	}
	for _, t := range block.TypeAliases {
		grokTypes[t.Name] = true
	}

	// Helper to check if a name is already declared in any form
	isKnown := func(name string) bool {
		if _, ok := grokStructs[name]; ok {
			return true
		}
		if _, ok := grokClasses[name]; ok {
			return true
		}
		if _, ok := grokInterfaces[name]; ok {
			return true
		}
		if grokEnums[name] || grokFuncs[name] || grokTypes[name] {
			return true
		}
		return false
	}

	// Track what line to assign new declarations (after last existing)
	nextLine := block.Span.End.Line

	// Add missing structs and update existing ones with missing fields/methods
	for name, sd := range info.Structs {
		if idx, ok := grokStructs[name]; ok {
			// Struct exists — add missing fields
			addMissingFields(&block.Structs[idx].Fields, sd.Fields)
		} else if idx, ok := grokClasses[name]; ok {
			// It's a class in .grok — add missing fields and methods
			addMissingFields(&block.Classes[idx].Fields, sd.Fields)
			addMissingMethods(&block.Classes[idx].Methods, sd.Methods, nextLine)
		} else if !isKnown(name) {
			// New struct — add it
			newStruct := makeGrokStruct(name, sd, nextLine)
			block.Structs = append(block.Structs, newStruct)
			nextLine++
		}
	}

	// Add missing interfaces and update existing ones with missing methods
	for name, id := range info.Interfaces {
		if idx, ok := grokInterfaces[name]; ok {
			addMissingIfaceMethods(&block.Interfaces[idx].Methods, id.Methods, nextLine)
		} else if !isKnown(name) {
			newIface := makeGrokInterface(name, id, nextLine)
			block.Interfaces = append(block.Interfaces, newIface)
			nextLine++
		}
	}

	// Add missing standalone functions
	for name, fd := range info.Functions {
		if isKnown(name) {
			continue
		}
		newFunc := makeGrokFunc(name, fd, nextLine)
		block.Functions = append(block.Functions, newFunc)
		nextLine++
	}

	// Add missing type aliases/defs
	for name, goType := range info.TypeDefs {
		if isKnown(name) {
			continue
		}
		newAlias := grokast.TypeAliasDecl{
			Name:     name,
			IsPublic: true,
			Type:     makeGrokTypeExpr(goTypeToGrok(goType)),
			Span:     grokast.Span{Start: grokast.Pos{Line: nextLine}},
		}
		block.TypeAliases = append(block.TypeAliases, newAlias)
		nextLine++
	}
}

// pruneStaleSymbols removes declarations from the .grok block that no longer
// exist in Go source. Only removes exported symbols — unexported ones are
// already excluded from auto-add.
func pruneStaleSymbols(block *grokast.GrokBlock, info *goInfo) {
	// Prune structs not in Go (unless they're in Go as interfaces or typedefs)
	var keepStructs []grokast.StructDecl
	for _, s := range block.Structs {
		if _, ok := info.Structs[s.Name]; ok {
			// Prune fields not in Go
			if isExported(s.Name) {
				pruneFields(&s.Fields, info.Structs[s.Name].Fields)
			}
			keepStructs = append(keepStructs, s)
		} else if !isExported(s.Name) {
			keepStructs = append(keepStructs, s) // keep unexported
		} else if _, ok := info.Interfaces[s.Name]; ok {
			keepStructs = append(keepStructs, s) // it's an interface in Go, keep
		} else if _, ok := info.TypeDefs[s.Name]; ok {
			keepStructs = append(keepStructs, s) // it's a typedef, keep
		}
		// else: exported, not in Go anywhere → drop
	}
	block.Structs = keepStructs

	// Prune interfaces
	var keepIfaces []grokast.InterfaceDecl
	for _, iface := range block.Interfaces {
		if goIface, ok := info.Interfaces[iface.Name]; ok {
			pruneIfaceMethods(&iface.Methods, goIface.Methods)
			keepIfaces = append(keepIfaces, iface)
		} else if !isExported(iface.Name) {
			keepIfaces = append(keepIfaces, iface)
		}
	}
	block.Interfaces = keepIfaces

	// Prune classes (Go structs declared as classes in .grok)
	var keepClasses []grokast.ClassDecl
	for _, c := range block.Classes {
		if goStruct, ok := info.Structs[c.Name]; ok {
			pruneFields(&c.Fields, goStruct.Fields)
			pruneMethods(&c.Methods, goStruct.Methods)
			keepClasses = append(keepClasses, c)
		} else if !isExported(c.Name) {
			keepClasses = append(keepClasses, c)
		}
	}
	block.Classes = keepClasses

	// Prune standalone functions — keep only if still a standalone Go function
	var keepFuncs []grokast.FuncDecl
	for _, f := range block.Functions {
		if _, ok := info.Functions[f.Name]; ok {
			keepFuncs = append(keepFuncs, f)
		} else {
			// Try case-insensitive match (Grok naming may differ)
			found := false
			for goName := range info.Functions {
				if strings.EqualFold(goName, f.Name) {
					found = true
					break
				}
			}
			if found {
				keepFuncs = append(keepFuncs, f)
			}
		}
	}
	block.Functions = keepFuncs

	// Prune type aliases
	var keepAliases []grokast.TypeAliasDecl
	for _, t := range block.TypeAliases {
		if _, ok := info.TypeDefs[t.Name]; ok {
			keepAliases = append(keepAliases, t)
		} else if !isExported(t.Name) {
			keepAliases = append(keepAliases, t)
		} else if _, ok := info.Structs[t.Name]; ok {
			keepAliases = append(keepAliases, t) // became a struct
		} else if _, ok := info.Interfaces[t.Name]; ok {
			keepAliases = append(keepAliases, t) // became an interface
		}
	}
	block.TypeAliases = keepAliases

	// Prune enums — keep if Go has a type with that name (typedef, struct, or interface)
	var keepEnums []grokast.EnumDecl
	for _, e := range block.Enums {
		if !isExported(e.Name) {
			keepEnums = append(keepEnums, e)
		} else if _, ok := info.TypeDefs[e.Name]; ok {
			keepEnums = append(keepEnums, e)
		} else if _, ok := info.Structs[e.Name]; ok {
			keepEnums = append(keepEnums, e)
		} else if _, ok := info.Interfaces[e.Name]; ok {
			keepEnums = append(keepEnums, e)
		}
	}
	block.Enums = keepEnums
}

func pruneFields(fields *[]grokast.Field, goFields map[string]string) {
	var keep []grokast.Field
	for _, f := range *fields {
		if _, ok := goFields[f.Name]; ok {
			keep = append(keep, f)
		}
	}
	*fields = keep
}

func pruneMethods(methods *[]grokast.FuncDecl, goMethods map[string]*goFuncDef) {
	var keep []grokast.FuncDecl
	for _, m := range *methods {
		if _, ok := goMethods[m.Name]; ok {
			keep = append(keep, m)
		} else if !isExported(m.Name) {
			keep = append(keep, m)
		}
	}
	*methods = keep
}

func pruneIfaceMethods(methods *[]grokast.FuncDecl, goMethods map[string]*goFuncDef) {
	var keep []grokast.FuncDecl
	for _, m := range *methods {
		if _, ok := goMethods[m.Name]; ok {
			keep = append(keep, m)
		}
	}
	*methods = keep
}

func addMissingFields(fields *[]grokast.Field, goFields map[string]string) {
	existing := make(map[string]bool)
	for _, f := range *fields {
		existing[f.Name] = true
	}
	// Sort for deterministic output
	var names []string
	for name := range goFields {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if existing[name] {
			continue
		}
		goType := goFields[name]
		*fields = append(*fields, grokast.Field{
			Name: name,
			Type: makeGrokTypeExpr(goTypeToGrok(goType)),
		})
	}
}

func addMissingMethods(methods *[]grokast.FuncDecl, goMethods map[string]*goFuncDef, line int) {
	existing := make(map[string]bool)
	for _, m := range *methods {
		existing[m.Name] = true
	}
	var names []string
	for name := range goMethods {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if existing[name] {
			continue
		}
		fd := goMethods[name]
		fn := makeGrokFunc(name, fd, line)
		// Add self param
		fn.Params = append([]grokast.Param{{IsSelf: true}}, fn.Params...)
		*methods = append(*methods, fn)
		line++
	}
}

func addMissingIfaceMethods(methods *[]grokast.FuncDecl, goMethods map[string]*goFuncDef, line int) {
	existing := make(map[string]bool)
	for _, m := range *methods {
		existing[m.Name] = true
	}
	var names []string
	for name := range goMethods {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if existing[name] {
			continue
		}
		fn := makeGrokFunc(name, goMethods[name], line)
		*methods = append(*methods, fn)
		line++
	}
}

// --- Builders ---

func makeGrokStruct(name string, sd *goStructDef, line int) grokast.StructDecl {
	s := grokast.StructDecl{
		Name:     name,
		IsPublic: true,
		Span:     grokast.Span{Start: grokast.Pos{Line: line}},
	}
	var fieldNames []string
	for n := range sd.Fields {
		fieldNames = append(fieldNames, n)
	}
	sort.Strings(fieldNames)
	for _, n := range fieldNames {
		s.Fields = append(s.Fields, grokast.Field{
			Name: n,
			Type: makeGrokTypeExpr(goTypeToGrok(sd.Fields[n])),
		})
	}
	return s
}

func makeGrokInterface(name string, id *goIfaceDef, line int) grokast.InterfaceDecl {
	iface := grokast.InterfaceDecl{
		Name:     name,
		IsPublic: true,
		Span:     grokast.Span{Start: grokast.Pos{Line: line}},
	}
	var methodNames []string
	for n := range id.Methods {
		methodNames = append(methodNames, n)
	}
	sort.Strings(methodNames)
	for _, n := range methodNames {
		fn := makeGrokFunc(n, id.Methods[n], line)
		iface.Methods = append(iface.Methods, fn)
	}
	return iface
}

func makeGrokFunc(name string, fd *goFuncDef, line int) grokast.FuncDecl {
	fn := grokast.FuncDecl{
		Name:     name,
		IsPublic: true,
		Span:     grokast.Span{Start: grokast.Pos{Line: line}},
	}
	for _, p := range fd.Params {
		param := grokast.Param{
			Name: p.Name,
			Type: makeGrokTypeExpr(goTypeToGrok(p.Type)),
		}
		if param.Name == "" {
			param.Name = "_"
		}
		fn.Params = append(fn.Params, param)
	}
	if len(fd.Returns) == 1 {
		ret := makeGrokTypeExpr(goTypeToGrok(fd.Returns[0]))
		fn.ReturnType = &ret
	} else if len(fd.Returns) > 1 {
		var fields []grokast.TupleField
		for _, r := range fd.Returns {
			fields = append(fields, grokast.TupleField{
				Type: makeGrokTypeExpr(goTypeToGrok(r)),
			})
		}
		ret := grokast.TypeExpr{
			Kind: grokast.TypeTuple,
			Data: grokast.TupleType{Fields: fields},
		}
		fn.ReturnType = &ret
	}
	return fn
}

// makeGrokTypeExpr creates a simple named TypeExpr from a Grok type string.
// For complex types (optionals, lists, maps), it does basic parsing.
func makeGrokTypeExpr(grokType string) grokast.TypeExpr {
	// Handle optional T?
	if strings.HasSuffix(grokType, "?") {
		inner := makeGrokTypeExpr(grokType[:len(grokType)-1])
		return grokast.TypeExpr{
			Kind: grokast.TypeOptional,
			Data: grokast.OptionalType{Inner: inner},
		}
	}
	// Handle list [T]
	if strings.HasPrefix(grokType, "[") && strings.HasSuffix(grokType, "]") {
		elem := makeGrokTypeExpr(grokType[1 : len(grokType)-1])
		return grokast.TypeExpr{
			Kind: grokast.TypeSequence,
			Data: grokast.SequenceType{Elem: elem},
		}
	}
	// Handle map[K]V
	if strings.HasPrefix(grokType, "map[") {
		depth := 0
		for i, c := range grokType {
			if c == '[' {
				depth++
			} else if c == ']' {
				depth--
				if depth == 0 {
					key := makeGrokTypeExpr(grokType[4:i])
					val := makeGrokTypeExpr(grokType[i+1:])
					return grokast.TypeExpr{
						Kind: grokast.TypeMap,
						Data: grokast.MapType{Key: key, Value: val},
					}
				}
			}
		}
	}
	// Simple named type
	return grokast.TypeExpr{
		Kind: grokast.TypeNamed,
		Data: grokast.NamedType{Name: grokType},
	}
}

// --- Helpers ---

func splitAtMarker(text, marker string) (string, string) {
	idx := strings.Index(text, marker)
	if idx < 0 {
		return text, ""
	}
	return text[:idx], text[idx:]
}

func extractFuncsFromGoFile(fullPath, baseName string) ([]funcEntry, []string, error) {
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
		if fn.Doc != nil && len(fn.Doc.List) > 0 {
			entry.DocLine = strings.TrimPrefix(fn.Doc.List[0].Text, "// ")
			entry.DocLine = strings.TrimPrefix(entry.DocLine, "/* ")
		}
		entry.Signature = buildSignature(fn, fset)
		funcs = append(funcs, entry)
	}

	sort.Slice(funcs, func(i, j int) bool { return funcs[i].Line < funcs[j].Line })

	var imports []string
	for _, imp := range f.Imports {
		imports = append(imports, strings.Trim(imp.Path.Value, `"`))
	}
	return funcs, imports, nil
}
