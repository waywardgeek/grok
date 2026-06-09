package ast

// DesugarInterfaceEmbeds flattens embedded interfaces by copying fields, methods,
// and destructors from the embedded interface into the embedding interface,
// substituting type parameters.
//
//	interface DoublyLinked<P, C> {
//	  field P.first: C?
//	  destructor C { ... }
//	}
//	interface OwningList<P, C> {
//	  embed DoublyLinked<P, C>
//	  destructor P { ... }
//	}
//
// becomes (OwningList gets DoublyLinked's fields and destructor C, plus its own destructor P):
//
//	interface OwningList<P, C> {
//	  field P.first: C?
//	  destructor C { ... }
//	  destructor P { ... }
//	}
func DesugarInterfaceEmbeds(file *File) {
	// Build index of all interfaces by name
	ifaceByName := map[string]*InterfaceDecl{}
	for bi := range file.Blocks {
		block := &file.Blocks[bi]
		for ii := range block.Interfaces {
			iface := &block.Interfaces[ii]
			ifaceByName[iface.Name] = iface
		}
	}

	for bi := range file.Blocks {
		block := &file.Blocks[bi]
		for ii := range block.Interfaces {
			iface := &block.Interfaces[ii]
			for _, emb := range iface.Embeds {
				parent, ok := ifaceByName[emb.Name]
				if !ok {
					continue // unknown interface — checker will report error
				}

				// Build type param substitution map: parent.TypeParams[i] → emb.TypeArgs[i]
				typeMap := map[string]string{}
				for i, tp := range parent.TypeParams {
					if i < len(emb.TypeArgs) {
						if emb.TypeArgs[i].Kind == TypeNamed {
							switch nt := emb.TypeArgs[i].Data.(type) {
							case NamedType:
								typeMap[tp.Name] = nt.Name
							case *NamedType:
								typeMap[tp.Name] = nt.Name
							}
						}
					}
				}

				// Copy fields with type param substitution
				for _, f := range parent.Fields {
					newField := f
					if mapped, ok := typeMap[f.TypeParam]; ok {
						newField.TypeParam = mapped
					}
					newField.Type = substituteTypeParamsInTypeExprCopy(f.Type, typeMap)
					iface.Fields = append(iface.Fields, newField)
				}

				// NOTE: Methods (standalone functions like dll_append) are NOT copied.
				// They remain on the embedded interface and are extracted by
				// DesugarDefaultImpls from there. Copying would cause duplicates.

				// Copy destructors with type param substitution
				for _, d := range parent.Destructors {
					newDestr := d
					if mapped, ok := typeMap[d.TypeParam]; ok {
						newDestr.TypeParam = mapped
					}
					newDestr.Body = deepCopyBlock(d.Body)
					substituteTypeParamsInBlock(&newDestr.Body, typeMap)
					iface.Destructors = append(iface.Destructors, newDestr)
				}
			}
		}
	}
}

// substituteTypeParamsInTypeExprCopy returns a copy of te with type params substituted.
func substituteTypeParamsInTypeExprCopy(te TypeExpr, typeMap map[string]string) TypeExpr {
	result := te
	substituteTypeParamsInTypeExpr(&result, typeMap)
	return result
}

// DesugarDefaultImpls extracts interface methods with bodies into top-level
// functions with relational where clauses. This must run before the checker.
//
//	interface Graph<G, N, E> {
//	  func G.nodes(self) -> [N]
//	  pub func count_edges(graph: G) -> i32 { ... }
//	}
//
// becomes:
//
//	pub func count_edges<G, N, E>(graph: G) -> i32 where Graph<G, N, E> { ... }
func DesugarDefaultImpls(file *File) {
	for bi := range file.Blocks {
		block := &file.Blocks[bi]
		for ii := range block.Interfaces {
			iface := &block.Interfaces[ii]
			var kept []FuncDecl
			for _, m := range iface.Methods {
				if m.Body != nil {
					// Extract as top-level function with interface type params + where clause
					fn := m
					fn.ReceiverType = "" // not a typed method anymore

					// Add interface type params
					for _, tp := range iface.TypeParams {
						fn.TypeParams = append(fn.TypeParams, TypeParam{
							Name:       tp.Name,
							Constraint: tp.Constraint,
						})
					}

					// Add relational where clause: where Graph<G, N, E>
					var typeArgs []TypeExpr
					for _, tp := range iface.TypeParams {
						typeArgs = append(typeArgs, TypeExpr{
							Kind: TypeNamed,
							Data: NamedType{Name: tp.Name},
						})
					}
					fn.Where = append(fn.Where, WhereClause{
						Constraint: iface.Name,
						TypeArgs:   typeArgs,
					})

					block.Functions = append(block.Functions, fn)
				} else {
					kept = append(kept, m)
				}
			}
			iface.Methods = kept
		}
	}
}

