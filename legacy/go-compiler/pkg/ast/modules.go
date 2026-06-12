package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveModuleImports resolves all import declarations by parsing imported packages,
// prefixing their names with the import alias, and rewriting qualified references
// in the root package to use the prefixed names. Returns a single merged File
// with all packages flattened into one namespace.
//
// moduleRoot is the directory containing forge.mod.
// rootFile is the parsed AST of the root package (already merged if multi-file).
// parseFn is called to parse a .fg file, returning its AST.
func ResolveModuleImports(moduleRoot string, rootFile *File, parseFn func(path string) (*File, error)) (*File, error) {
	// Collect all imports from root package
	type importInfo struct {
		Alias string
		Path  string
	}
	var imports []importInfo
	for _, block := range rootFile.Blocks {
		for _, imp := range block.Imports {
			imports = append(imports, importInfo{Alias: imp.Alias, Path: imp.Path})
		}
	}

	if len(imports) == 0 {
		return rootFile, nil
	}

	// Parse and prefix each imported package
	importAliases := make(map[string]map[string]string) // alias → {origName → prefixedName}

	for _, imp := range imports {
		pkgDir := filepath.Join(moduleRoot, imp.Path)
		info, err := os.Stat(pkgDir)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("import %q: directory %s not found", imp.Alias, pkgDir)
		}

		// Parse all .fg files in the package directory
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			return nil, fmt.Errorf("import %q: %w", imp.Alias, err)
		}

		var pkgFiles []*File
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".fg") {
				continue
			}
			f, err := parseFn(filepath.Join(pkgDir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("import %q: parsing %s: %w", imp.Alias, entry.Name(), err)
			}
			pkgFiles = append(pkgFiles, f)
		}

		if len(pkgFiles) == 0 {
			return nil, fmt.Errorf("import %q: no .fg files in %s", imp.Alias, pkgDir)
		}

		// Merge package files
		merged := MergeFiles(pkgFiles)

		// Build name map and prefix all declarations
		nameMap := make(map[string]string) // origName → prefixedName
		prefix := imp.Alias + "_"

		for i := range merged.Blocks {
			prefixBlockDeclarations(&merged.Blocks[i], prefix, nameMap)
		}

		importAliases[imp.Alias] = nameMap

		// Rewrite internal references within the imported package
		for i := range merged.Blocks {
			rewriteBlockReferences(&merged.Blocks[i], nameMap)
		}

		// Append imported blocks to root file
		rootFile.Blocks = append(rootFile.Blocks, merged.Blocks...)
	}

	// Rewrite qualified references in root package (mylib.add → mylib_add)
	// and strip import declarations
	for i := range rootFile.Blocks {
		rewriteQualifiedAccess(&rootFile.Blocks[i], importAliases)
		rootFile.Blocks[i].Imports = nil
	}

	return rootFile, nil
}

// prefixBlockDeclarations adds prefix to all top-level declarations and builds nameMap.
func prefixBlockDeclarations(block *ForgeBlock, prefix string, nameMap map[string]string) {
	for i := range block.Functions {
		orig := block.Functions[i].Name
		block.Functions[i].Name = prefix + orig
		nameMap[orig] = block.Functions[i].Name
	}
	for i := range block.Structs {
		orig := block.Structs[i].Name
		block.Structs[i].Name = prefix + orig
		nameMap[orig] = block.Structs[i].Name
	}
	for i := range block.Classes {
		orig := block.Classes[i].Name
		block.Classes[i].Name = prefix + orig
		nameMap[orig] = block.Classes[i].Name
	}
	for i := range block.Enums {
		orig := block.Enums[i].Name
		block.Enums[i].Name = prefix + orig
		nameMap[orig] = block.Enums[i].Name
	}
	for i := range block.Interfaces {
		orig := block.Interfaces[i].Name
		block.Interfaces[i].Name = prefix + orig
		nameMap[orig] = block.Interfaces[i].Name
	}
	for i := range block.Constants {
		orig := block.Constants[i].Name
		block.Constants[i].Name = prefix + orig
		nameMap[orig] = block.Constants[i].Name
	}
	for i := range block.TypeAliases {
		orig := block.TypeAliases[i].Name
		block.TypeAliases[i].Name = prefix + orig
		nameMap[orig] = block.TypeAliases[i].Name
	}
}

