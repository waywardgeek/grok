### USER (gemini)

# Task: Expert Module Analysis

Create or improve `pkg/verifier/design.md` — the expert-level design documentation for the **verifier** module.

## Module Context

- Name: verifier
- Path: pkg/verifier
- Output: `pkg/verifier/design.md`
- Files: 2 source files

## Files to Analyze

- pkg/verifier/verifier.go
- pkg/verifier/verifier_test.go

## Your Strategy

1. **Read Existing Design**: Read `pkg/verifier/design.md` if it exists. This is your starting point — it may contain valuable architectural insights from prior runs that you MUST preserve. Also read `project-design.md` at the project root for system-wide context.
2. **Analyze Code**: Read ALL source files in this module. Identify internal dependencies and read their `design.md` files to understand the interfaces you consume.
3. **Cross-Link**: In your documentation, include **clickable markdown links** to the `design.md` files of any modules this package depends on (e.g., `[pkg/database](../database/design.md)`).
4. **Map Interfaces**: Explicitly identify which interfaces this module implements (e.g., check pkg/database/interfaces.go).
5. **Verify and Update**: Check that all information in the existing doc is still accurate against the current source files. Fix any bit-rot — but **do not drop design insights that are still valid**. Enhance, don't replace.
6. **Document Deeply**: Create a technical narrative that passes the Reimplementation Test. Ensure EVERY non-test source file is accounted for in your File Inventory, and **link to each file** (e.g., `[main.go](main.go)`).
7. **Write Output**: Use `write_file` to create `pkg/verifier/design.md`. Use `edit_file` to append additional sections if needed to stay under output token limits.

When finished, confirm: 'Documentation complete.'


---

### ASSISTANT (gemini)

**Tool Call: `list_directory`**
```json
{
  "directory": "pkg/verifier"
}
```

---

### USER (gemini)

**Tool Result:**
```
📁 **Directory Listing: /Users/bill/projects/grok/pkg/verifier**

📄 design.md (6.2 KB)
📄 verifier.go (23.6 KB)
📄 verifier.grok (1.4 KB)
📄 verifier_test.go (3.4 KB)

**Summary:** 4 items (4 files, 0 directories)
**Total Size:** 0.03 MB
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/verifier/design.md"
}
```

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/verifier/verifier.go"
}
```

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/verifier/verifier_test.go"
}
```

**Tool Call: `read_file`**
```json
{
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_852cc24b49ee758b]
File: /Users/bill/projects/grok/pkg/verifier/design.md (72 lines)

# Verifier Module Design

## Executive Summary

The `verifier` module is the structural integrity engine of the Grok system. Its primary purpose is to detect "architectural drift"—the divergence between the high-level design declared in `.grok` files and the actual implementation in Go source code. By comparing the Abstract Syntax Tree (AST) of a Grok specification against the AST of the corresponding Go code, the verifier ensures that the developer's mental model (as captured in Grok) remains synchronized with the reality of the codebase. It reports mismatches in types, field names, function signatures, and interface implementations as errors, warnings, or informational findings.

## File Inventory

- [verifier.go](verifier.go): The core implementation of the verification logic. It handles Go source parsing, type extraction, and the recursive comparison of Grok declarations against Go implementations.
- [verifier_test.go](verifier_test.go): Comprehensive tests for the verifier, including integration tests that run the verifier against the project's own `.grok` files and unit tests for naming convention conversions.

## Architecture and Data Flow

The verifier operates in a multi-stage pipeline that bridges the gap between the Grok and Go type systems.

1.  **Grok Parsing**: The process begins by using the [pkg/parser](../parser/design.md) to transform a `.grok` file into a [pkg/ast](../ast/design.md) representation.
2.  **Go Source Discovery**: For each `grok` block in the file, the verifier identifies the associated Go source files or directories specified in the `source:` annotations.
3.  **Go Type Extraction**: The verifier uses the standard library's `go/parser` and `go/ast` packages to parse the referenced Go code. It then traverses the Go AST to build an internal `goTypeInfo` map, which aggregates all structs, interfaces, functions, and type definitions found across the specified sources.
4.  **Structural Comparison**: The verifier performs a deep, recursive comparison between the Grok AST nodes and the extracted Go type information. It maps Grok's snake_case identifiers to Go's PascalCase or camelCase conventions and converts Grok's type expressions into a normalized string format for comparison.
5.  **Finding Generation**: Any discrepancies found during comparison are recorded as `Finding` objects, categorized by `Severity` (Error, Warning, or Info).

## Interface Implementations

The `verifier` module does not currently implement any external interfaces. It provides a functional API for performing verification and returning a structured `Result`.

## Public API

The public API is designed for simplicity and integration into build pipelines or CI/CD workflows.

- **Verify(grokPath string) (*Result, error)**: The primary entry point. It parses the specified `.grok` file, locates the corresponding Go source code, performs the verification, and returns a `Result` containing all findings.
- **Result**: A container for the findings of a verification run. It provides an `ErrorCount()` method to quickly determine if any critical drift was detected.
- **Finding**: Represents a single instance of drift. It includes the `Severity`, the paths to the involved files, and a descriptive message.
- **Severity**: An enumeration (`Error`, `Warning`, `Info`) that classifies the impact of a finding.

## Implementation Details

### Go Type Extraction

The `extractGoTypes` function is the heart of the Go analysis phase. It processes `*ast.File` nodes and populates a `goTypeInfo` structure. It specifically handles:
-   **Structs**: Capturing field names and types.
-   **Interfaces**: Capturing method signatures.
-   **Functions**: Capturing top-level functions and their signatures.
-   **Methods**: Identifying receiver types to associate methods with their respective structs.
-   **Type Definitions**: Tracking simple type aliases.

### Type Mapping and Comparison

Grok and Go have different type systems and naming conventions. The verifier bridges these using several strategies:
-   **Naming Conventions**: The `snakeToPascal` and `snakeToCamel` helpers allow the verifier to match a Grok field like `user_id` to a Go field `UserID` or `userId`.
-   **Type Normalization**: The `grokTypeToGoString` function converts complex Grok `TypeExpr` nodes (like `[string]?`) into their Go equivalents (like `*[]string`).
-   **Fuzzy Matching**: The `typesMatch` function handles Go-specific nuances, such as the equivalence of `any` and `interface{}` and the presence of package prefixes (e.g., matching `ast.File` in Go to `File` in Grok).

### Verification Logic

The verification is partitioned by declaration type:
-   **Structs/Classes**: Checks for field existence and type compatibility. It also reports "extra" fields in Go that are missing from Grok as warnings, encouraging complete documentation.
-   **Interfaces**: Ensures that all methods declared in the Grok interface exist in the Go interface with matching signatures.
-   **Functions**: Compares parameter counts, parameter types, and return types. It handles Grok's "self" parameter (used in class methods) by excluding it from the positional comparison with Go parameters.

## Dependencies

-   [pkg/ast](../ast/design.md): Provides the AST structures for the Grok language.
-   [pkg/parser](../parser/design.md): Used to parse `.grok` files.
-   **Go Standard Library**: Relies heavily on `go/ast`, `go/parser`, and `go/token` for analyzing Go source code.

## Technical Debt and Future Work

-   **Expression Verification**: Currently, the verifier only checks structural types. Future versions should verify the `requires` and `ensures` contracts by parsing the expressions and potentially using static analysis or symbolic execution.
-   **Pointer Nuances**: The current type comparison is somewhat simplified regarding pointer indirection. More robust handling of pointer vs. value receivers and types would reduce false positives/negatives.
-   **Import Resolution**: The verifier currently assumes that unqualified types in Grok match types in the Go package being analyzed. Support for cross-package type verification would be a significant improvement.
-   **Performance**: For very large projects, parsing all Go files for every `grok` block might become slow. Implementing a cache for parsed Go type information would improve performance.
```

