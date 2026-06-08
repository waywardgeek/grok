package lir

import (
	"fmt"

	"github.com/waywardgeek/forge/pkg/ast"
)

// ValidatePostLower checks LIR invariants after lowering. Returns violations.
func ValidatePostLower(prog *LProgram) []ast.InvariantViolation {
	var violations []ast.InvariantViolation

	for _, f := range prog.Functions {
		// Check params for void*
		for _, p := range f.Params {
			if p.Type != nil && p.Type.Kind == LTyAny {
				violations = append(violations, ast.InvariantViolation{
					Stage:   "post-lower",
					Check:   "no-void-star",
					Message: fmt.Sprintf("%s param %q has type void* (LTyAny)", f.Name, p.Name),
				})
			}
		}
		// Check return type
		if f.ReturnType != nil && f.ReturnType.Kind == LTyAny {
			violations = append(violations, ast.InvariantViolation{
				Stage:   "post-lower",
				Check:   "no-void-star",
				Message: fmt.Sprintf("%s return type is void* (LTyAny)", f.Name),
			})
		}
		// Check temps/vars
		collectVoidStarViolations(f.Name, f.Body, &violations)
	}

	return violations
}

func collectVoidStarViolations(funcName string, stmts []LStmt, violations *[]ast.InvariantViolation) {
	for _, stmt := range stmts {
		switch stmt.Kind {
		case LStmtTempDef:
			td := stmt.Data.(*LTempDef)
			if td.Expr.Type != nil && td.Expr.Type.Kind == LTyAny {
				detail := lExprKindName(td.Expr.Kind)
				switch td.Expr.Kind {
				case LExprCall:
					if d := dataAs[LCallData](td.Expr.Data); d != nil {
						detail = fmt.Sprintf("Call(%s)", d.Func)
					}
				case LExprMethodCall:
					if d := dataAs[LMethodCallData](td.Expr.Data); d != nil {
						detail = fmt.Sprintf("MethodCall(.%s)", d.Method)
					}
				case LExprStructField:
					if d := dataAs[LStructFieldData](td.Expr.Data); d != nil {
						detail = fmt.Sprintf("StructField(.%s)", d.Field)
					}
				case LExprClassGet:
					if d := dataAs[LClassGetData](td.Expr.Data); d != nil {
						detail = fmt.Sprintf("ClassGet(%s.%s)", d.Class, d.Field)
					}
				}
				*violations = append(*violations, ast.InvariantViolation{
					Stage:   "post-lower",
					Check:   "no-void-star",
					Message: fmt.Sprintf("%s _t%d → void* via %s", funcName, td.ID, detail),
				})
			}
		case LStmtVarDecl:
			vd := stmt.Data.(*LVarDecl)
			if vd.Type != nil && vd.Type.Kind == LTyAny {
				*violations = append(*violations, ast.InvariantViolation{
					Stage:   "post-lower",
					Check:   "no-void-star",
					Message: fmt.Sprintf("%s var %q has type void* (LTyAny)", funcName, vd.Name),
				})
			}
		case LStmtBlock:
			b := stmt.Data.(*LBlock)
			collectVoidStarViolations(funcName, b.Stmts, violations)
		case LStmtIf:
			ifData := stmt.Data.(*LIf)
			collectVoidStarViolations(funcName, ifData.Then, violations)
			collectVoidStarViolations(funcName, ifData.Else, violations)
		case LStmtSwitch:
			sw := stmt.Data.(*LSwitch)
			for _, c := range sw.Cases {
				collectVoidStarViolations(funcName, c.Body, violations)
			}
		case LStmtFor:
			forData := stmt.Data.(*LFor)
			collectVoidStarViolations(funcName, forData.Body, violations)
		case LStmtWhile:
			whileData := stmt.Data.(*LWhile)
			collectVoidStarViolations(funcName, whileData.CondBlock, violations)
			collectVoidStarViolations(funcName, whileData.Body, violations)
		case LStmtTypeSwitch:
			ts := stmt.Data.(*LTypeSwitch)
			for _, c := range ts.Cases {
				collectVoidStarViolations(funcName, c.Body, violations)
			}
		}
	}
}