// rewriteBlockReferences rewrites all name references within a block using nameMap.
func rewriteBlockReferences(block *ForgeBlock, nameMap map[string]string) {
	for i := range block.Functions {
		rewriteFuncReferences(&block.Functions[i], nameMap)
	}
	for i := range block.Structs {
		rewriteStructReferences(&block.Structs[i], nameMap)
	}
	for i := range block.Classes {
		for j := range block.Classes[i].Fields {
			rewriteTypeExprReferences(&block.Classes[i].Fields[j].Type, nameMap)
		}
		for k := range block.Classes[i].Methods {
			rewriteFuncReferences(&block.Classes[i].Methods[k], nameMap)
		}
	}
	for i := range block.Enums {
		for j := range block.Enums[i].Variants {
			for k := range block.Enums[i].Variants[j].Fields {
				rewriteTypeExprReferences(&block.Enums[i].Variants[j].Fields[k].Type, nameMap)
			}
		}
	}
	for i := range block.Interfaces {
		for j := range block.Interfaces[i].Methods {
			rewriteFuncReferences(&block.Interfaces[i].Methods[j], nameMap)
		}
	}
	for i := range block.ImplBlocks {
		if newName, ok := nameMap[block.ImplBlocks[i].InterfaceName]; ok {
			block.ImplBlocks[i].InterfaceName = newName
		}
		if newName, ok := nameMap[block.ImplBlocks[i].ForType]; ok {
			block.ImplBlocks[i].ForType = newName
		}
	}
}

// rewriteFuncReferences rewrites type and expression references within a function.
func rewriteFuncReferences(fn *FuncDecl, nameMap map[string]string) {
	for i := range fn.Params {
		rewriteTypeExprReferences(&fn.Params[i].Type, nameMap)
	}
	if fn.ReturnType != nil {
		rewriteTypeExprReferences(fn.ReturnType, nameMap)
	}
	if fn.ReceiverType != "" {
		if newName, ok := nameMap[fn.ReceiverType]; ok {
			fn.ReceiverType = newName
		}
	}
	if fn.Body != nil {
		for i := range fn.Body.Stmts {
			rewriteStmtReferences(&fn.Body.Stmts[i], nameMap)
		}
	}
}

// rewriteStructReferences rewrites type references within struct fields.
func rewriteStructReferences(s *StructDecl, nameMap map[string]string) {
	for i := range s.Fields {
		rewriteTypeExprReferences(&s.Fields[i].Type, nameMap)
	}
}

// rewriteTypeExprReferences rewrites type names using nameMap.
func rewriteTypeExprReferences(te *TypeExpr, nameMap map[string]string) {
	if te == nil {
		return
	}
	switch te.Kind {
	case TypeNamed:
		// Parser stores NamedType as value (not pointer) — need to extract, modify, reassign
		if nt, ok := te.Data.(NamedType); ok {
			if newName, ok := nameMap[nt.Name]; ok {
				nt.Name = newName
			}
			for i := range nt.Args {
				rewriteTypeExprReferences(&nt.Args[i], nameMap)
			}
			te.Data = nt // write back modified value
		} else if ntp, ok := te.Data.(*NamedType); ok && ntp != nil {
			if newName, ok := nameMap[ntp.Name]; ok {
				ntp.Name = newName
			}
			for i := range ntp.Args {
				rewriteTypeExprReferences(&ntp.Args[i], nameMap)
			}
		}
	case TypeOptional:
		if ot, ok := te.Data.(OptionalType); ok {
			rewriteTypeExprReferences(&ot.Inner, nameMap)
			te.Data = ot
		} else if otp, ok := te.Data.(*OptionalType); ok {
			rewriteTypeExprReferences(&otp.Inner, nameMap)
		}
	case TypeSequence:
		if st, ok := te.Data.(SequenceType); ok {
			rewriteTypeExprReferences(&st.Elem, nameMap)
			te.Data = st
		} else if stp, ok := te.Data.(*SequenceType); ok {
			rewriteTypeExprReferences(&stp.Elem, nameMap)
		}
	case TypeTuple:
		if tt, ok := te.Data.(TupleType); ok {
			for i := range tt.Fields {
				rewriteTypeExprReferences(&tt.Fields[i].Type, nameMap)
			}
			te.Data = tt
		} else if ttp, ok := te.Data.(*TupleType); ok {
			for i := range ttp.Fields {
				rewriteTypeExprReferences(&ttp.Fields[i].Type, nameMap)
			}
		}
	case TypeFunc:
		if ft, ok := te.Data.(FuncType); ok {
			for i := range ft.Params {
				rewriteTypeExprReferences(&ft.Params[i], nameMap)
			}
			rewriteTypeExprReferences(&ft.Return, nameMap)
			te.Data = ft
		} else if ftp, ok := te.Data.(*FuncType); ok {
			for i := range ftp.Params {
				rewriteTypeExprReferences(&ftp.Params[i], nameMap)
			}
			rewriteTypeExprReferences(&ftp.Return, nameMap)
		}
	case TypeMap:
		if mt, ok := te.Data.(MapType); ok {
			rewriteTypeExprReferences(&mt.Key, nameMap)
			rewriteTypeExprReferences(&mt.Value, nameMap)
			te.Data = mt
		} else if mtp, ok := te.Data.(*MapType); ok {
			rewriteTypeExprReferences(&mtp.Key, nameMap)
			rewriteTypeExprReferences(&mtp.Value, nameMap)
		}
	case TypeChannel:
		if ct, ok := te.Data.(ChannelType); ok {
			rewriteTypeExprReferences(&ct.Elem, nameMap)
			te.Data = ct
		} else if ctp, ok := te.Data.(*ChannelType); ok {
			rewriteTypeExprReferences(&ctp.Elem, nameMap)
		}
	case TypeGenerator:
		if gt, ok := te.Data.(GeneratorType); ok {
			rewriteTypeExprReferences(&gt.Elem, nameMap)
			te.Data = gt
		} else if gtp, ok := te.Data.(*GeneratorType); ok {
			rewriteTypeExprReferences(&gtp.Elem, nameMap)
		}
	}
}