**Tool Result:**
```
[ID: tool_c640243cb72efdf1]
File: /Users/bill/projects/grok/pkg/verifier/verifier.go (892 lines)

// Package verifier compares .grok understanding files against Go source code,
// reporting structural drift between the declared understanding and the implementation.
package verifier

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	grokast "github.com/waywardgeek/grok/pkg/ast"
	"github.com/waywardgeek/grok/pkg/parser"
)

// Severity classifies how serious a drift finding is.
type Severity int

const (
	Error   Severity = iota // missing type, wrong field type
	Warning                 // extra fields not in .grok, naming convention mismatch
	Info                    // informational (e.g., Go has more methods than .grok declares)
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "ERROR"
	case Warning:
		return "WARNING"
	case Info:
		return "INFO"
	}
	return "UNKNOWN"
}

// Finding is a single drift report.
type Finding struct {
	Severity Severity
	GrokFile string
	GoFile   string
	Message  string
}

func (f Finding) String() string {
	loc := f.GrokFile
	if f.GoFile != "" {
		loc = fmt.Sprintf("%s ↔ %s", f.GrokFile, f.GoFile)
	}
	return fmt.Sprintf("[%s] %s: %s", f.Severity, loc, f.Message)
}

// Result holds all findings from a verification run.
type Result struct {
	Findings []Finding
}

func (r *Result) add(sev Severity, grokFile, goFile, msg string) {
	r.Findings = append(r.Findings, Finding{
		Severity: sev,
		GrokFile: grokFile,
		GoFile:   goFile,
		Message:  msg,
	})
}

// ErrorCount returns the number of error-level findings.
func (r *Result) ErrorCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == Error {
			n++
		}
	}
	return n
}

// Verify parses a .grok file and compares it against the Go source files
// referenced in source: annotations. baseDir is the project root used to
// resolve relative source paths.
func Verify(grokPath string) (*Result, error) {
	src, err := os.ReadFile(grokPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", grokPath, err)
	}

	grokFile, err := parser.ParseFile(string(src), grokPath)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", grokPath, err)
	}

	result := &Result{}

	grokDir := filepath.Dir(grokPath)

	for _, block := range grokFile.Blocks {
		if len(block.Source) == 0 {
			result.add(Info, grokPath, "", fmt.Sprintf("grok block %q has no source: annotations", block.Name))
			continue
		}

		// Aggregate all Go types across all source files
		goInfo := &goTypeInfo{
			Structs:    make(map[string]*goStructInfo),
			Interfaces: make(map[string]*goInterfaceInfo),
			Functions:  make(map[string]*goFuncInfo),
			TypeDefs:   make(map[string]bool),
		}

		for _, srcPath := range block.Source {
			// Resolve source paths relative to the .grok file's directory
			goFullPath := filepath.Join(grokDir, srcPath)
			info, err := os.Stat(goFullPath)
			if err != nil {
				if os.IsNotExist(err) {
					result.add(Error, grokPath, srcPath, "source file does not exist")
				} else {
					result.add(Error, grokPath, srcPath, fmt.Sprintf("cannot stat: %v", err))
				}
				continue
			}

			var fileInfo *goTypeInfo
			if info.IsDir() {
				fileInfo, err = parseGoDir(goFullPath)
			} else {
				fileInfo, err = parseGoFile(goFullPath)
			}
			if err != nil {
				result.add(Error, grokPath, srcPath, fmt.Sprintf("failed to parse Go file: %v", err))
				continue
			}
			mergeGoInfo(goInfo, fileInfo)
		}

		// Now compare the grok block against the aggregated Go types
		verifyBlock(block, goInfo, grokPath, result)
	}

	return result, nil
}

// goTypeInfo holds extracted Go type information.
type goTypeInfo struct {
	Structs    map[string]*goStructInfo
	Interfaces map[string]*goInterfaceInfo
	Functions  map[string]*goFuncInfo
	TypeDefs   map[string]bool // type names that are simple typedefs (e.g., type Foo int)
}

type goStructInfo struct {
	Fields  map[string]string // field name → type string
	Methods map[string]*goFuncInfo
}

type goInterfaceInfo struct {
	Methods map[string]*goFuncInfo
}

type goFuncInfo struct {
	Params  []goParam
	Returns []string
}

type goParam struct {
	Name string
	Type string
}

func mergeGoInfo(dst, src *goTypeInfo) {
	for k, v := range src.Structs {
		if existing, ok := dst.Structs[k]; ok {
			// Merge methods
			for mk, mv := range v.Methods {
				existing.Methods[mk] = mv
			}
			for fk, fv := range v.Fields {
				existing.Fields[fk] = fv
			}
		} else {
			dst.Structs[k] = v
		}
	}
	for k, v := range src.Interfaces {
		dst.Interfaces[k] = v
	}
	for k, v := range src.Functions {
		dst.Functions[k] = v
	}
	for k, v := range src.TypeDefs {
		dst.TypeDefs[k] = v
	}
}

func verifyBlock(block grokast.GrokBlock, goInfo *goTypeInfo, grokPath string, result *Result) {
	srcStr := strings.Join(block.Source, ", ")

	for _, s := range block.Structs {
		verifyStruct(s, goInfo, grokPath, srcStr, result)
	}
	for _, c := range block.Classes {
		verifyClass(c, goInfo, grokPath, srcStr, result)
	}
	for _, e := range block.Enums {
		verifyEnum(e, goInfo, grokPath, srcStr, result)
	}
	for _, i := range block.Interfaces {
		verifyInterface(i, goInfo, grokPath, srcStr, result)
	}
	for _, f := range block.Functions {
		verifyFunction(f, goInfo, grokPath, srcStr, result)
	}
}

func parseGoDir(dir string) (*goTypeInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := goparser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return nil, err
	}

	info := &goTypeInfo{
		Structs:    make(map[string]*goStructInfo),
		Interfaces: make(map[string]*goInterfaceInfo),
		Functions:  make(map[string]*goFuncInfo),
		TypeDefs:   make(map[string]bool),
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			extractGoTypes(file, info)
		}
	}
	return info, nil
}

func parseGoFile(path string) (*goTypeInfo, error) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	info := &goTypeInfo{
		Structs:    make(map[string]*goStructInfo),
		Interfaces: make(map[string]*goInterfaceInfo),
		Functions:  make(map[string]*goFuncInfo),
		TypeDefs:   make(map[string]bool),
	}

	extractGoTypes(file, info)
	return info, nil
}

func extractGoTypes(file *ast.File, info *goTypeInfo) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					switch t := ts.Type.(type) {
					case *ast.StructType:
						si := &goStructInfo{
							Fields:  make(map[string]string),
							Methods: make(map[string]*goFuncInfo),
						}
						if t.Fields != nil {
							for _, f := range t.Fields.List {
								typStr := typeExprString(f.Type)
								for _, name := range f.Names {
									si.Fields[name.Name] = typStr
								}
							}
						}
						info.Structs[ts.Name.Name] = si

					case *ast.InterfaceType:
						ii := &goInterfaceInfo{
							Methods: make(map[string]*goFuncInfo),
						}
						if t.Methods != nil {
							for _, m := range t.Methods.List {
								if ft, ok := m.Type.(*ast.FuncType); ok {
									for _, name := range m.Names {
										ii.Methods[name.Name] = extractFuncType(ft)
									}
								}
							}
						}
						info.Interfaces[ts.Name.Name] = ii

					default:
						// Simple typedef (type Foo int, type Bar string, etc.)
						info.TypeDefs[ts.Name.Name] = true
					}
				}
			}

		case *ast.FuncDecl:
			fi := extractFuncInfo(d)
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recvType := receiverTypeName(d.Recv.List[0].Type)
				if recvType != "" {
					si, ok := info.Structs[recvType]
					if !ok {
						si = &goStructInfo{
							Fields:  make(map[string]string),
							Methods: make(map[string]*goFuncInfo),
						}
						info.Structs[recvType] = si
					}
					si.Methods[d.Name.Name] = fi
				}
			} else {
				info.Functions[d.Name.Name] = fi
			}
		}
	}
}

func extractFuncInfo(d *ast.FuncDecl) *goFuncInfo {
	return extractFuncType(d.Type)
}

func extractFuncType(ft *ast.FuncType) *goFuncInfo {
	fi := &goFuncInfo{}
	if ft.Params != nil {
		for _, p := range ft.Params.List {
			typStr := typeExprString(p.Type)
			if len(p.Names) == 0 {
				fi.Params = append(fi.Params, goParam{Type: typStr})
			} else {
				for _, name := range p.Names {
					fi.Params = append(fi.Params, goParam{Name: name.Name, Type: typStr})
				}
			}
		}
	}
	if ft.Results != nil {
		for _, r := range ft.Results.List {
			typStr := typeExprString(r.Type)
			count := len(r.Names)
			if count == 0 {
				count = 1
			}
			for i := 0; i < count; i++ {
				fi.Returns = append(fi.Returns, typStr)
			}
		}
	}
	return fi
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.IndexListExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeExprString(t.Elt)
		}
		return "[...]" + typeExprString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprString(t.Key) + "]" + typeExprString(t.Value)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + typeExprString(t.Value)
	case *ast.Ellipsis:
		return "..." + typeExprString(t.Elt)
	case *ast.IndexExpr:
		return typeExprString(t.X) + "[" + typeExprString(t.Index) + "]"
	case *ast.IndexListExpr:
		var parts []string
		for _, idx := range t.Indices {
			parts = append(parts, typeExprString(idx))
		}
		return typeExprString(t.X) + "[" + strings.Join(parts, ", ") + "]"
	default:
		return "?"
	}
}

// ---- Grok type → Go type string conversion ----

// grokTypeToGoString converts a grok TypeExpr into the string format that
// typeExprString produces from Go AST, enabling structural comparison.
func grokTypeToGoString(t grokast.TypeExpr) string {
	switch t.Kind {
	case grokast.TypeNamed:
		if t.Data == nil {
			return "?"
		}
		nt := t.Data.(grokast.NamedType)
		name := grokNameToGo(nt.Name)
		if len(nt.Args) == 0 {
			return name
		}
		// Generic: Stack<T> → Stack[T], Map<K,V> → Map[K, V]
		var args []string
		for _, a := range nt.Args {
			args = append(args, grokTypeToGoString(a))
		}
		return name + "[" + strings.Join(args, ", ") + "]"

	case grokast.TypeOptional:
		if t.Data == nil {
			return "?"
		}
		ot := t.Data.(grokast.OptionalType)
		// T? in grok → *T in Go
		return "*" + grokTypeToGoString(ot.Inner)

	case grokast.TypeSequence:
		if t.Data == nil {
			return "[]?"
		}
		st := t.Data.(grokast.SequenceType)
		return "[]" + grokTypeToGoString(st.Elem)

	case grokast.TypeMap:
		if t.Data == nil {
			return "map[?]?"
		}
		mt := t.Data.(grokast.MapType)
		return "map[" + grokTypeToGoString(mt.Key) + "]" + grokTypeToGoString(mt.Value)

	case grokast.TypeTuple:
		// Go doesn't have tuples; skip comparison
		return "?"

	case grokast.TypeFunc:
		return "func(...)"

	case grokast.TypeChannel:
		if t.Data == nil {
			return "chan ?"
		}
		ct := t.Data.(grokast.ChannelType)
		return "chan " + grokTypeToGoString(ct.Elem)

	case grokast.TypeLock:
		return "sync.Mutex"

	case grokast.TypeUnit:
		return ""

	default:
		return "?"
	}
}

// grokNameToGo maps grok primitive type names to Go equivalents.
func grokNameToGo(name string) string {
	switch name {
	case "string":
		return "string"
	case "bool":
		return "bool"
	case "i8":
		return "int8"
	case "i16":
		return "int16"
	case "i32":
		return "int32"
	case "i64":
		return "int64"
	case "i128", "i256":
		return name // no Go equivalent, keep as-is
	case "u8":
		return "uint8"
	case "u16":
		return "uint16"
	case "u32":
		return "uint32"
	case "u64":
		return "uint64"
	case "f32":
		return "float32"
	case "f64":
		return "float64"
	case "f128":
		return name
	case "int":
		return "int"
	case "any":
		return "any"
	case "error":
		return "error"
	default:
		return name // user-defined types pass through
	}
}

// typesMatch compares a grok type string against a Go type string.
// Returns true if they match or if the grok type is "?" (unknown/unconvertible).
func typesMatch(grokStr, goStr string) bool {
	if grokStr == "?" || grokStr == "" {
		return true // can't compare, don't report false positive
	}
	if grokStr == goStr {
		return true
	}
	// Go 1.18+: "any" is an alias for "interface{}"
	if (grokStr == "any" && goStr == "interface{}") || (grokStr == "interface{}" && goStr == "any") {
		return true
	}
	// Also handle in composite types: map[string]any == map[string]interface{}, etc.
	if strings.ReplaceAll(grokStr, "any", "interface{}") == goStr || grokStr == strings.ReplaceAll(goStr, "interface{}", "any") {
		return true
	}
	// Strip Go package prefix: "ast.Span" → "Span", "*ast.File" → "*File"
	// since .grok files use unqualified type names
	stripped := stripPackagePrefix(goStr)
	if grokStr == stripped {
		return true
	}
	return false
}

// stripPackagePrefix removes Go package qualifiers from a type string.
// Handles nested cases like "*ast.File", "[]ast.Node", "map[string]*ast.File".
func stripPackagePrefix(goType string) string {
	// Handle pointer prefix
	if strings.HasPrefix(goType, "*") {
		return "*" + stripPackagePrefix(goType[1:])
	}
	// Handle slice prefix
	if strings.HasPrefix(goType, "[]") {
		return "[]" + stripPackagePrefix(goType[2:])
	}
	// Handle map
	if strings.HasPrefix(goType, "map[") {
		// Find the ] that closes the key
		depth := 1
		i := 4
		for i < len(goType) && depth > 0 {
			if goType[i] == '[' {
				depth++
			} else if goType[i] == ']' {
				depth--
			}
			i++
		}
		return "map[" + stripPackagePrefix(goType[4:i-1]) + "]" + stripPackagePrefix(goType[i:])
	}
	// Strip package prefix from qualified name: "ast.File" → "File"
	if idx := strings.LastIndex(goType, "."); idx >= 0 {
		return goType[idx+1:]
	}
	return goType
}

// ---- Verification helpers ----

// findGoName checks if a name exists in a map, trying both PascalCase and camelCase.
// Returns the found name and whether it was found.
func findGoName(name string, names map[string]*goFuncInfo) (string, bool) {
	pascal := snakeToPascal(name)
	if _, ok := names[pascal]; ok {
		return pascal, true
	}
	camel := snakeToCamel(name)
	if _, ok := names[camel]; ok {
		return camel, true
	}
	// Try exact match
	if _, ok := names[name]; ok {
		return name, true
	}
	return pascal, false
}

func verifyStruct(s grokast.StructDecl, goInfo *goTypeInfo, grokFile, goFile string, result *Result) {
	goStruct, ok := goInfo.Structs[s.Name]
	if !ok {
		result.add(Error, grokFile, goFile, fmt.Sprintf("struct %s declared in .grok but not found in Go", s.Name))
		return
	}

	for _, grokField := range s.Fields {
		goType, found := findGoFieldType(grokField.Name, goStruct.Fields)
		if !found {
			result.add(Error, grokFile, goFile, fmt.Sprintf("struct %s: field %s not found in Go", s.Name, grokField.Name))
			continue
		}
		grokType := grokTypeToGoString(grokField.Type)
		if !typesMatch(grokType, goType) {
			result.add(Error, grokFile, goFile, fmt.Sprintf("struct %s: field %s type mismatch: .grok=%s, Go=%s", s.Name, grokField.Name, grokType, goType))
		}
	}

	grokFieldSet := make(map[string]bool)
	for _, f := range s.Fields {
		grokFieldSet[snakeToPascal(f.Name)] = true
		grokFieldSet[snakeToCamel(f.Name)] = true
		grokFieldSet[f.Name] = true
	}
	var extras []string
	for goField := range goStruct.Fields {
		if !grokFieldSet[goField] {
			extras = append(extras, goField)
		}
	}
	sort.Strings(extras)
	for _, extra := range extras {
		result.add(Warning, grokFile, goFile, fmt.Sprintf("struct %s: Go has field %s not in .grok", s.Name, extra))
	}
}

func verifyClass(c grokast.ClassDecl, goInfo *goTypeInfo, grokFile, goFile string, result *Result) {
	goStruct, ok := goInfo.Structs[c.Name]
	if !ok {
		result.add(Error, grokFile, goFile, fmt.Sprintf("class %s declared in .grok but not found as Go struct", c.Name))
		return
	}

	for _, grokField := range c.Fields {
		goType, found := findGoFieldType(grokField.Name, goStruct.Fields)
		if !found {
			result.add(Error, grokFile, goFile, fmt.Sprintf("class %s: field %s not found in Go", c.Name, grokField.Name))
			continue
		}
		grokType := grokTypeToGoString(grokField.Type)
		if !typesMatch(grokType, goType) {
			result.add(Error, grokFile, goFile, fmt.Sprintf("class %s: field %s type mismatch: .grok=%s, Go=%s", c.Name, grokField.Name, grokType, goType))
		}
	}

	for _, param := range c.CtorParams {
		if param.IsSelf {
			continue
		}
		if !findGoField(param.Name, goStruct.Fields) {
			result.add(Warning, grokFile, goFile, fmt.Sprintf("class %s: ctor param %s not found as field", c.Name, param.Name))
		}
	}

	for _, grokMethod := range c.Methods {
		goName, found := findGoName(grokMethod.Name, goStruct.Methods)
		if !found {
			pascal := snakeToPascal(grokMethod.Name)
			camel := snakeToCamel(grokMethod.Name)
			result.add(Error, grokFile, goFile, fmt.Sprintf("class %s: method %s (tried Go: %s, %s) not found", c.Name, grokMethod.Name, pascal, camel))
			continue
		}
		goFunc := goStruct.Methods[goName]
		verifyFuncSignature(fmt.Sprintf("class %s method %s", c.Name, grokMethod.Name), grokMethod, goFunc, grokFile, goFile, result)
	}

	grokMethodSet := make(map[string]bool)
	for _, m := range c.Methods {
		grokMethodSet[snakeToPascal(m.Name)] = true
		grokMethodSet[snakeToCamel(m.Name)] = true
		grokMethodSet[m.Name] = true
	}
	var extras []string
	for goMethod := range goStruct.Methods {
		if !grokMethodSet[goMethod] {
			extras = append(extras, goMethod)
		}
	}
	sort.Strings(extras)
	for _, extra := range extras {
		result.add(Info, grokFile, goFile, fmt.Sprintf("class %s: Go has method %s not in .grok", c.Name, extra))
	}
}

func verifyEnum(e grokast.EnumDecl, goInfo *goTypeInfo, grokFile, goFile string, result *Result) {
	// Go enums are typically: type FooKind int (typedef) or type Foo int
	// Check for the type as a typedef, struct, or interface
	_, hasStruct := goInfo.Structs[e.Name]
	_, hasIface := goInfo.Interfaces[e.Name]
	_, hasTypedef := goInfo.TypeDefs[e.Name]

	if hasStruct || hasIface || hasTypedef {
		return
	}

	// Also check XxxKind pattern
	kindName := e.Name + "Kind"
	if goInfo.TypeDefs[kindName] {
		return
	}

	result.add(Warning, grokFile, goFile, fmt.Sprintf("enum %s: no matching Go type found (looked for %s, %s as typedef/struct/interface)", e.Name, e.Name, kindName))
}

func verifyInterface(i grokast.InterfaceDecl, goInfo *goTypeInfo, grokFile, goFile string, result *Result) {
	goIface, ok := goInfo.Interfaces[i.Name]
	if !ok {
		result.add(Error, grokFile, goFile, fmt.Sprintf("interface %s declared in .grok but not found in Go", i.Name))
		return
	}

	for _, grokMethod := range i.Methods {
		goName := toGoMethodName(grokMethod.Name)
		goFunc, ok := goIface.Methods[goName]
		if !ok {
			result.add(Error, grokFile, goFile, fmt.Sprintf("interface %s: method %s (Go: %s) not found", i.Name, grokMethod.Name, goName))
			continue
		}
		verifyFuncSignature(fmt.Sprintf("interface %s method %s", i.Name, grokMethod.Name), grokMethod, goFunc, grokFile, goFile, result)
	}
}

func verifyFunction(f grokast.FuncDecl, goInfo *goTypeInfo, grokFile, goFile string, result *Result) {
	pascal := snakeToPascal(f.Name)
	camel := snakeToCamel(f.Name)

	var goFunc *goFuncInfo
	if fn, ok := goInfo.Functions[pascal]; ok {
		goFunc = fn
	} else if fn, ok := goInfo.Functions[camel]; ok {
		goFunc = fn
	} else if fn, ok := goInfo.Functions[f.Name]; ok {
		goFunc = fn
	}

	if goFunc == nil {
		result.add(Error, grokFile, goFile, fmt.Sprintf("function %s (tried Go: %s, %s) not found", f.Name, pascal, camel))
		return
	}

	verifyFuncSignature(fmt.Sprintf("function %s", f.Name), f, goFunc, grokFile, goFile, result)
}

// ---- Naming convention helpers ----

// toGoFieldName tries PascalCase first (exported), then the original name (unexported).
func toGoFieldName(name string) string {
	return snakeToPascal(name)
}

// findGoField checks if a field exists, trying PascalCase, camelCase, and exact match.
func findGoField(name string, fields map[string]string) bool {
	_, found := findGoFieldType(name, fields)
	return found
}

// findGoFieldType returns the Go type string for a field, trying PascalCase, camelCase, and exact match.
func findGoFieldType(name string, fields map[string]string) (string, bool) {
	pascal := snakeToPascal(name)
	if typ, ok := fields[pascal]; ok {
		return typ, true
	}
	camel := snakeToCamel(name)
	if typ, ok := fields[camel]; ok {
		return typ, true
	}
	if typ, ok := fields[name]; ok {
		return typ, true
	}
	return "", false
}

// verifyFuncSignature checks parameter count and return type of a grok function against Go.
func verifyFuncSignature(context string, grokFunc grokast.FuncDecl, goFunc *goFuncInfo, grokFile, goFile string, result *Result) {
	// Count grok params excluding self
	grokParamCount := 0
	for _, p := range grokFunc.Params {
		if !p.IsSelf {
			grokParamCount++
		}
	}
	goParamCount := len(goFunc.Params)

	if grokParamCount != goParamCount {
		result.add(Error, grokFile, goFile, fmt.Sprintf("%s: param count mismatch: .grok=%d, Go=%d", context, grokParamCount, goParamCount))
	} else {
		// Check param types positionally
		gi := 0
		for _, grokParam := range grokFunc.Params {
			if grokParam.IsSelf {
				continue
			}
			if gi < len(goFunc.Params) {
				grokType := grokTypeToGoString(grokParam.Type)
				goType := goFunc.Params[gi].Type
				if !typesMatch(grokType, goType) {
					paramName := grokParam.Name
					if paramName == "" {
						paramName = fmt.Sprintf("#%d", gi+1)
					}
					result.add(Error, grokFile, goFile, fmt.Sprintf("%s: param %s type mismatch: .grok=%s, Go=%s", context, paramName, grokType, goType))
				}
			}
			gi++
		}
	}

	// Check return types
	grokReturnCount := 0
	var grokReturnStr string
	if grokFunc.ReturnType != nil {
		grokReturnStr = grokTypeToGoString(*grokFunc.ReturnType)
		if grokFunc.ReturnType.Kind == grokast.TypeTuple {
			if grokFunc.ReturnType.Data != nil {
				tt := grokFunc.ReturnType.Data.(grokast.TupleType)
				grokReturnCount = len(tt.Fields)
			}
		} else if grokFunc.ReturnType.Kind == grokast.TypeUnit {
			grokReturnCount = 0
		} else {
			grokReturnCount = 1
		}
	}
	goReturnCount := len(goFunc.Returns)

	if grokReturnCount != goReturnCount {
		result.add(Error, grokFile, goFile, fmt.Sprintf("%s: return count mismatch: .grok=%d, Go=%d", context, grokReturnCount, goReturnCount))
	} else if grokReturnCount == 1 && goReturnCount == 1 {
		if !typesMatch(grokReturnStr, goFunc.Returns[0]) {
			result.add(Error, grokFile, goFile, fmt.Sprintf("%s: return type mismatch: .grok=%s, Go=%s", context, grokReturnStr, goFunc.Returns[0]))
		}
	} else if grokReturnCount > 1 && grokFunc.ReturnType != nil && grokFunc.ReturnType.Kind == grokast.TypeTuple {
		// Element-wise comparison for tuple returns
		tt := grokFunc.ReturnType.Data.(grokast.TupleType)
		for i, field := range tt.Fields {
			if i < len(goFunc.Returns) {
				grokElemStr := grokTypeToGoString(field.Type)
				if !typesMatch(grokElemStr, goFunc.Returns[i]) {
					result.add(Error, grokFile, goFile, fmt.Sprintf("%s: return #%d type mismatch: .grok=%s, Go=%s", context, i+1, grokElemStr, goFunc.Returns[i]))
				}
			}
		}
	}
}

func toGoMethodName(name string) string {
	return snakeToPascal(name)
}

// snakeToPascal converts snake_case to PascalCase.
func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		result.WriteString(strings.ToUpper(p[:1]))
		result.WriteString(p[1:])
	}
	return result.String()
}

// snakeToCamel converts snake_case to camelCase.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			result.WriteString(p)
		} else {
			result.WriteString(strings.ToUpper(p[:1]))
			result.WriteString(p[1:])
		}
	}
	return result.String()
}
```