// DesugarInterfaceFields converts interface field declarations into getter/setter
// methods on the interface. Must run before DesugarRelations and DesugarDefaultImpls.
//
//	field P.first: C?
//
// becomes:
//
//	func P.first(self) -> C?
//	func P.set_first(mut self, val: C?)
func DesugarInterfaceFields(file *File) {
	for bi := range file.Blocks {
		block := &file.Blocks[bi]
		for ii := range block.Interfaces {
			iface := &block.Interfaces[ii]
			for _, fd := range iface.Fields {
				// Getter: func T.name(self) -> Type
				getter := FuncDecl{
					Name:         fd.Name,
					ReceiverType: fd.TypeParam,
					Params: []Param{
						{Name: "self", IsSelf: true},
					},
					ReturnType: &fd.Type,
					Span:       fd.Span,
				}
				iface.Methods = append(iface.Methods, getter)

				// Setter: func T.set_name(mut self, val: Type)
				setter := FuncDecl{
					Name:         "set_" + fd.Name,
					ReceiverType: fd.TypeParam,
					Params: []Param{
						{Name: "self", IsSelf: true, IsMut: true},
						{Name: "val", Type: fd.Type},
					},
					Span: fd.Span,
				}
				iface.Methods = append(iface.Methods, setter)
			}
		}
	}
}

// DesugarRelations processes relation declarations:
// 1. Injects default fields from the interface into concrete classes (with label prefixing)
// 2. Generates impl blocks with field bindings mapping interface getters to concrete fields
func DesugarRelations(file *File) {
	// Build global interface lookup across ALL blocks (interfaces may be in stdlib block 0
	// while relations referencing them are in user blocks)
	globalIfaceMap := make(map[string]*InterfaceDecl)
	for bi := range file.Blocks {
		for ii := range file.Blocks[bi].Interfaces {
			globalIfaceMap[file.Blocks[bi].Interfaces[ii].Name] = &file.Blocks[bi].Interfaces[ii]
		}
	}

	for bi := range file.Blocks {
		block := &file.Blocks[bi]

		// Build class lookup: name -> index in block.Classes
		classIdx := make(map[string]int)
		for ci := range block.Classes {
			classIdx[block.Classes[ci].Name] = ci
		}

		for _, rel := range block.Relations {
			iface := globalIfaceMap[rel.Hint]
			if iface == nil || len(iface.Fields) == 0 {
				continue
			}

			if len(iface.TypeParams) < 2 {
				continue
			}

			// Map interface type params to concrete types from the relation
			typeMap := make(map[string]RelationSide) // type param name -> relation side
			typeMap[iface.TypeParams[0].Name] = rel.Parent
			typeMap[iface.TypeParams[1].Name] = rel.Child

			// Collect impl mappings for the auto-generated impl block
			var mappings []ImplMapping

			// For each interface field, inject into the appropriate concrete class
			for _, fd := range iface.Fields {
				side, ok := typeMap[fd.TypeParam]
				if !ok {
					continue
				}
				ci, ok := classIdx[side.TypeName]
				if !ok {
					continue
				}

				// Build field name with label prefix
				fieldName := fd.Name
				if side.Label != "" {
					fieldName = side.Label + "_" + fd.Name
				}

				// Rewrite type: replace type param references with concrete types
				fieldType := rewriteFieldType(fd.Type, iface.TypeParams, rel)

				block.Classes[ci].Fields = append(block.Classes[ci].Fields, Field{
					Name: fieldName,
					Type: fieldType,
					Span: fd.Span,
				})

				// Generate field binding for getter: T.name <-> ConcreteClass.prefixed_name
				mappings = append(mappings, ImplMapping{
					TypeParam:    fd.TypeParam,
					MethodName:   fd.Name,
					Kind:         ImplFieldBind,
					TargetClass:  side.TypeName,
					TargetMember: fieldName,
					Span:         fd.Span,
				})

				// Generate field binding for setter: T.set_name <-> ConcreteClass.prefixed_name
				mappings = append(mappings, ImplMapping{
					TypeParam:    fd.TypeParam,
					MethodName:   "set_" + fd.Name,
					Kind:         ImplFieldBind,
					TargetClass:  side.TypeName,
					TargetMember: fieldName,
					Span:         fd.Span,
				})
			}

			// Merge into existing user impl block if present, otherwise create new one
			if len(mappings) > 0 {
				parentName := rel.Parent.TypeName
				childName := rel.Child.TypeName

				// Look for existing impl block for same interface+types
				var existingImpl *ImplBlock
				for ii := range block.ImplBlocks {
					ib := &block.ImplBlocks[ii]
					if ib.InterfaceName == rel.Hint && len(ib.TypeArgs) >= 2 {
						ta0, _ := ib.TypeArgs[0].Data.(NamedType)
						ta1, _ := ib.TypeArgs[1].Data.(NamedType)
						if ta0.Name == parentName && ta1.Name == childName {
							existingImpl = ib
							break
						}
					}
				}

				if existingImpl != nil {
					// Merge: only add mappings not already present
					existing := make(map[string]bool)
					for _, m := range existingImpl.Mappings {
						existing[m.TypeParam+"."+m.MethodName] = true
					}
					for _, m := range mappings {
						key := m.TypeParam + "." + m.MethodName
						if !existing[key] {
							existingImpl.Mappings = append(existingImpl.Mappings, m)
						}
					}
				} else {
					// Build TypeArgs with type parameters from the relation sides
					buildTypeExpr := func(side RelationSide) TypeExpr {
						nt := NamedType{Name: side.TypeName}
						for _, ta := range side.TypeArgs {
							nt.Args = append(nt.Args, TypeExpr{
								Kind: TypeNamed,
								Data: NamedType{Name: ta},
							})
						}
						return TypeExpr{Kind: TypeNamed, Data: nt}
					}
					var typeArgs []TypeExpr
					typeArgs = append(typeArgs, buildTypeExpr(rel.Parent))
					typeArgs = append(typeArgs, buildTypeExpr(rel.Child))

					block.ImplBlocks = append(block.ImplBlocks, ImplBlock{
						InterfaceName: rel.Hint,
						TypeArgs:      typeArgs,
						Mappings:      mappings,
						Span:          rel.Span,
					})
				}
			}
		}
	}
}