// rewriteStmtReferences rewrites name references in statements.
func rewriteStmtReferences(stmt *Stmt, nameMap map[string]string) {
	if stmt == nil {
		return
	}
	switch s := stmt.Data.(type) {
	case *VarDeclStmt:
		if s.Type != nil {
			rewriteTypeExprReferences(s.Type, nameMap)
		}
		if s.Value != nil {
			rewriteExprReferences(s.Value, nameMap)
		}
	case *AssignStmt:
		rewriteExprReferences(&s.Target, nameMap)
		rewriteExprReferences(&s.Value, nameMap)
	case *ReturnStmt:
		if s.Value != nil {
			rewriteExprReferences(s.Value, nameMap)
		}
	case *ExprStmt:
		rewriteExprReferences(&s.Expr, nameMap)
	case *IfStmt:
		rewriteExprReferences(&s.Condition, nameMap)
		for i := range s.Then.Stmts {
			rewriteStmtReferences(&s.Then.Stmts[i], nameMap)
		}
		if s.Else != nil {
			for i := range s.Else.Stmts {
				rewriteStmtReferences(&s.Else.Stmts[i], nameMap)
			}
		}
		for i := range s.ElseIfs {
			rewriteExprReferences(&s.ElseIfs[i].Condition, nameMap)
			for j := range s.ElseIfs[i].Body.Stmts {
				rewriteStmtReferences(&s.ElseIfs[i].Body.Stmts[j], nameMap)
			}
		}
	case *WhileStmt:
		rewriteExprReferences(&s.Condition, nameMap)
		for i := range s.Body.Stmts {
			rewriteStmtReferences(&s.Body.Stmts[i], nameMap)
		}
	case *ForStmt:
		rewriteExprReferences(&s.Collection, nameMap)
		for i := range s.Body.Stmts {
			rewriteStmtReferences(&s.Body.Stmts[i], nameMap)
		}
	case *MatchStmt:
		rewriteExprReferences(&s.Value, nameMap)
		for i := range s.Arms {
			for j := range s.Arms[i].Body.Stmts {
				rewriteStmtReferences(&s.Arms[i].Body.Stmts[j], nameMap)
			}
			rewritePatternReferences(&s.Arms[i].Pattern, nameMap)
			for k := range s.Arms[i].Patterns {
				rewritePatternReferences(&s.Arms[i].Patterns[k], nameMap)
			}
			if s.Arms[i].Guard != nil {
				rewriteExprReferences(s.Arms[i].Guard, nameMap)
			}
		}
	case *SpawnStmt:
		for i := range s.Body.Stmts {
			rewriteStmtReferences(&s.Body.Stmts[i], nameMap)
		}
	case *LockStmt:
		rewriteExprReferences(&s.Mutex, nameMap)
		for i := range s.Body.Stmts {
			rewriteStmtReferences(&s.Body.Stmts[i], nameMap)
		}
	case *SelectStmt:
		for i := range s.Cases {
			if s.Cases[i].Expr != nil {
				rewriteExprReferences(s.Cases[i].Expr, nameMap)
			}
			for j := range s.Cases[i].Body.Stmts {
				rewriteStmtReferences(&s.Cases[i].Body.Stmts[j], nameMap)
			}
		}
	}
}

