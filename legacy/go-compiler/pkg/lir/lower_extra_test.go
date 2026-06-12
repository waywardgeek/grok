package lir

import (
	"testing"

	"github.com/waywardgeek/forge/pkg/ast"
	"github.com/waywardgeek/forge/pkg/checker"
)

// ============================================================
// Class lowering
// ============================================================

func TestLowerClassDecl(t *testing.T) {
	l := NewLowerer()
	file := &ast.File{
		Blocks: []ast.ForgeBlock{{
			Name: "test",
			Classes: []ast.ClassDecl{{
				Name: "Node",
				Fields: []ast.Field{
					{Name: "value", Type: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "i32"}}},
					{Name: "label", Type: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "string"}}},
				},
			}},
		}},
	}
	prog := l.Lower(file)
	if len(prog.Classes) != 1 {
		t.Fatalf("expected 1 class, got %d", len(prog.Classes))
	}
	c := prog.Classes[0]
	if c.Name != "Node" {
		t.Errorf("class name = %q, want %q", c.Name, "Node")
	}
	if len(c.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(c.Fields))
	}
	if c.Fields[0].Name != "value" || c.Fields[0].Type.Kind != LTyI32 {
		t.Errorf("field 0: got %q/%d", c.Fields[0].Name, c.Fields[0].Type.Kind)
	}
	if c.Fields[1].Name != "label" || c.Fields[1].Type.Kind != LTyString {
		t.Errorf("field 1: got %q/%d", c.Fields[1].Name, c.Fields[1].Type.Kind)
	}
}

// ============================================================
// Sequence/map/tuple types
// ============================================================

func TestLowerSequenceType(t *testing.T) {
	l := NewLowerer()
	te := &ast.TypeExpr{
		Kind: ast.TypeSequence,
		Data: &ast.SequenceType{
			Elem: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "string"}},
		},
	}
	lt := l.lowerTypeExpr(te)
	if lt.Kind != LTySlice {
		t.Errorf("expected Slice, got %d", lt.Kind)
	}
	if lt.Elem.Kind != LTyString {
		t.Errorf("expected string elem, got %d", lt.Elem.Kind)
	}
}

func TestLowerMapType(t *testing.T) {
	l := NewLowerer()
	te := &ast.TypeExpr{
		Kind: ast.TypeMap,
		Data: &ast.MapType{
			Key:   ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "string"}},
			Value: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "i32"}},
		},
	}
	lt := l.lowerTypeExpr(te)
	if lt.Kind != LTyMap {
		t.Errorf("expected Map, got %d", lt.Kind)
	}
	if lt.Key.Kind != LTyString {
		t.Errorf("expected string key, got %d", lt.Key.Kind)
	}
	if lt.Elem.Kind != LTyI32 {
		t.Errorf("expected i32 value, got %d", lt.Elem.Kind)
	}
}

func TestLowerTupleType(t *testing.T) {
	l := NewLowerer()
	te := &ast.TypeExpr{
		Kind: ast.TypeTuple,
		Data: &ast.TupleType{
			Fields: []ast.TupleField{
				{Type: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "i32"}}},
				{Type: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "bool"}}},
			},
		},
	}
	lt := l.lowerTypeExpr(te)
	if lt.Kind != LTyTuple {
		t.Errorf("expected Tuple, got %d", lt.Kind)
	}
	if len(lt.Fields) != 2 {
		t.Fatalf("expected 2 tuple fields, got %d", len(lt.Fields))
	}
	if lt.Fields[0].Type.Kind != LTyI32 {
		t.Errorf("tuple field 0: expected i32, got %d", lt.Fields[0].Type.Kind)
	}
	if lt.Fields[1].Type.Kind != LTyBool {
		t.Errorf("tuple field 1: expected bool, got %d", lt.Fields[1].Type.Kind)
	}
}

// ============================================================
// Expression lowering
// ============================================================