// deepCopyBlock creates a deep copy of a Block by JSON-like recursive copying.
// deepCopyBlock creates a true recursive deep copy of a Block.
// This is critical for DesugarDestructors: each relation gets its own copy of
// the interface destructor body. Without deep copying, Stmt.Data pointers are
// shared and mutations (rename, type substitution) bleed across relations.
func deepCopyBlock(b Block) Block {
	stmts := make([]Stmt, len(b.Stmts))
	for i := range b.Stmts {
		stmts[i] = deepCopyStmt(b.Stmts[i])
	}
	return Block{Stmts: stmts}
}

func deepCopyStmt(s Stmt) Stmt {
	out := Stmt{Kind: s.Kind, Span: s.Span}
	switch d := s.Data.(type) {
	case *ExprStmt:
		c := *d
		c.Expr = deepCopyExpr(d.Expr)
		out.Data = &c
	case *VarDeclStmt:
		c := *d
		if d.Value != nil {
			v := deepCopyExpr(*d.Value)
			c.Value = &v
		}
		if d.Type != nil {
			t := deepCopyTypeExpr(*d.Type)
			c.Type = &t
		}
		out.Data = &c
	case *AssignStmt:
		c := *d
		c.Target = deepCopyExpr(d.Target)
		c.Value = deepCopyExpr(d.Value)
		out.Data = &c
	case *ReturnStmt:
		c := *d
		if d.Value != nil {
			v := deepCopyExpr(*d.Value)
			c.Value = &v
		}
		out.Data = &c
	case *IfStmt:
		c := *d
		c.Condition = deepCopyExpr(d.Condition)
		c.Then = deepCopyBlock(d.Then)
		if d.Else != nil {
			e := deepCopyBlock(*d.Else)
			c.Else = &e
		}
		elseIfs := make([]ElseIf, len(d.ElseIfs))
		for i, ei := range d.ElseIfs {
			elseIfs[i] = ElseIf{
				Condition: deepCopyExpr(ei.Condition),
				Body:      deepCopyBlock(ei.Body),
			}
		}
		c.ElseIfs = elseIfs
		out.Data = &c
	case *WhileStmt:
		c := *d
		c.Condition = deepCopyExpr(d.Condition)
		c.Body = deepCopyBlock(d.Body)
		out.Data = &c
	case *ForStmt:
		c := *d
		c.Collection = deepCopyExpr(d.Collection)
		c.Body = deepCopyBlock(d.Body)
		out.Data = &c
	case *MatchStmt:
		c := *d
		c.Value = deepCopyExpr(d.Value)
		arms := make([]MatchArm, len(d.Arms))
		for i, a := range d.Arms {
			arms[i] = a
			arms[i].Body = deepCopyBlock(a.Body)
		}
		c.Arms = arms
		out.Data = &c
	case *Block:
		b := deepCopyBlock(*d)
		out.Data = &b
	default:
		// For statement types that don't contain Exprs (e.g. BreakStmt, ContinueStmt),
		// the shallow copy is fine.
		out.Data = s.Data
	}
	return out
}