// rewriteExprReferences rewrites name references in expressions.
func rewriteExprReferences(expr *Expr, nameMap map[string]string) {
	if expr == nil {
		return
	}
	switch expr.Kind {
	case ExprIdent:
		if d, ok := expr.Data.(*IdentExpr); ok {
			if newName, ok := nameMap[d.Name]; ok {
				d.Name = newName
			}
		}
	case ExprCall:
		if d, ok := expr.Data.(*CallExpr); ok {
			rewriteExprReferences(&d.Func, nameMap)
			for i := range d.TypeArgs {
				rewriteTypeExprReferences(&d.TypeArgs[i], nameMap)
			}
			for i := range d.Args {
				rewriteExprReferences(&d.Args[i], nameMap)
			}
		}
	case ExprMethodCall:
		if d, ok := expr.Data.(*MethodCallExpr); ok {
			rewriteExprReferences(&d.Receiver, nameMap)
			for i := range d.TypeArgs {
				rewriteTypeExprReferences(&d.TypeArgs[i], nameMap)
			}
			for i := range d.Args {
				rewriteExprReferences(&d.Args[i], nameMap)
			}
		}
	case ExprFieldAccess:
		if d, ok := expr.Data.(*FieldAccessExpr); ok {
			rewriteExprReferences(&d.Receiver, nameMap)
		}
	case ExprIndex:
		if d, ok := expr.Data.(*IndexExpr); ok {
			rewriteExprReferences(&d.Receiver, nameMap)
			rewriteExprReferences(&d.Index, nameMap)
		}
	case ExprUnary:
		if d, ok := expr.Data.(*UnaryExpr); ok {
			rewriteExprReferences(&d.Operand, nameMap)
		}
	case ExprBinary:
		if d, ok := expr.Data.(*BinaryExpr); ok {
			rewriteExprReferences(&d.Left, nameMap)
			rewriteExprReferences(&d.Right, nameMap)
		}
	case ExprTupleLit:
		if d, ok := expr.Data.(*TupleLitExpr); ok {
			for i := range d.Elems {
				rewriteExprReferences(&d.Elems[i], nameMap)
			}
		}
	case ExprListLit:
		if d, ok := expr.Data.(*ListLitExpr); ok {
			for i := range d.Elems {
				rewriteExprReferences(&d.Elems[i], nameMap)
			}
		}
	case ExprMapLit:
		if d, ok := expr.Data.(*MapLitExpr); ok {
			for i := range d.Entries {
				rewriteExprReferences(&d.Entries[i].Key, nameMap)
				rewriteExprReferences(&d.Entries[i].Value, nameMap)
			}
		}
	case ExprLambda:
		if d, ok := expr.Data.(*LambdaExpr); ok {
			for i := range d.Params {
				rewriteTypeExprReferences(&d.Params[i].Type, nameMap)
			}
			if d.ReturnType != nil {
				rewriteTypeExprReferences(d.ReturnType, nameMap)
			}
			for i := range d.Body.Stmts {
				rewriteStmtReferences(&d.Body.Stmts[i], nameMap)
			}
		}
	case ExprMatch:
		if d, ok := expr.Data.(*MatchStmt); ok {
			rewriteExprReferences(&d.Value, nameMap)
			for i := range d.Arms {
				for j := range d.Arms[i].Body.Stmts {
					rewriteStmtReferences(&d.Arms[i].Body.Stmts[j], nameMap)
				}
				rewritePatternReferences(&d.Arms[i].Pattern, nameMap)
				if d.Arms[i].Guard != nil {
					rewriteExprReferences(d.Arms[i].Guard, nameMap)
				}
			}
		}
	case ExprStructLit:
		if d, ok := expr.Data.(*StructLitExpr); ok {
			if newName, ok := nameMap[d.TypeName]; ok {
				d.TypeName = newName
			}
			for i := range d.TypeArgs {
				rewriteTypeExprReferences(&d.TypeArgs[i], nameMap)
			}
			for i := range d.Fields {
				rewriteExprReferences(&d.Fields[i].Value, nameMap)
			}
		}
	case ExprCast:
		if d, ok := expr.Data.(*CastExpr); ok {
			rewriteExprReferences(&d.Operand, nameMap)
			rewriteTypeExprReferences(&d.TargetType, nameMap)
		}
	case ExprUnwrap:
		if d, ok := expr.Data.(*UnwrapExpr); ok {
			rewriteExprReferences(&d.Operand, nameMap)
		}
	case ExprSlice:
		if d, ok := expr.Data.(*SliceExpr); ok {
			rewriteExprReferences(&d.Receiver, nameMap)
			if d.Low != nil {
				rewriteExprReferences(d.Low, nameMap)
			}
			if d.High != nil {
				rewriteExprReferences(d.High, nameMap)
			}
		}
	case ExprTry:
		if d, ok := expr.Data.(*TryExpr); ok {
			rewriteExprReferences(&d.Operand, nameMap)
		}
	case ExprIs:
		if d, ok := expr.Data.(*IsExpr); ok {
			rewriteExprReferences(&d.Operand, nameMap)
		}
	case ExprIfElse:
		if d, ok := expr.Data.(*IfElseExpr); ok {
			rewriteExprReferences(&d.Cond, nameMap)
			for i := range d.Then.Stmts {
				rewriteStmtReferences(&d.Then.Stmts[i], nameMap)
			}
			for i := range d.Else.Stmts {
				rewriteStmtReferences(&d.Else.Stmts[i], nameMap)
			}
		}
	case ExprStringInterp:
		if d, ok := expr.Data.(*StringInterpExpr); ok {
			for i := range d.Parts {
				rewriteExprReferences(&d.Parts[i], nameMap)
			}
		}
	}
}