func TestLowerStringLiteral(t *testing.T) {
	l := NewLowerer()
	l.stmts = nil
	result := l.lowerExpr(&ast.Expr{
		Kind:         ast.ExprStringLit,
		Data:         &ast.StringLitExpr{Value: "hello"},
		ResolvedType: checker.TypeString,
	})
	if result.Kind != LValLitString {
		t.Errorf("expected LitString, got %d", result.Kind)
	}
	if result.StrVal != "hello" {
		t.Errorf("value = %q, want %q", result.StrVal, "hello")
	}
}

func TestLowerIntLiteral(t *testing.T) {
	l := NewLowerer()
	l.stmts = nil
	result := l.lowerExpr(&ast.Expr{
		Kind:         ast.ExprIntLit,
		Data:         &ast.IntLitExpr{Value: "42"},
		ResolvedType: checker.TypeI32,
	})
	if result.Kind != LValLitInt {
		t.Errorf("expected LitInt, got %d", result.Kind)
	}
	if result.IntVal != 42 {
		t.Errorf("value = %d, want 42", result.IntVal)
	}
}

func TestLowerBoolLiteral(t *testing.T) {
	l := NewLowerer()
	l.stmts = nil
	result := l.lowerExpr(&ast.Expr{
		Kind:         ast.ExprBoolLit,
		Data:         &ast.BoolLitExpr{Value: true},
		ResolvedType: checker.TypeBool,
	})
	if result.Kind != LValLitBool {
		t.Errorf("expected LitBool, got %d", result.Kind)
	}
	if !result.BoolVal {
		t.Error("expected true")
	}
}

func TestLowerNegation(t *testing.T) {
	l := NewLowerer()
	l.stmts = nil
	result := l.lowerExpr(&ast.Expr{
		Kind: ast.ExprUnary,
		Data: &ast.UnaryExpr{
			Op:      ast.OpNeg,
			Operand: ast.Expr{Kind: ast.ExprIntLit, Data: &ast.IntLitExpr{Value: "5"}, ResolvedType: checker.TypeI32},
		},
		ResolvedType: checker.TypeI32,
	})
	if len(l.stmts) != 1 {
		t.Fatalf("expected 1 temp, got %d", len(l.stmts))
	}
	if result.Kind != LValTemp {
		t.Errorf("expected temp, got %d", result.Kind)
	}
}

// ============================================================
// Variable declaration
// ============================================================

func TestLowerVarDecl(t *testing.T) {
	l := NewLowerer()
	file := &ast.File{
		Blocks: []ast.ForgeBlock{{
			Name: "test",
			Functions: []ast.FuncDecl{{
				Name: "init",
				Body: &ast.Block{
					Stmts: []ast.Stmt{{
						Kind: ast.StmtVarDecl,
						Data: &ast.VarDeclStmt{
							Name:  "x",
							IsMut: true,
							Type:  &ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "i32"}},
							Value: &ast.Expr{Kind: ast.ExprIntLit, Data: &ast.IntLitExpr{Value: "99"}, ResolvedType: checker.TypeI32},
						},
					}},
				},
			}},
		}},
	}
	prog := l.Lower(file)
	fn := prog.Functions[0]
	if len(fn.Body) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(fn.Body))
	}
	if fn.Body[0].Kind != LStmtVarDecl {
		t.Fatalf("expected VarDecl, got %d", fn.Body[0].Kind)
	}
	vd := fn.Body[0].Data.(*LVarDecl)
	if vd.Name != "x" {
		t.Errorf("name = %q, want %q", vd.Name, "x")
	}
	if !vd.Mutable {
		t.Error("expected mutable")
	}
	if vd.Init == nil {
		t.Fatal("expected init value")
	}
	if vd.Init.Kind != LValLitInt || vd.Init.IntVal != 99 {
		t.Errorf("init = %v, want 99", vd.Init)
	}
}

// ============================================================
// For loop
// ============================================================