func deepCopyExpr(e Expr) Expr {
	out := Expr{Kind: e.Kind, Span: e.Span, ResolvedType: e.ResolvedType}
	switch d := e.Data.(type) {
	case *MethodCallExpr:
		c := *d
		c.Receiver = deepCopyExpr(d.Receiver)
		args := make([]Expr, len(d.Args))
		for i := range d.Args {
			args[i] = deepCopyExpr(d.Args[i])
		}
		c.Args = args
		out.Data = &c
	case MethodCallExpr:
		c := d
		c.Receiver = deepCopyExpr(d.Receiver)
		args := make([]Expr, len(d.Args))
		for i := range d.Args {
			args[i] = deepCopyExpr(d.Args[i])
		}
		c.Args = args
		out.Data = &c
	case *CallExpr:
		c := *d
		c.Func = deepCopyExpr(d.Func)
		args := make([]Expr, len(d.Args))
		for i := range d.Args {
			args[i] = deepCopyExpr(d.Args[i])
		}
		c.Args = args
		out.Data = &c
	case CallExpr:
		c := d
		c.Func = deepCopyExpr(d.Func)
		args := make([]Expr, len(d.Args))
		for i := range d.Args {
			args[i] = deepCopyExpr(d.Args[i])
		}
		c.Args = args
		out.Data = &c
	case UnaryExpr:
		c := d
		c.Operand = deepCopyExpr(d.Operand)
		out.Data = c
	case BinaryExpr:
		c := d
		c.Left = deepCopyExpr(d.Left)
		c.Right = deepCopyExpr(d.Right)
		out.Data = c
	case FieldAccessExpr:
		c := d
		c.Receiver = deepCopyExpr(d.Receiver)
		out.Data = c
	case IndexExpr:
		c := d
		c.Receiver = deepCopyExpr(d.Receiver)
		c.Index = deepCopyExpr(d.Index)
		out.Data = c
	default:
		// Literals, identifiers, etc. — no nested Exprs to copy
		out.Data = e.Data
	}
	return out
}

func deepCopyTypeExpr(te TypeExpr) TypeExpr {
	out := TypeExpr{Kind: te.Kind, Span: te.Span}
	switch d := te.Data.(type) {
	case NamedType:
		c := d
		if len(d.Args) > 0 {
			args := make([]TypeExpr, len(d.Args))
			for i := range d.Args {
				args[i] = deepCopyTypeExpr(d.Args[i])
			}
			c.Args = args
		}
		out.Data = c
	case *TypeExpr:
		inner := deepCopyTypeExpr(*d)
		out.Data = &inner
	default:
		out.Data = te.Data
	}
	return out
}

// substituteTypeParamsInBlock rewrites type parameter references in a block's statements.
// Used by DesugarDestructors to replace interface type params (e.g. P, C) with concrete class names.
func substituteTypeParamsInBlock(block *Block, typeMap map[string]string) {
	for i := range block.Stmts {
		substituteTypeParamsInStmt(&block.Stmts[i], typeMap)
	}
}

func substituteTypeParamsInStmt(stmt *Stmt, typeMap map[string]string) {
	switch stmt.Kind {
	case StmtExpr:
		if d, ok := stmt.Data.(*ExprStmt); ok {
			substituteTypeParamsInExpr(&d.Expr, typeMap)
		}
	case StmtAssign:
		if d, ok := stmt.Data.(*AssignStmt); ok {
			substituteTypeParamsInExpr(&d.Value, typeMap)
		} else if d, ok := stmt.Data.(AssignStmt); ok {
			substituteTypeParamsInExpr(&d.Value, typeMap)
			stmt.Data = d
		}
	case StmtVarDecl:
		if d, ok := stmt.Data.(*VarDeclStmt); ok {
			if d.Value != nil {
				substituteTypeParamsInExpr(d.Value, typeMap)
			}
			if d.ElseBlock != nil {
				substituteTypeParamsInBlock(d.ElseBlock, typeMap)
			}
		}
	case StmtReturn:
		if d, ok := stmt.Data.(*ReturnStmt); ok {
			if d.Value != nil {
				substituteTypeParamsInExpr(d.Value, typeMap)
			}
		}
	case StmtIf:
		if d, ok := stmt.Data.(*IfStmt); ok {
			if d.LetValue != nil {
				substituteTypeParamsInExpr(d.LetValue, typeMap)
			} else {
				substituteTypeParamsInExpr(&d.Condition, typeMap)
			}
			substituteTypeParamsInBlock(&d.Then, typeMap)
			for ei := range d.ElseIfs {
				substituteTypeParamsInExpr(&d.ElseIfs[ei].Condition, typeMap)
				substituteTypeParamsInBlock(&d.ElseIfs[ei].Body, typeMap)
			}
			if d.Else != nil {
				substituteTypeParamsInBlock(d.Else, typeMap)
			}
		}
	case StmtWhile:
		if d, ok := stmt.Data.(*WhileStmt); ok {
			substituteTypeParamsInExpr(&d.Condition, typeMap)
			substituteTypeParamsInBlock(&d.Body, typeMap)
		}
	case StmtFor:
		if d, ok := stmt.Data.(*ForStmt); ok {
			substituteTypeParamsInExpr(&d.Collection, typeMap)
			substituteTypeParamsInBlock(&d.Body, typeMap)
		}
	}
}