// rewritePatternReferences rewrites type names in match patterns.
func rewritePatternReferences(pat *Pattern, nameMap map[string]string) {
	if pat == nil {
		return
	}
	if vp, ok := pat.Data.(*VariantPattern); ok {
		if newName, ok := nameMap[vp.Name]; ok {
			vp.Name = newName
		}
		for i := range vp.Bindings {
			rewritePatternReferences(&vp.Bindings[i], nameMap)
		}
	}
	if tp, ok := pat.Data.(*TuplePattern); ok {
		for i := range tp.Elems {
			rewritePatternReferences(&tp.Elems[i], nameMap)
		}
	}
}

// rewriteQualifiedAccess rewrites pkg.name expressions to pkg_name in a block.
func rewriteQualifiedAccess(block *ForgeBlock, importAliases map[string]map[string]string) {
	for i := range block.Functions {
		rewriteQualifiedInFunc(&block.Functions[i], importAliases)
	}
	for i := range block.Structs {
		for j := range block.Structs[i].Fields {
			rewriteQualifiedInTypeExpr(&block.Structs[i].Fields[j].Type, importAliases)
		}
	}
	for i := range block.Classes {
		for j := range block.Classes[i].Fields {
			rewriteQualifiedInTypeExpr(&block.Classes[i].Fields[j].Type, importAliases)
		}
		for k := range block.Classes[i].Methods {
			rewriteQualifiedInFunc(&block.Classes[i].Methods[k], importAliases)
		}
	}
}

func rewriteQualifiedInFunc(fn *FuncDecl, importAliases map[string]map[string]string) {
	for i := range fn.Params {
		rewriteQualifiedInTypeExpr(&fn.Params[i].Type, importAliases)
	}
	if fn.ReturnType != nil {
		rewriteQualifiedInTypeExpr(fn.ReturnType, importAliases)
	}
	if fn.Body != nil {
		for i := range fn.Body.Stmts {
			rewriteQualifiedInStmt(&fn.Body.Stmts[i], importAliases)
		}
	}
}

