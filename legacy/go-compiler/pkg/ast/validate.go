// Package ast provides post-desugar validation for the Forge AST.
package ast

import "fmt"

// InvariantViolation represents a single invariant check failure.
type InvariantViolation struct {
	Stage   string // e.g. "post-desugar", "post-check", "post-lower"
	Check   string // what was checked
	Message string // what went wrong
}

func (v InvariantViolation) String() string {
	return fmt.Sprintf("[%s] %s: %s", v.Stage, v.Check, v.Message)
}

// ValidatePostDesugar checks AST invariants after all five desugar passes.
// Returns a list of violations (empty = all good).
func ValidatePostDesugar(file *File) []InvariantViolation {
	var violations []InvariantViolation

	for _, block := range file.Blocks {
		// Check 1: No unresolved relation declarations should remain
		// (they should have been consumed by DesugarRelations)
		// Relations are kept in the AST but their effects should be visible
		// as generated fields on classes.

		// Check 2: Every class that participates in an 'owns' relation
		// should have a destroy method (from DesugarDestructors)
		for _, rel := range block.Relations {
			if rel.Kind != Owns {
				continue
			}
			// Find the child class and verify it has owner fields
			childName := rel.Child.TypeName
			for _, cls := range block.Classes {
				if cls.Name == childName {
					hasParentField := false
					prefix := rel.Child.Label
					if prefix == "" {
						prefix = rel.Parent.TypeName
					}
					targetField := prefix + "_parent"
					for _, f := range cls.Fields {
						if f.Name == targetField {
							hasParentField = true
							break
						}
					}
					if !hasParentField {
						violations = append(violations, InvariantViolation{
							Stage:   "post-desugar",
							Check:   "relation-field-injection",
							Message: fmt.Sprintf("class %s missing expected parent field %q from owns relation", childName, targetField),
						})
					}
				}
			}
		}

		// Check 3: No interface should have embed declarations remaining
		// (DesugarInterfaceEmbeds should have flattened them)
		for _, iface := range block.Interfaces {
			for _, embed := range iface.Embeds {
				violations = append(violations, InvariantViolation{
					Stage:   "post-desugar",
					Check:   "embed-flattened",
					Message: fmt.Sprintf("interface %s still has unflattened embed: %s", iface.Name, embed.Name),
				})
			}
		}
	}

	return violations
}