func substituteTypeParamsInExpr(expr *Expr, typeMap map[string]string) {
	if expr == nil {
		return
	}
	switch d := expr.Data.(type) {
	case *CallExpr:
		for i := range d.TypeArgs {
			substituteTypeParamsInTypeExpr(&d.TypeArgs[i], typeMap)
		}
		substituteTypeParamsInExpr(&d.Func, typeMap)
		for i := range d.Args {
			substituteTypeParamsInExpr(&d.Args[i], typeMap)
		}
	case CallExpr:
		for i := range d.TypeArgs {
			substituteTypeParamsInTypeExpr(&d.TypeArgs[i], typeMap)
		}
		substituteTypeParamsInExpr(&d.Func, typeMap)
		for i := range d.Args {
			substituteTypeParamsInExpr(&d.Args[i], typeMap)
		}
		expr.Data = d
	case *MethodCallExpr:
		for i := range d.TypeArgs {
			substituteTypeParamsInTypeExpr(&d.TypeArgs[i], typeMap)
		}
		substituteTypeParamsInExpr(&d.Receiver, typeMap)
		for i := range d.Args {
			substituteTypeParamsInExpr(&d.Args[i], typeMap)
		}
	case MethodCallExpr:
		for i := range d.TypeArgs {
			substituteTypeParamsInTypeExpr(&d.TypeArgs[i], typeMap)
		}
		substituteTypeParamsInExpr(&d.Receiver, typeMap)
		for i := range d.Args {
			substituteTypeParamsInExpr(&d.Args[i], typeMap)
		}
		expr.Data = d
	case UnaryExpr:
		substituteTypeParamsInExpr(&d.Operand, typeMap)
		expr.Data = d
	case BinaryExpr:
		substituteTypeParamsInExpr(&d.Left, typeMap)
		substituteTypeParamsInExpr(&d.Right, typeMap)
		expr.Data = d
	case FieldAccessExpr:
		substituteTypeParamsInExpr(&d.Receiver, typeMap)
		expr.Data = d
	case IndexExpr:
		substituteTypeParamsInExpr(&d.Receiver, typeMap)
		substituteTypeParamsInExpr(&d.Index, typeMap)
		expr.Data = d
	}
}

func substituteTypeParamsInTypeExpr(te *TypeExpr, typeMap map[string]string) {
	if te == nil || te.Data == nil {
		return
	}
	switch te.Kind {
	case TypeNamed:
		nt := te.Data.(NamedType)
		if replacement, ok := typeMap[nt.Name]; ok {
			te.Data = NamedType{Name: replacement, Args: nt.Args}
		}
		for i := range nt.Args {
			substituteTypeParamsInTypeExpr(&nt.Args[i], typeMap)
		}
	case TypeOptional:
		if inner, ok := te.Data.(*TypeExpr); ok {
			substituteTypeParamsInTypeExpr(inner, typeMap)
		}
	case TypeSequence:
		if inner, ok := te.Data.(*TypeExpr); ok {
			substituteTypeParamsInTypeExpr(inner, typeMap)
		}
	}
}

// substituteTypeParamsRich* variants use map[string]TypeExpr for rich substitution.
// When replacing type param P with Dict<V>, the replacement carries Args so that
// hash_remove<P, C>() becomes hash_remove<Dict<V>, DictEntry<V>>() not hash_remove<Dict, DictEntry>().

func substituteTypeParamsRichInBlock(block *Block, typeMap map[string]TypeExpr) {
	for i := range block.Stmts {
		substituteTypeParamsRichInStmt(&block.Stmts[i], typeMap)
	}
}