// rewriteQualifiedInTypeExpr handles qualified type names like mylib.Point → mylib_Point
func rewriteQualifiedInTypeExpr(te *TypeExpr, importAliases map[string]map[string]string) {
	if te == nil {
		return
	}
	switch te.Kind {
	case TypeNamed:
		if nt, ok := te.Data.(NamedType); ok {
			if parts := strings.SplitN(nt.Name, ".", 2); len(parts) == 2 {
				if nameMap, ok := importAliases[parts[0]]; ok {
					if prefixed, ok := nameMap[parts[1]]; ok {
						nt.Name = prefixed
					} else {
						nt.Name = parts[0] + "_" + parts[1]
					}
				}
			}
			for i := range nt.Args {
				rewriteQualifiedInTypeExpr(&nt.Args[i], importAliases)
			}
			te.Data = nt
		} else if ntp, ok := te.Data.(*NamedType); ok && ntp != nil {
			if parts := strings.SplitN(ntp.Name, ".", 2); len(parts) == 2 {
				if nameMap, ok := importAliases[parts[0]]; ok {
					if prefixed, ok := nameMap[parts[1]]; ok {
						ntp.Name = prefixed
					} else {
						ntp.Name = parts[0] + "_" + parts[1]
					}
				}
			}
			for i := range ntp.Args {
				rewriteQualifiedInTypeExpr(&ntp.Args[i], importAliases)
			}
		}
	case TypeOptional:
		if ot, ok := te.Data.(OptionalType); ok {
			rewriteQualifiedInTypeExpr(&ot.Inner, importAliases)
			te.Data = ot
		} else if otp, ok := te.Data.(*OptionalType); ok {
			rewriteQualifiedInTypeExpr(&otp.Inner, importAliases)
		}
	case TypeSequence:
		if st, ok := te.Data.(SequenceType); ok {
			rewriteQualifiedInTypeExpr(&st.Elem, importAliases)
			te.Data = st
		} else if stp, ok := te.Data.(*SequenceType); ok {
			rewriteQualifiedInTypeExpr(&stp.Elem, importAliases)
		}
	case TypeTuple:
		if tt, ok := te.Data.(TupleType); ok {
			for i := range tt.Fields {
				rewriteQualifiedInTypeExpr(&tt.Fields[i].Type, importAliases)
			}
			te.Data = tt
		} else if ttp, ok := te.Data.(*TupleType); ok {
			for i := range ttp.Fields {
				rewriteQualifiedInTypeExpr(&ttp.Fields[i].Type, importAliases)
			}
		}
	case TypeFunc:
		if ft, ok := te.Data.(FuncType); ok {
			for i := range ft.Params {
				rewriteQualifiedInTypeExpr(&ft.Params[i], importAliases)
			}
			rewriteQualifiedInTypeExpr(&ft.Return, importAliases)
			te.Data = ft
		} else if ftp, ok := te.Data.(*FuncType); ok {
			for i := range ftp.Params {
				rewriteQualifiedInTypeExpr(&ftp.Params[i], importAliases)
			}
			rewriteQualifiedInTypeExpr(&ftp.Return, importAliases)
		}
	case TypeMap:
		if mt, ok := te.Data.(MapType); ok {
			rewriteQualifiedInTypeExpr(&mt.Key, importAliases)
			rewriteQualifiedInTypeExpr(&mt.Value, importAliases)
			te.Data = mt
		} else if mtp, ok := te.Data.(*MapType); ok {
			rewriteQualifiedInTypeExpr(&mtp.Key, importAliases)
			rewriteQualifiedInTypeExpr(&mtp.Value, importAliases)
		}
	case TypeChannel:
		if ct, ok := te.Data.(ChannelType); ok {
			rewriteQualifiedInTypeExpr(&ct.Elem, importAliases)
			te.Data = ct
		} else if ctp, ok := te.Data.(*ChannelType); ok {
			rewriteQualifiedInTypeExpr(&ctp.Elem, importAliases)
		}
	case TypeGenerator:
		if gt, ok := te.Data.(GeneratorType); ok {
			rewriteQualifiedInTypeExpr(&gt.Elem, importAliases)
			te.Data = gt
		} else if gtp, ok := te.Data.(*GeneratorType); ok {
			rewriteQualifiedInTypeExpr(&gtp.Elem, importAliases)
		}
	}
}