**Tool Result:**
```
[ID: tool_81d844dcbb49fecc]
File: /Users/bill/projects/grok/pkg/verifier/verifier_test.go (170 lines)

package verifier

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}

func TestVerifyParserGrok(t *testing.T) {
	dir := findProjectRoot(t)

	grokFile := filepath.Join(dir, "pkg", "parser", "parser.grok")
	result, err := Verify(grokFile)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	for _, f := range result.Findings {
		switch f.Severity {
		case Error:
			t.Errorf("%s", f)
		case Warning:
			t.Logf("%s", f)
		case Info:
			t.Logf("%s", f)
		}
	}
}

func TestVerifyAstGrok(t *testing.T) {
	dir := findProjectRoot(t)
	grokFile := filepath.Join(dir, "pkg", "ast", "ast.grok")
	result, err := Verify(grokFile)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	for _, f := range result.Findings {
		switch f.Severity {
		case Error:
			t.Errorf("%s", f)
		case Warning:
			t.Logf("%s", f)
		case Info:
			t.Logf("%s", f)
		}
	}
}

func TestTypeDriftDetected(t *testing.T) {
	// Create a temporary Go file and .grok file with deliberate type mismatches
	dir := t.TempDir()

	// Write a Go source file
	goSrc := `package example

type Widget struct {
	Name    string
	Count   int
	Tags    []string
	Options map[string]bool
}