func substituteTypeParamsRichInStmt(stmt *Stmt, typeMap map[string]TypeExpr) {
	switch stmt.Kind {
	case StmtExpr:
		if d, ok := stmt.Data.(*ExprStmt); ok {
			substituteTypeParamsRichInExpr(&d.Expr, typeMap)
		}
	case StmtAssign:
		if d, ok := stmt.Data.(*AssignStmt); ok {
			substituteTypeParamsRichInExpr(&d.Value, typeMap)
		} else if d, ok := stmt.Data.(AssignStmt); ok {
			substituteTypeParamsRichInExpr(&d.Value, typeMap)
			stmt.Data = d
		}
	case StmtVarDecl:
		if d, ok := stmt.Data.(*VarDeclStmt); ok {
			if d.Value != nil {
				substituteTypeParamsRichInExpr(d.Value, typeMap)
			}
			if d.ElseBlock != nil {
				substituteTypeParamsRichInBlock(d.ElseBlock, typeMap)
			}
		}
	case StmtReturn:
		if d, ok := stmt.Data.(*ReturnStmt); ok {
			if d.Value != nil {
				substituteTypeParamsRichInExpr(d.Value, typeMap)
			}
		}
	case StmtIf:
		if d, ok := stmt.Data.(*IfStmt); ok {
			if d.LetValue != nil {
				substituteTypeParamsRichInExpr(d.LetValue, typeMap)
			} else {
				substituteTypeParamsRichInExpr(&d.Condition, typeMap)
			}
			substituteTypeParamsRichInBlock(&d.Then, typeMap)
			for ei := range d.ElseIfs {
				substituteTypeParamsRichInExpr(&d.ElseIfs[ei].Condition, typeMap)
				substituteTypeParamsRichInBlock(&d.ElseIfs[ei].Body, typeMap)
			}
			if d.Else != nil {
				substituteTypeParamsRichInBlock(d.Else, typeMap)
			}
		}
	case StmtWhile:
		if d, ok := stmt.Data.(*WhileStmt); ok {
			substituteTypeParamsRichInExpr(&d.Condition, typeMap)
			substituteTypeParamsRichInBlock(&d.Body, typeMap)
		}
	case StmtFor:
		if d, ok := stmt.Data.(*ForStmt); ok {
			substituteTypeParamsRichInExpr(&d.Collection, typeMap)
			substituteTypeParamsRichInBlock(&d.Body, typeMap)
		}
	case StmtBlock:
		if d, ok := stmt.Data.(*Block); ok {
			substituteTypeParamsRichInBlock(d, typeMap)
		}
	}
}

func substituteTypeParamsRichInExpr(expr *Expr, typeMap map[string]TypeExpr) {
	if expr == nil {
		return
	}
	switch d := expr.Data.(type) {
	case *CallExpr:
		for i := range d.TypeArgs {
			substituteTypeParamsRichInTypeExpr(&d.TypeArgs[i], typeMap)
		}
		substituteTypeParamsRichInExpr(&d.Func, typeMap)
		for i := range d.Args {
			substituteTypeParamsRichInExpr(&d.Args[i], typeMap)
		}
	case CallExpr:
		for i := range d.TypeArgs {
			substituteTypeParamsRichInTypeExpr(&d.TypeArgs[i], typeMap)
		}
		substituteTypeParamsRichInExpr(&d.Func, typeMap)
		for i := range d.Args {
			substituteTypeParamsRichInExpr(&d.Args[i], typeMap)
		}
		expr.Data = d
	case *MethodCallExpr:
		for i := range d.TypeArgs {
			substituteTypeParamsRichInTypeExpr(&d.TypeArgs[i], typeMap)
		}
		substituteTypeParamsRichInExpr(&d.Receiver, typeMap)
		for i := range d.Args {
			substituteTypeParamsRichInExpr(&d.Args[i], typeMap)
		}
	case MethodCallExpr:
		for i := range d.TypeArgs {
			substituteTypeParamsRichInTypeExpr(&d.TypeArgs[i], typeMap)
		}
		substituteTypeParamsRichInExpr(&d.Receiver, typeMap)
		for i := range d.Args {
			substituteTypeParamsRichInExpr(&d.Args[i], typeMap)
		}
		expr.Data = d
	case UnaryExpr:
		substituteTypeParamsRichInExpr(&d.Operand, typeMap)
		expr.Data = d
	case BinaryExpr:
		substituteTypeParamsRichInExpr(&d.Left, typeMap)
		substituteTypeParamsRichInExpr(&d.Right, typeMap)
		expr.Data = d
	case FieldAccessExpr:
		substituteTypeParamsRichInExpr(&d.Receiver, typeMap)
		expr.Data = d
	case IndexExpr:
		substituteTypeParamsRichInExpr(&d.Receiver, typeMap)
		substituteTypeParamsRichInExpr(&d.Index, typeMap)
		expr.Data = d
	}
}

func substituteTypeParamsRichInTypeExpr(te *TypeExpr, typeMap map[string]TypeExpr) {
	if te == nil || te.Data == nil {
		return
	}
	switch te.Kind {
	case TypeNamed:
		nt := te.Data.(NamedType)
		if replacement, ok := typeMap[nt.Name]; ok {
			// Replace entirely with the rich TypeExpr (which carries Args)
			*te = replacement
			return
		}
		for i := range nt.Args {
			substituteTypeParamsRichInTypeExpr(&nt.Args[i], typeMap)
		}
	case TypeOptional:
		if inner, ok := te.Data.(*TypeExpr); ok {
			substituteTypeParamsRichInTypeExpr(inner, typeMap)
		}
	case TypeSequence:
		if inner, ok := te.Data.(*TypeExpr); ok {
			substituteTypeParamsRichInTypeExpr(inner, typeMap)
		}
	}
}