func rewriteQualifiedInStmt(stmt *Stmt, importAliases map[string]map[string]string) {
	if stmt == nil {
		return
	}
	switch s := stmt.Data.(type) {
	case *VarDeclStmt:
		if s.Type != nil {
			rewriteQualifiedInTypeExpr(s.Type, importAliases)
		}
		if s.Value != nil {
			rewriteQualifiedInExpr(s.Value, importAliases)
		}
	case *AssignStmt:
		rewriteQualifiedInExpr(&s.Target, importAliases)
		rewriteQualifiedInExpr(&s.Value, importAliases)
	case *ReturnStmt:
		if s.Value != nil {
			rewriteQualifiedInExpr(s.Value, importAliases)
		}
	case *ExprStmt:
		rewriteQualifiedInExpr(&s.Expr, importAliases)
	case *IfStmt:
		rewriteQualifiedInExpr(&s.Condition, importAliases)
		for i := range s.Then.Stmts {
			rewriteQualifiedInStmt(&s.Then.Stmts[i], importAliases)
		}
		if s.Else != nil {
			for i := range s.Else.Stmts {
				rewriteQualifiedInStmt(&s.Else.Stmts[i], importAliases)
			}
		}
		for i := range s.ElseIfs {
			rewriteQualifiedInExpr(&s.ElseIfs[i].Condition, importAliases)
			for j := range s.ElseIfs[i].Body.Stmts {
				rewriteQualifiedInStmt(&s.ElseIfs[i].Body.Stmts[j], importAliases)
			}
		}
	case *WhileStmt:
		rewriteQualifiedInExpr(&s.Condition, importAliases)
		for i := range s.Body.Stmts {
			rewriteQualifiedInStmt(&s.Body.Stmts[i], importAliases)
		}
	case *ForStmt:
		rewriteQualifiedInExpr(&s.Collection, importAliases)
		for i := range s.Body.Stmts {
			rewriteQualifiedInStmt(&s.Body.Stmts[i], importAliases)
		}
	case *MatchStmt:
		rewriteQualifiedInExpr(&s.Value, importAliases)
		for i := range s.Arms {
			for j := range s.Arms[i].Body.Stmts {
				rewriteQualifiedInStmt(&s.Arms[i].Body.Stmts[j], importAliases)
			}
			if s.Arms[i].Guard != nil {
				rewriteExprReferences(s.Arms[i].Guard, nil) // no-op with nil map
			}
		}
	case *SpawnStmt:
		for i := range s.Body.Stmts {
			rewriteQualifiedInStmt(&s.Body.Stmts[i], importAliases)
		}
	case *LockStmt:
		rewriteQualifiedInExpr(&s.Mutex, importAliases)
		for i := range s.Body.Stmts {
			rewriteQualifiedInStmt(&s.Body.Stmts[i], importAliases)
		}
	case *SelectStmt:
		for i := range s.Cases {
			if s.Cases[i].Expr != nil {
				rewriteQualifiedInExpr(s.Cases[i].Expr, importAliases)
			}
			for j := range s.Cases[i].Body.Stmts {
				rewriteQualifiedInStmt(&s.Cases[i].Body.Stmts[j], importAliases)
			}
		}
	}
}