func NewWidget(name string, count int) *Widget {
	return &Widget{Name: name, Count: count}
}
`
	goFile := filepath.Join(dir, "widget.go")
	if err := os.WriteFile(goFile, []byte(goSrc), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a .grok file with deliberate drift
	grokSrc := `grok Example {
  struct Widget {
    name:    string
    count:   string
    tags:    [int]
    options: map[string]string
    missing: bool
  }

  func new_widget(name: string) -> Widget

  source: ["widget.go"]
}
`
	grokFile := filepath.Join(dir, "example.grok")
	if err := os.WriteFile(grokFile, []byte(grokSrc), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Verify(grokFile)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	// Collect error messages
	var errors []string
	for _, f := range result.Findings {
		if f.Severity == Error {
			errors = append(errors, f.Message)
		}
	}

	// Expected errors:
	// 1. count type mismatch (string vs int)
	// 2. tags type mismatch ([]int vs []string)
	// 3. options type mismatch (map[string]string vs map[string]bool)
	// 4. missing field not in Go
	// 5. new_widget param count mismatch (1 vs 2)
	if len(errors) < 5 {
		t.Errorf("expected at least 5 errors, got %d:", len(errors))
		for _, e := range errors {
			t.Logf("  %s", e)
		}
	}

	// Verify specific drift was caught
	assertContains := func(substr string) {
		for _, e := range errors {
			if strings.Contains(e, substr) {
				return
			}
		}
		t.Errorf("expected error containing %q, not found in: %v", substr, errors)
	}

	assertContains("count")
	assertContains("type mismatch")
	assertContains("missing")
	assertContains("param count mismatch")
}

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"name", "Name"},
		{"type_params", "TypeParams"},
		{"is_many", "IsMany"},
		{"return_type", "ReturnType"},
		{"parse_grok_block", "ParseGrokBlock"},
	}
	for _, tt := range tests {
		got := snakeToPascal(tt.in)
		if got != tt.want {
			t.Errorf("snakeToPascal(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
```