// DesugarDestructors generates destroy methods on classes involved in relations.
// For each relation with destructors, copies the interface's destructor blocks to the
// concrete classes as `destroy` methods. Must run after DesugarInterfaceFields and DesugarRelations.
func DesugarDestructors(file *File) {
	for bi := range file.Blocks {
		block := &file.Blocks[bi]

		// Build interface lookup
		ifaceMap := make(map[string]*InterfaceDecl)
		for ii := range block.Interfaces {
			ifaceMap[block.Interfaces[ii].Name] = &block.Interfaces[ii]
		}

		// Build class lookup
		classIdx := make(map[string]int)
		for ci := range block.Classes {
			classIdx[block.Classes[ci].Name] = ci
		}

		// Collect destructor bodies per class (multiple relations can append)
		// className -> []Block
		destructorBodies := make(map[string][]Block)

		for _, rel := range block.Relations {
			iface := ifaceMap[rel.Hint]
			if iface == nil || len(iface.Destructors) == 0 {
				continue
			}
			if len(iface.TypeParams) < 2 {
				continue
			}

			// Map type params to concrete class names (simple, for class-name lookups)
			typeParamToClass := make(map[string]string)
			typeParamToClass[iface.TypeParams[0].Name] = rel.Parent.TypeName
			typeParamToClass[iface.TypeParams[1].Name] = rel.Child.TypeName

			// Rich map for type substitution: includes TypeArgs (e.g., P → Dict<V>)
			typeParamToTypeExpr := make(map[string]TypeExpr)
			buildRichTypeExpr := func(side RelationSide) TypeExpr {
				nt := NamedType{Name: side.TypeName}
				for _, ta := range side.TypeArgs {
					nt.Args = append(nt.Args, TypeExpr{
						Kind: TypeNamed,
						Data: NamedType{Name: ta},
					})
				}
				return TypeExpr{Kind: TypeNamed, Data: nt}
			}
			typeParamToTypeExpr[iface.TypeParams[0].Name] = buildRichTypeExpr(rel.Parent)
			typeParamToTypeExpr[iface.TypeParams[1].Name] = buildRichTypeExpr(rel.Child)

			// Build method rename map for label-prefixed fields.
			// Interface fields like "children" become "fb_children" when the label is "fb".
			methodRenames := make(map[string]string)
			typeParamToLabel := make(map[string]string)
			typeParamToLabel[iface.TypeParams[0].Name] = rel.Parent.Label
			typeParamToLabel[iface.TypeParams[1].Name] = rel.Child.Label
			for _, fd := range iface.Fields {
				label := typeParamToLabel[fd.TypeParam]
				if label != "" {
					methodRenames[fd.Name] = label + "_" + fd.Name
					methodRenames["set_"+fd.Name] = "set_" + label + "_" + fd.Name
				}
			}

			for _, db := range iface.Destructors {
				className, ok := typeParamToClass[db.TypeParam]
				if !ok {
					continue
				}
				if _, ok := classIdx[className]; !ok {
					continue
				}
				// Deep copy and substitute type params in the body
				bodyCopy := deepCopyBlock(db.Body)
				substituteTypeParamsRichInBlock(&bodyCopy, typeParamToTypeExpr)
				// Rename generic interface method calls to label-prefixed versions
				if len(methodRenames) > 0 {
					renameMethodCallsInBlock(&bodyCopy, methodRenames)
				}
				destructorBodies[className] = append(destructorBodies[className], bodyCopy)
			}
		}

		// Generate destroy methods
		for className, bodies := range destructorBodies {
			var allStmts []Stmt
			for _, body := range bodies {
				// Wrap each destructor body in a block to avoid variable name collisions
				b := body // copy for addressability
				allStmts = append(allStmts, Stmt{
					Kind: StmtBlock,
					Data: &b,
				})
			}

			destroyMethod := FuncDecl{
				Name:         "destroy",
				ReceiverType: "", // method on the class itself
				IsPublic:     true,
				Params: []Param{
					{Name: "self", IsSelf: true, IsMut: true},
				},
				Body: &Block{Stmts: allStmts},
			}

			// Add as a method on the class
			ci := classIdx[className]
			block.Classes[ci].Methods = append(block.Classes[ci].Methods, destroyMethod)
		}
	}
}

// renameMethodCallsInBlock renames method calls in a block using the provided map.
// Used by DesugarDestructors to rewrite generic interface method names (e.g. "children")
// to label-prefixed concrete names (e.g. "fb_children") so the checker can resolve them
// on the correct concrete class.
func renameMethodCallsInBlock(block *Block, renames map[string]string) {
	for i := range block.Stmts {
		renameMethodCallsInStmt(&block.Stmts[i], renames)
	}
}