// rewriteQualifiedInExpr converts pkg.name references to pkg_name.
func rewriteQualifiedInExpr(expr *Expr, importAliases map[string]map[string]string) {
	if expr == nil {
		return
	}
	switch expr.Kind {
	case ExprFieldAccess:
		if d, ok := expr.Data.(*FieldAccessExpr); ok {
			if d.Receiver.Kind == ExprIdent {
				if ident, ok2 := d.Receiver.Data.(*IdentExpr); ok2 {
					if nameMap, ok3 := importAliases[ident.Name]; ok3 {
						prefixed := ident.Name + "_" + d.Field
						if p, ok4 := nameMap[d.Field]; ok4 {
							prefixed = p
						}
						expr.Kind = ExprIdent
						expr.Data = &IdentExpr{Name: prefixed}
						return
					}
				}
			}
			rewriteQualifiedInExpr(&d.Receiver, importAliases)
		}
	case ExprMethodCall:
		if d, ok := expr.Data.(*MethodCallExpr); ok {
			if d.Receiver.Kind == ExprIdent {
				if ident, ok2 := d.Receiver.Data.(*IdentExpr); ok2 {
					if nameMap, ok3 := importAliases[ident.Name]; ok3 {
						prefixed := ident.Name + "_" + d.Method
						if p, ok4 := nameMap[d.Method]; ok4 {
							prefixed = p
						}
						funcExpr := Expr{
							Kind: ExprIdent,
							Data: &IdentExpr{Name: prefixed},
							Span: d.Receiver.Span,
						}
						// Rewrite args first
						for i := range d.Args {
							rewriteQualifiedInExpr(&d.Args[i], importAliases)
						}
						expr.Kind = ExprCall
						expr.Data = &CallExpr{
							Func:     funcExpr,
							TypeArgs: d.TypeArgs,
							Args:     d.Args,
							MutArgs:  d.MutArgs,
						}
						return
					}
				}
			}
			rewriteQualifiedInExpr(&d.Receiver, importAliases)
			for i := range d.Args {
				rewriteQualifiedInExpr(&d.Args[i], importAliases)
			}
		}
	case ExprCall:
		if d, ok := expr.Data.(*CallExpr); ok {
			rewriteQualifiedInExpr(&d.Func, importAliases)
			for i := range d.Args {
				rewriteQualifiedInExpr(&d.Args[i], importAliases)
			}
		}
	case ExprIndex:
		if d, ok := expr.Data.(*IndexExpr); ok {
			rewriteQualifiedInExpr(&d.Receiver, importAliases)
			rewriteQualifiedInExpr(&d.Index, importAliases)
		}
	case ExprUnary:
		if d, ok := expr.Data.(*UnaryExpr); ok {
			rewriteQualifiedInExpr(&d.Operand, importAliases)
		}
	case ExprBinary:
		if d, ok := expr.Data.(*BinaryExpr); ok {
			rewriteQualifiedInExpr(&d.Left, importAliases)
			rewriteQualifiedInExpr(&d.Right, importAliases)
		}
	case ExprTupleLit:
		if d, ok := expr.Data.(*TupleLitExpr); ok {
			for i := range d.Elems {
				rewriteQualifiedInExpr(&d.Elems[i], importAliases)
			}
		}
	case ExprListLit:
		if d, ok := expr.Data.(*ListLitExpr); ok {
			for i := range d.Elems {
				rewriteQualifiedInExpr(&d.Elems[i], importAliases)
			}
		}
	case ExprMapLit:
		if d, ok := expr.Data.(*MapLitExpr); ok {
			for i := range d.Entries {
				rewriteQualifiedInExpr(&d.Entries[i].Key, importAliases)
				rewriteQualifiedInExpr(&d.Entries[i].Value, importAliases)
			}
		}
	case ExprLambda:
		if d, ok := expr.Data.(*LambdaExpr); ok {
			for i := range d.Params {
				rewriteQualifiedInTypeExpr(&d.Params[i].Type, importAliases)
			}
			if d.ReturnType != nil {
				rewriteQualifiedInTypeExpr(d.ReturnType, importAliases)
			}
			for i := range d.Body.Stmts {
				rewriteQualifiedInStmt(&d.Body.Stmts[i], importAliases)
			}
		}
	case ExprMatch:
		if d, ok := expr.Data.(*MatchStmt); ok {
			rewriteQualifiedInExpr(&d.Value, importAliases)
			for i := range d.Arms {
				for j := range d.Arms[i].Body.Stmts {
					rewriteQualifiedInStmt(&d.Arms[i].Body.Stmts[j], importAliases)
				}
			}
		}
	case ExprStructLit:
		if d, ok := expr.Data.(*StructLitExpr); ok {
			if parts := strings.SplitN(d.TypeName, ".", 2); len(parts) == 2 {
				if nameMap, ok2 := importAliases[parts[0]]; ok2 {
					if prefixed, ok3 := nameMap[parts[1]]; ok3 {
						d.TypeName = prefixed
					} else {
						d.TypeName = parts[0] + "_" + parts[1]
					}
				}
			}
			for i := range d.Fields {
				rewriteQualifiedInExpr(&d.Fields[i].Value, importAliases)
			}
		}
	case ExprCast:
		if d, ok := expr.Data.(*CastExpr); ok {
			rewriteQualifiedInExpr(&d.Operand, importAliases)
			rewriteQualifiedInTypeExpr(&d.TargetType, importAliases)
		}
	case ExprUnwrap:
		if d, ok := expr.Data.(*UnwrapExpr); ok {
			rewriteQualifiedInExpr(&d.Operand, importAliases)
		}
	case ExprSlice:
		if d, ok := expr.Data.(*SliceExpr); ok {
			rewriteQualifiedInExpr(&d.Receiver, importAliases)
			if d.Low != nil {
				rewriteQualifiedInExpr(d.Low, importAliases)
			}
			if d.High != nil {
				rewriteQualifiedInExpr(d.High, importAliases)
			}
		}
	case ExprTry:
		if d, ok := expr.Data.(*TryExpr); ok {
			rewriteQualifiedInExpr(&d.Operand, importAliases)
		}
	case ExprIs:
		if d, ok := expr.Data.(*IsExpr); ok {
			rewriteQualifiedInExpr(&d.Operand, importAliases)
		}
	case ExprIfElse:
		if d, ok := expr.Data.(*IfElseExpr); ok {
			rewriteQualifiedInExpr(&d.Cond, importAliases)
			for i := range d.Then.Stmts {
				rewriteQualifiedInStmt(&d.Then.Stmts[i], importAliases)
			}
			for i := range d.Else.Stmts {
				rewriteQualifiedInStmt(&d.Else.Stmts[i], importAliases)
			}
		}
	case ExprStringInterp:
		if d, ok := expr.Data.(*StringInterpExpr); ok {
			for i := range d.Parts {
				rewriteQualifiedInExpr(&d.Parts[i], importAliases)
			}
		}
	}
}