func TestLowerForLoop(t *testing.T) {
	l := NewLowerer()
	file := &ast.File{
		Blocks: []ast.ForgeBlock{{
			Name: "test",
			Functions: []ast.FuncDecl{{
				Name: "iter",
				Body: &ast.Block{
					Stmts: []ast.Stmt{{
						Kind: ast.StmtFor,
						Data: &ast.ForStmt{
							Var: "x",
							Collection: ast.Expr{
								Kind: ast.ExprIdent,
								Data: &ast.IdentExpr{Name: "items"},
								ResolvedType: &checker.Type{Kind: checker.TyList, Elem: checker.TypeI32},
							},
							Body: ast.Block{
								Stmts: []ast.Stmt{{Kind: ast.StmtBreak}},
							},
						},
					}},
				},
			}},
		}},
	}
	prog := l.Lower(file)
	fn := prog.Functions[0]
	if len(fn.Body) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(fn.Body))
	}
	if fn.Body[0].Kind != LStmtFor {
		t.Fatalf("expected For, got %d", fn.Body[0].Kind)
	}
	forData := fn.Body[0].Data.(*LFor)
	if forData.Var != "x" {
		t.Errorf("var = %q, want %q", forData.Var, "x")
	}
}

// ============================================================
// Enum variant tags with fields
// ============================================================

func TestLowerEnumVariantTags(t *testing.T) {
	l := NewLowerer()
	file := &ast.File{
		Blocks: []ast.ForgeBlock{{
			Name: "test",
			Enums: []ast.EnumDecl{{
				Name: "Shape",
				Variants: []ast.EnumVariant{
					{Name: "Circle", Fields: []ast.TupleField{{Type: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "f64"}}}}},
					{Name: "Rect", Fields: []ast.TupleField{
						{Type: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "f64"}}},
						{Type: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "f64"}}},
					}},
					{Name: "Point"},
				},
			}},
		}},
	}
	prog := l.Lower(file)
	e := prog.Enums[0]
	for i, v := range e.Variants {
		if v.Tag != i {
			t.Errorf("variant %q tag = %d, want %d", v.Name, v.Tag, i)
		}
	}
	if len(e.Variants[0].Fields) != 1 {
		t.Errorf("Circle fields = %d, want 1", len(e.Variants[0].Fields))
	}
	if len(e.Variants[1].Fields) != 2 {
		t.Errorf("Rect fields = %d, want 2", len(e.Variants[1].Fields))
	}
	if len(e.Variants[2].Fields) != 0 {
		t.Errorf("Point fields = %d, want 0", len(e.Variants[2].Fields))
	}
}

// ============================================================
// Channel type
// ============================================================

func TestLowerChannelType(t *testing.T) {
	l := NewLowerer()
	te := &ast.TypeExpr{
		Kind: ast.TypeChannel,
		Data: &ast.ChannelType{
			Elem: ast.TypeExpr{Kind: ast.TypeNamed, Data: &ast.NamedType{Name: "i32"}},
		},
	}
	lt := l.lowerTypeExpr(te)
	if lt.Kind != LTyChannel {
		t.Errorf("expected Channel, got %d", lt.Kind)
	}
	if lt.Elem.Kind != LTyI32 {
		t.Errorf("expected i32 elem, got %d", lt.Elem.Kind)
	}
}

// ============================================================
// Multiple functions
// ============================================================

func TestLowerMultipleFunctions(t *testing.T) {
	l := NewLowerer()
	file := &ast.File{
		Blocks: []ast.ForgeBlock{{
			Name: "test",
			Functions: []ast.FuncDecl{
				{Name: "alpha", Body: &ast.Block{}},
				{Name: "beta", Body: &ast.Block{}},
				{Name: "gamma", Body: &ast.Block{}},
			},
		}},
	}
	prog := l.Lower(file)
	if len(prog.Functions) != 3 {
		t.Fatalf("expected 3 functions, got %d", len(prog.Functions))
	}
	expected := []string{"alpha", "beta", "gamma"}
	for i, want := range expected {
		if prog.Functions[i].Name != want {
			t.Errorf("func %d = %q, want %q", i, prog.Functions[i].Name, want)
		}
	}
}