**Tool Result:**
```
[ID: tool_efca99c28a24c65d]
File: /Users/bill/projects/grok/project-design.md (48 lines)

# Grok Project Architecture

## Executive Summary

The Grok project is a structural integrity engine and language toolchain designed to bridge the gap between high-level architectural design and concrete implementation. It introduces the Grok language, which allows developers to declare the structural and semantic contracts of their systems—such as types, relations, and concurrency constraints—in `.grok` files. The primary goal of the project is to detect "architectural drift" by comparing these declarative models against the actual Go source code implementation. By ensuring that the developer's mental model remains synchronized with the reality of the codebase, Grok acts as a continuous verification system for software architecture.

## System Architecture

The architecture of the Grok project follows a classic compiler frontend pipeline integrated with a static analysis engine. The system is composed of three primary modules: the Abstract Syntax Tree (`ast`), the `parser`, and the `verifier`. The philosophy of the architecture is centered around a single source of truth for the structural model, which is the AST. Data flows linearly from source text into the parser, which constructs the AST. This AST then serves as the foundational data structure for the verifier. The verifier acts as a bridge between two distinct type systems, extracting type information from the target Go source code and performing a deep, recursive comparison against the Grok AST. This decoupled design ensures that the parsing logic is entirely independent of the verification logic, allowing the AST to potentially be used for other tooling, such as code generation or language servers, in the future.

## Interface & Contract Map

While the Grok project currently relies on concrete data structures rather than abstract Go interfaces for its internal boundaries, the system is defined by several critical contracts that govern module interactions.

The primary structural contract for the entire system is defined by the AST file structure. The parser module acts as the sole producer of this contract, transforming raw source text into a valid syntax tree. The verifier module acts as the primary consumer, traversing this file structure to drive its analysis of the Go source code.

Another critical contract is the verification result structure exposed by the verifier module. This structure contains a collection of finding objects, serving as the external contract for any build pipeline, continuous integration system, or integrated development environment that consumes the output of the Grok toolchain.

Finally, the verifier heavily consumes the Go standard library's abstract syntax tree and parser interfaces to extract type information from the target codebase. In this capacity, the verifier acts as a client to the Go compiler's frontend, relying on these external interfaces to build its internal representation of the implementation's reality.

## Module Map

- **Core Data Structures**
  - [pkg/ast](pkg/ast/design.md)
- **Frontend Toolchain**
  - [pkg/parser](pkg/parser/design.md)
- **Analysis Engine**
  - [pkg/verifier](pkg/verifier/design.md)

### AST Module
The `ast` module is the foundational data model for the Grok language. It defines the Abstract Syntax Tree, capturing the hierarchical structure of `.grok` files, including blocks, type declarations, functions, and complex relations. A critical aspect of its design is the use of a tagged union pattern for type expressions, allowing for a flexible and recursive representation of Grok's rich type system. It manages no internal state, serving purely as a collection of data structures. It implements no external interfaces but provides the core types consumed by the parser and verifier.

### Parser Module
The `parser` module is responsible for lexical and syntactic analysis, transforming raw Grok source code into the structured AST. It employs a two-phase architecture with a hand-written, stateful lexer and a recursive descent parser. The lexer manages state to handle complex tokens like triple-quoted strings and significant newlines, while the parser manages the state of the token stream, occasionally using lookahead and state restoration to resolve ambiguities. The parser consumes raw text and produces the `ast.File` contract.

### Verifier Module
The `verifier` module is the structural integrity engine that detects architectural drift. It manages a complex internal state during execution, specifically the `goTypeInfo` map, which aggregates all structs, interfaces, and functions extracted from the target Go source code. The verifier consumes the `ast.File` contract produced by the parser and the Go AST produced by the standard library. It bridges the gap between these two models by normalizing naming conventions and type expressions, ultimately producing a structured `Result` containing any detected discrepancies.

## Integration Patterns & Workflows

The most critical cross-module workflow is the end-to-end verification process. When a user invokes the verifier on a `.grok` file, the workflow begins in the `verifier` module, which immediately delegates to the `parser` module. The `parser` instantiates its lexer, tokenizes the source, and constructs an `ast.File` tree, returning it to the verifier. The verifier then inspects the source annotations within the AST to locate the corresponding Go files. It invokes the Go standard library parser to build a Go AST and extracts the relevant type information into its internal state. Finally, the verifier recursively traverses the Grok AST, comparing each declaration against the extracted Go types, normalizing names and types on the fly, and accumulating findings into a final result.

A complex micro-workflow occurs during the structural comparison phase. When the verifier encounters a Grok type expression, it must compare this against a Go type. The verifier traverses the Grok type expression AST node, recursively building a normalized Go string representation. It simultaneously applies naming convention transformations, converting Grok's snake case identifiers to Go's Pascal case or camel case, ensuring that the conceptual model accurately maps to the idiomatic implementation.

## Dependency Overview

The dependency graph of the Grok project is strictly layered and unidirectional, preventing circular dependencies and ensuring a clean separation of concerns. At the base of the graph is the `ast` module, which is a leaf package depending only on the Go standard library. The `parser` module sits above the `ast`, depending on it to construct the syntax tree. At the top of the hierarchy is the `verifier` module, which depends on both the `parser` to read `.grok` files and the `ast` to traverse the resulting model. The verifier also introduces a heavy dependency on the Go standard library's `go/ast` and `go/parser` packages. This layered architecture constrains the flow of information: the AST has no knowledge of how it is parsed, and the parser has no knowledge of how the AST will be verified.
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/verifier/verifier.grok"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_b74ee0945967a137]
File: /Users/bill/projects/grok/pkg/verifier/verifier.grok (48 lines)

// verifier.grok — the Grok verifier's own understanding

grok Verifier {
  why: "Compares .grok understanding files against Go source code, reporting structural drift."

  doc "Architecture": """
    Parse .grok file → AST (via pkg/parser). For each source: annotation,
    parse the Go file(s) with go/ast. Aggregate all Go types, then compare
    structurally: structs, classes, enums, interfaces, functions.
    Deep type comparison: field types, param counts, return types.
    Package-prefix stripping for cross-package type references.
  """

  doc "Key Decisions": """
    Aggregate all source files before comparing (not per-file).
    Three severity levels: Error (missing/wrong type), Warning (extra fields), Info (undocumented Go methods).
    Enums check for typedef, struct, and interface forms in Go.
    Classes map to Go structs with methods; ctor params checked as fields.
    Grok T? maps to Go *T; Go zero-value optionals (string) should be string not string?.
  """

  // ---- Severity ----

  enum Severity { Error Warning Info }

  // ---- Findings ----

  struct Finding {
    Severity: Severity
    GrokFile: string
    GoFile:   string
    Message:  string
  }

  class Result() {
    Findings: [Finding]

    func ErrorCount(self) -> int
  }

  // ---- Public API ----

  func Verify(grokPath: string) -> (Result?, error)
    why: "Entry point. Parses .grok, finds source files, compares structurally."

  source: ["verifier.go"]
}
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "content": "# Verifier Module Design\n\n## Executive Summary\n\nThe `verifier` module serves as the structural integrity engine of the Grok system, acting as the bridge between high-level architectural declarations and concrete implementation. Its primary mission is to detect \"architectural drift\"—the inevitable divergence that occurs as a codebase evolves away from its original design. By performing a deep, recursive comparison between the Abstract Syntax Tree (AST) of a `.grok` specification and the AST of the corresponding Go source code, the verifier ensures that the developer's mental model remains synchronized with reality. It identifies mismatches in types, field names, function signatures, and interface implementations, reporting them as errors, warnings, or informational findings depending on their impact on system integrity.\n\n## File Inventory\n\n- [verifier.go](verifier.go): The primary implementation file containing the core verification logic, Go source analysis, and type normalization routines.\n- [verifier_test.go](verifier_test.go): The test suite for the module, featuring integration tests that run the verifier against the project's own `.grok` files and unit tests for naming convention transformations.\n- [verifier.grok](verifier.grok): The self-documenting Grok specification for the verifier module, defining its own structural contracts and architectural intent.\n\n## Architecture and Data Flow\n\nThe verifier operates as a multi-stage analysis pipeline that translates between two distinct type systems. The process is linear and stateless between calls, though it maintains significant internal state during a single verification run.\n\nThe workflow begins with **Grok Parsing**, where the module utilizes [pkg/parser](../parser/design.md) to transform a target `.grok` file into a [pkg/ast](../ast/design.md) representation. Once the Grok AST is available, the verifier enters the **Go Source Discovery** phase, inspecting the `source:` annotations within each `grok` block to identify the relevant Go files or directories.\n\nThe core of the analysis happens during **Go Type Extraction**. The verifier leverages the Go standard library's `go/parser` and `go/ast` packages to parse the identified source code. It traverses the resulting Go AST to populate an internal `goTypeInfo` structure, which serves as a comprehensive registry of all structs, interfaces, functions, and type definitions found across the specified sources. This aggregation allows the verifier to resolve types that may be spread across multiple files within the same package.\n\nFinally, the module performs a **Structural Comparison**. It recursively walks the Grok AST, comparing each declaration against the aggregated Go type information. This phase involves complex normalization, mapping Grok's snake_case identifiers to Go's PascalCase or camelCase conventions and converting Grok's recursive type expressions into a format compatible with Go's type system. Discrepancies are accumulated into a `Result` object containing a list of `Finding` records.\n\n## Interface Implementations\n\nThe `verifier` module does not implement any external interfaces defined within the Grok project. Instead, it acts as a high-level consumer of the [pkg/parser](../parser/design.md) and [pkg/ast](../ast/design.md) contracts. It also functions as a client to the Go standard library's AST interfaces, specifically consuming `*ast.File` and `*ast.GenDecl` nodes to build its internal model of the implementation.\n\n## Public API\n\nThe public API is centered around a single, powerful entry point designed for integration into developer workflows and CI/CD pipelines.\n\n- **Verify(grokPath string) (*Result, error)**: This is the primary \"front door\" of the module. It accepts a path to a `.grok` file, orchestrates the entire parsing and comparison pipeline, and returns a `Result` object. It handles file I/O and coordinates the interaction between the Grok parser and the Go source analyzer.\n- **Result**: A container for the findings of a verification run. It provides an `ErrorCount()` method, allowing callers to quickly determine if any critical architectural drift was detected that should fail a build.\n- **Finding**: A detailed record of a single discrepancy. Each finding includes a `Severity` (Error, Warning, or Info), the paths to the involved files, and a descriptive message explaining the nature of the drift.\n- **Severity**: An enumeration that classifies the impact of a finding. `Error` indicates a fundamental mismatch (e.g., a missing type or wrong field type), `Warning` indicates minor drift (e.g., extra fields in Go not documented in Grok), and `Info` provides context (e.g., additional methods in Go that are omitted from the Grok interface).\n\n## Implementation Details\n\n### Go Type Extraction\n\nThe extraction process is handled by `extractGoTypes`, which performs a deep traversal of the Go AST. It populates the `goTypeInfo` structure, which is the central repository of implementation facts.\n- **Structs**: The verifier captures field names and their types as strings. It also associates methods with their respective structs by inspecting function receivers.\n- **Interfaces**: It extracts method signatures, including parameter types and return values, to ensure full interface compliance.\n- **Functions**: Top-level functions are captured with their full signatures.\n- **Type Definitions**: Simple type aliases (e.g., `type MyInt int`) are tracked to allow the verifier to recognize them as valid types even if they don't have complex internal structure.\n\n### Type Mapping and Comparison\n\nBridging the gap between Grok and Go requires sophisticated normalization logic:\n- **Naming Conventions**: The `snakeToPascal` and `snakeToCamel` functions are used to match Grok's idiomatic snake_case identifiers to Go's PascalCase (for exported fields/methods) or camelCase (for unexported ones).\n- **Type Normalization**: The `grokTypeToGoString` function recursively converts Grok `TypeExpr` nodes into their Go string equivalents. For example, it maps Grok's optional types (`T?`) to Go pointers (`*T`) and Grok sequences (`[T]`) to Go slices (`[]T`).\n- **Fuzzy Matching**: The `typesMatch` function handles nuances such as the equivalence of `any` and `interface{}`. Crucially, it implements `stripPackagePrefix`, which allows the verifier to match unqualified types in Grok (e.g., `File`) to qualified types in Go (e.g., `ast.File`), facilitating cross-package verification.\n\n### Verification Logic\n\nThe verification logic is partitioned by declaration type, each with its own specific rules:\n- **Structs and Classes**: The verifier checks for field existence and type compatibility. It also identifies \"extra\" fields in the Go implementation that are missing from the Grok declaration, reporting them as warnings to encourage complete documentation. For classes, it additionally verifies that constructor parameters correspond to struct fields.\n- **Interfaces**: It ensures that every method declared in the Grok interface exists in the Go implementation with a matching signature.\n- **Functions**: It performs positional comparison of parameters and return types. It specifically handles Grok's `self` parameter in class methods, excluding it from the positional count when comparing against Go's receiver-based methods.\n\n## Dependencies\n\n- [pkg/ast](../ast/design.md): Provides the foundational data structures for representing the Grok language.\n- [pkg/parser](../parser/design.md): Used to transform Grok source text into the AST consumed by the verifier.\n- **Go Standard Library**: The verifier depends heavily on `go/ast`, `go/parser`, and `go/token` for its analysis of the Go implementation.\n\n## Technical Debt and Future Work\n\n- **Expression Verification**: The current implementation focuses on structural types. Future work should include verifying the `requires` and `ensures` contracts by parsing and analyzing the expressions within those blocks.\n- **Pointer Nuances**: While the verifier handles basic optional-to-pointer mapping, more robust handling of pointer vs. value receivers and complex indirection would reduce false positives in edge cases.\n- **Import Resolution**: The current `stripPackagePrefix` approach is a heuristic. A more robust system would resolve imports to ensure that a Grok type `File` actually refers to the same `ast.File` being used in the Go code.\n- **Performance Optimization**: For large projects, the verifier currently re-parses Go files for every `grok` block. Implementing a cache for parsed Go type information would significantly improve performance in large-scale applications.\n",
  "file_path": "pkg/verifier/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/grok/pkg/verifier/design.md`
**Size**: 8635 bytes
**Permissions**: -rw-r--r--
**Lines**: 72
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

Documentation complete.

---

