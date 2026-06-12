package ast_test

import (
	"testing"

	"github.com/waywardgeek/forge/pkg/ast"
	"github.com/waywardgeek/forge/pkg/parser"
)

func parseForTest(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(src, "test.fg")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return file
}

// ============================================================
// 1. DesugarInterfaceEmbeds
// ============================================================

func TestEmbedCopiesFields(t *testing.T) {
	src := `forge t {
		interface Base<P, C> {
			field P.base_field: i32
		}
		interface Extended<P, C> {
			embed Base<P, C>
			field P.ext_field: string
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)

	extended := &file.Blocks[0].Interfaces[1]
	if len(extended.Fields) < 2 {
		t.Fatalf("expected ≥2 fields after embed, got %d", len(extended.Fields))
	}
}

func TestEmbedSubstitutesTypeParams(t *testing.T) {
	src := `forge t {
		interface Base<A, B> {
			field A.data: B?
		}
		interface Derived<X, Y> {
			embed Base<X, Y>
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)

	derived := &file.Blocks[0].Interfaces[1]
	if len(derived.Fields) < 1 {
		t.Fatal("expected embed to copy field")
	}
	if derived.Fields[0].TypeParam != "X" {
		t.Errorf("expected type param substitution A→X, got %q", derived.Fields[0].TypeParam)
	}
}

func TestEmbedCopiesDestructors(t *testing.T) {
	src := `forge t {
		interface Base<P, C> {
			destructor P {
				let x = 1
			}
		}
		interface Extended<P, C> {
			embed Base<P, C>
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)

	extended := &file.Blocks[0].Interfaces[1]
	if len(extended.Destructors) < 1 {
		t.Fatal("expected embed to copy destructor")
	}
	if extended.Destructors[0].TypeParam != "P" {
		t.Errorf("expected destructor type param P, got %q", extended.Destructors[0].TypeParam)
	}
}

// ============================================================
// 2. DesugarInterfaceFields
// ============================================================

func TestFieldGeneratesGetterSetter(t *testing.T) {
	src := `forge t {
		interface Foo<P, C> {
			field P.name: string
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)

	iface := &file.Blocks[0].Interfaces[0]
	if len(iface.Methods) < 2 {
		t.Fatalf("expected ≥2 methods from field, got %d", len(iface.Methods))
	}

	getter := iface.Methods[0]
	if getter.Name != "name" {
		t.Errorf("getter name = %q, want %q", getter.Name, "name")
	}
	if getter.ReceiverType != "P" {
		t.Errorf("getter receiver = %q, want %q", getter.ReceiverType, "P")
	}
	if len(getter.Params) != 1 || !getter.Params[0].IsSelf {
		t.Error("getter should have 1 self param")
	}
	if getter.ReturnType == nil {
		t.Error("getter should have return type")
	}

	setter := iface.Methods[1]
	if setter.Name != "set_name" {
		t.Errorf("setter name = %q, want %q", setter.Name, "set_name")
	}
	if len(setter.Params) != 2 {
		t.Fatalf("setter should have 2 params, got %d", len(setter.Params))
	}
	if !setter.Params[0].IsSelf || !setter.Params[0].IsMut {
		t.Error("setter first param should be mut self")
	}
	if setter.Params[1].Name != "val" {
		t.Errorf("setter second param = %q, want %q", setter.Params[1].Name, "val")
	}
}

func TestMultipleFieldsGenerateMultipleMethods(t *testing.T) {
	src := `forge t {
		interface Container<P, C> {
			field P.items: [C]
			field C.parent_ref: P?
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)

	iface := &file.Blocks[0].Interfaces[0]
	if len(iface.Methods) != 4 {
		t.Fatalf("expected 4 methods (2 fields × getter+setter), got %d", len(iface.Methods))
	}
	expected := []string{"items", "set_items", "parent_ref", "set_parent_ref"}
	for i, want := range expected {
		if iface.Methods[i].Name != want {
			t.Errorf("method[%d] = %q, want %q", i, iface.Methods[i].Name, want)
		}
	}
}

// ============================================================
// 3. DesugarRelations
// ============================================================

func TestRelationInjectsFields(t *testing.T) {
	src := `forge t {
		interface ArrayList<P, C> {
			field P.children: [C]
			field C.parent_ref: P?
		}
		class Parent { }
		class Child { }
		relation ArrayList Parent:p owns [Child:c]
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)
	ast.DesugarRelations(file)

	block := &file.Blocks[0]
	parent := &block.Classes[0]
	child := &block.Classes[1]

	if len(parent.Fields) < 1 {
		t.Fatal("parent should have injected field")
	}
	if parent.Fields[0].Name != "p_children" {
		t.Errorf("parent field = %q, want %q", parent.Fields[0].Name, "p_children")
	}

	if len(child.Fields) < 1 {
		t.Fatal("child should have injected field")
	}
	if child.Fields[0].Name != "c_parent_ref" {
		t.Errorf("child field = %q, want %q", child.Fields[0].Name, "c_parent_ref")
	}
}

func TestRelationGeneratesImplBlock(t *testing.T) {
	src := `forge t {
		interface ArrayList<P, C> {
			field P.children: [C]
		}
		class Parent { }
		class Child { }
		relation ArrayList Parent:p owns [Child:c]
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)
	ast.DesugarRelations(file)

	block := &file.Blocks[0]
	if len(block.ImplBlocks) < 1 {
		t.Fatal("expected generated impl block")
	}
	impl := block.ImplBlocks[0]
	if impl.InterfaceName != "ArrayList" {
		t.Errorf("impl interface = %q, want %q", impl.InterfaceName, "ArrayList")
	}
	if len(impl.TypeArgs) != 2 {
		t.Fatalf("expected 2 type args, got %d", len(impl.TypeArgs))
	}
	if len(impl.Mappings) < 2 {
		t.Errorf("expected ≥2 mappings (getter+setter), got %d", len(impl.Mappings))
	}
}

func TestRelationFieldTypeRewriting(t *testing.T) {
	src := `forge t {
		interface ArrayList<P, C> {
			field P.children: [C]
		}
		class Parent { }
		class Child { }
		relation ArrayList Parent:p owns [Child:c]
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)
	ast.DesugarRelations(file)

	parent := &file.Blocks[0].Classes[0]
	if len(parent.Fields) < 1 {
		t.Fatal("parent should have field")
	}
	field := parent.Fields[0]
	if field.Type.Kind == 0 {
		t.Fatal("field should have type")
	}
	// Type should be [Child], not [C]
	if field.Type.Kind != ast.TypeSequence {
		t.Fatalf("field type kind = %v, want TypeSequence", field.Type.Kind)
	}
}

func TestRelationMergesExistingImpl(t *testing.T) {
	src := `forge t {
		interface ArrayList<P, C> {
			field P.children: [C]
		}
		class A { }
		class B { }
		relation ArrayList A:a owns [B:b]
		impl ArrayList<A, B> {
			A.children = A.a_children
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)
	ast.DesugarRelations(file)

	block := &file.Blocks[0]
	implCount := 0
	for _, ib := range block.ImplBlocks {
		if ib.InterfaceName == "ArrayList" {
			implCount++
		}
	}
	if implCount > 1 {
		t.Errorf("expected 1 impl block (merged), got %d", implCount)
	}
}

// ============================================================
// 4. DesugarDestructors
// ============================================================

func TestDestructorCreatesDestroyMethod(t *testing.T) {
	src := `forge t {
		interface Owning<P, C> {
			field P.items: [C]
			destructor P {
				let x = 42
			}
		}
		class Container { }
		class Item { }
		relation Owning Container:ct owns [Item:it]
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)
	ast.DesugarRelations(file)
	ast.DesugarDestructors(file)

	container := &file.Blocks[0].Classes[0]
	found := false
	for _, m := range container.Methods {
		if m.Name == "destroy" {
			found = true
			if m.Body == nil {
				t.Error("destroy method should have a body")
			}
		}
	}
	if !found {
		t.Error("expected destroy method on Container")
	}
}

// ============================================================
// 5. DesugarDefaultImpls
// ============================================================

func TestDefaultImplExtractsToToplevel(t *testing.T) {
	src := `forge t {
		interface Printable<T> {
			func T.to_string(self) -> string {
				return "default"
			}
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarDefaultImpls(file)

	block := &file.Blocks[0]
	found := false
	for _, f := range block.Functions {
		if f.Name == "to_string" {
			found = true
			if len(f.Where) < 1 {
				t.Error("extracted func should have where clause")
			} else if f.Where[0].Constraint != "Printable" {
				t.Errorf("where constraint = %q, want %q", f.Where[0].Constraint, "Printable")
			}
		}
	}
	if !found {
		t.Error("to_string should be extracted to top-level")
	}

	iface := &block.Interfaces[0]
	for _, m := range iface.Methods {
		if m.Body != nil {
			t.Errorf("method %q should not have body after desugar", m.Name)
		}
	}
}

func TestDefaultImplKeepsAbstractMethods(t *testing.T) {
	src := `forge t {
		interface Renderable<T> {
			func T.render(self) -> string
			func T.debug(self) -> string {
				return "debug"
			}
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarDefaultImpls(file)

	iface := &file.Blocks[0].Interfaces[0]
	if len(iface.Methods) < 1 {
		t.Fatal("abstract method should remain")
	}
	if iface.Methods[0].Name != "render" {
		t.Errorf("remaining method = %q, want %q", iface.Methods[0].Name, "render")
	}
	if iface.Methods[0].Body != nil {
		t.Error("render should have no body")
	}
}

// ============================================================
// Integration & ordering
// ============================================================

func TestDesugarAllSmoke(t *testing.T) {
	src := `forge t {
		interface OwnList<P, C> {
			field P.items: [C]
			destructor P {
				let x = 1
			}
			func P.count(self) -> i32 {
				return 0
			}
		}
		class Folder { }
		class Document { }
		relation OwnList Folder:f owns [Document:d]
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)
	ast.DesugarRelations(file)
	ast.DesugarDestructors(file)
	ast.DesugarDefaultImpls(file)

	block := &file.Blocks[0]

	if len(block.Classes[0].Fields) < 1 {
		t.Error("folder should have injected field")
	}

	hasDestroy := false
	for _, m := range block.Classes[0].Methods {
		if m.Name == "destroy" {
			hasDestroy = true
		}
	}
	if !hasDestroy {
		t.Error("folder should have destroy method")
	}

	hasCount := false
	for _, f := range block.Functions {
		if f.Name == "count" {
			hasCount = true
		}
	}
	if !hasCount {
		t.Error("count should be extracted to top-level")
	}

	if len(block.ImplBlocks) < 1 {
		t.Error("should have generated impl block")
	}
}

func TestEmbedsBeforeFields(t *testing.T) {
	src := `forge t {
		interface Base<P, C> {
			field P.base_val: i32
		}
		interface Extended<P, C> {
			embed Base<P, C>
		}
	}`
	file := parseForTest(t, src)
	ast.DesugarInterfaceEmbeds(file)
	ast.DesugarInterfaceFields(file)

	extended := &file.Blocks[0].Interfaces[1]
	if len(extended.Methods) < 2 {
		t.Fatalf("expected ≥2 methods from embedded field, got %d", len(extended.Methods))
	}
	if extended.Methods[0].Name != "base_val" {
		t.Errorf("getter = %q, want %q", extended.Methods[0].Name, "base_val")
	}
	if extended.Methods[1].Name != "set_base_val" {
		t.Errorf("setter = %q, want %q", extended.Methods[1].Name, "set_base_val")
	}
}
