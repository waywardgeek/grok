package ast

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

// DesugarRelations processes relation declarations:
// 1. Injects default fields from the interface into concrete classes (with label prefixing)
// 2. Generates impl blocks mapping interface fields to concrete class fields
func DesugarRelations(file *File) {
	for bi := range file.Blocks {
		block := &file.Blocks[bi]

		// Build interface lookup: name -> InterfaceDecl
		ifaceMap := make(map[string]*InterfaceDecl)
		for ii := range block.Interfaces {
			ifaceMap[block.Interfaces[ii].Name] = &block.Interfaces[ii]
		}

		// Build class lookup: name -> index in block.Classes
		classIdx := make(map[string]int)
		for ci := range block.Classes {
			classIdx[block.Classes[ci].Name] = ci
		}

		for _, rel := range block.Relations {
			iface := ifaceMap[rel.Hint]
			if iface == nil || len(iface.Fields) == 0 {
				continue
			}

			// Build type param -> concrete type mapping from the relation
			// Interface has type params like <P, C>
			// Relation maps: Parent -> first type param, Child -> second (if IsMany/exists)
			if len(iface.TypeParams) < 2 {
				continue
			}

			// Map interface type params to concrete types from the relation
			typeMap := make(map[string]RelationSide) // type param name -> relation side
			typeMap[iface.TypeParams[0].Name] = rel.Parent
			typeMap[iface.TypeParams[1].Name] = rel.Child

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
			}
		}
	}
}

// rewriteFieldType replaces type parameter references in a field type with
// concrete class names from the relation.
func rewriteFieldType(te TypeExpr, typeParams []TypeParam, rel RelationDecl) TypeExpr {
	switch te.Kind {
	case TypeNamed:
		nt := te.Data.(NamedType)
		if len(typeParams) >= 1 && nt.Name == typeParams[0].Name {
			return TypeExpr{Kind: TypeNamed, Data: NamedType{Name: rel.Parent.TypeName}, Span: te.Span}
		}
		if len(typeParams) >= 2 && nt.Name == typeParams[1].Name {
			return TypeExpr{Kind: TypeNamed, Data: NamedType{Name: rel.Child.TypeName}, Span: te.Span}
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