func renameMethodCallsInStmt(stmt *Stmt, renames map[string]string) {
	switch d := stmt.Data.(type) {
	case *ExprStmt:
		renameMethodCallsInExpr(&d.Expr, renames)
	case *AssignStmt:
		renameMethodCallsInExpr(&d.Target, renames)
		renameMethodCallsInExpr(&d.Value, renames)
	case *VarDeclStmt:
		if d.Value != nil {
			renameMethodCallsInExpr(d.Value, renames)
		}
	case *IfStmt:
		renameMethodCallsInExpr(&d.Condition, renames)
		renameMethodCallsInBlock(&d.Then, renames)
		for ei := range d.ElseIfs {
			renameMethodCallsInExpr(&d.ElseIfs[ei].Condition, renames)
			renameMethodCallsInBlock(&d.ElseIfs[ei].Body, renames)
		}
		if d.Else != nil {
			renameMethodCallsInBlock(d.Else, renames)
		}
	case *WhileStmt:
		renameMethodCallsInExpr(&d.Condition, renames)
		renameMethodCallsInBlock(&d.Body, renames)
	case *ForStmt:
		renameMethodCallsInExpr(&d.Collection, renames)
		renameMethodCallsInBlock(&d.Body, renames)
	case *MatchStmt:
		renameMethodCallsInExpr(&d.Value, renames)
		for ai := range d.Arms {
			renameMethodCallsInBlock(&d.Arms[ai].Body, renames)
		}
	case *Block:
		renameMethodCallsInBlock(d, renames)
	case *ReturnStmt:
		if d.Value != nil {
			renameMethodCallsInExpr(d.Value, renames)
		}
	}
}

func renameMethodCallsInExpr(expr *Expr, renames map[string]string) {
	if expr == nil {
		return
	}
	switch d := expr.Data.(type) {
	case *MethodCallExpr:
		if newName, ok := renames[d.Method]; ok {
			// Clone the MethodCallExpr to avoid mutating the original (shared via shallow deepCopyBlock)
			clone := *d
			clone.Method = newName
			expr.Data = &clone
			d = &clone
		}
		renameMethodCallsInExpr(&d.Receiver, renames)
		for i := range d.Args {
			renameMethodCallsInExpr(&d.Args[i], renames)
		}
	case *CallExpr:
		renameMethodCallsInExpr(&d.Func, renames)
		for i := range d.Args {
			renameMethodCallsInExpr(&d.Args[i], renames)
		}
	case *BinaryExpr:
		renameMethodCallsInExpr(&d.Left, renames)
		renameMethodCallsInExpr(&d.Right, renames)
	case *UnaryExpr:
		renameMethodCallsInExpr(&d.Operand, renames)
	case *FieldAccessExpr:
		renameMethodCallsInExpr(&d.Receiver, renames)
	case *IndexExpr:
		renameMethodCallsInExpr(&d.Receiver, renames)
		renameMethodCallsInExpr(&d.Index, renames)
	case *UnwrapExpr:
		renameMethodCallsInExpr(&d.Operand, renames)
	}
}

// concrete class names from the relation.
func rewriteFieldType(te TypeExpr, typeParams []TypeParam, rel RelationDecl) TypeExpr {
	switch te.Kind {
	case TypeNamed:
		nt := te.Data.(NamedType)
		if len(typeParams) >= 1 && nt.Name == typeParams[0].Name {
			args := make([]TypeExpr, len(rel.Parent.TypeArgs))
			for i, arg := range rel.Parent.TypeArgs {
				args[i] = TypeExpr{Kind: TypeNamed, Data: NamedType{Name: arg}, Span: te.Span}
			}
			return TypeExpr{Kind: TypeNamed, Data: NamedType{Name: rel.Parent.TypeName, Args: args}, Span: te.Span}
		}
		if len(typeParams) >= 2 && nt.Name == typeParams[1].Name {
			args := make([]TypeExpr, len(rel.Child.TypeArgs))
			for i, arg := range rel.Child.TypeArgs {
				args[i] = TypeExpr{Kind: TypeNamed, Data: NamedType{Name: arg}, Span: te.Span}
			}
			return TypeExpr{Kind: TypeNamed, Data: NamedType{Name: rel.Child.TypeName, Args: args}, Span: te.Span}
		}
		return te
	case TypeOptional:
		ot := te.Data.(OptionalType)
		inner := rewriteFieldType(ot.Inner, typeParams, rel)
		return TypeExpr{Kind: TypeOptional, Data: OptionalType{Inner: inner}, Span: te.Span}
	case TypeSequence:
		st := te.Data.(SequenceType)
		elem := rewriteFieldType(st.Elem, typeParams, rel)
		return TypeExpr{Kind: TypeSequence, Data: SequenceType{Elem: elem}, Span: te.Span}
	default:
		return te
	}
}
