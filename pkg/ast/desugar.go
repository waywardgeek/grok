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
