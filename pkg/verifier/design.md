# Verifier Module Design

## Executive Summary

The `verifier` module serves as the structural integrity engine of the Grok system, acting as the bridge between high-level architectural declarations and concrete implementation. Its primary mission is to detect "architectural drift"—the inevitable divergence that occurs as a codebase evolves away from its original design. By performing a deep, recursive comparison between the Abstract Syntax Tree (AST) of a `.grok` specification and the AST of the corresponding Go source code, the verifier ensures that the developer's mental model remains synchronized with reality. It identifies mismatches in types, field names, function signatures, and interface implementations, reporting them as errors, warnings, or informational findings depending on their impact on system integrity.

## File Inventory

- [verifier.go](verifier.go): The primary implementation file containing the core verification logic, Go source analysis, and type normalization routines.
- [verifier_test.go](verifier_test.go): The test suite for the module, featuring integration tests that run the verifier against the project's own `.grok` files and unit tests for naming convention transformations.
- [verifier.grok](verifier.grok): The self-documenting Grok specification for the verifier module, defining its own structural contracts and architectural intent.

## Architecture and Data Flow

The verifier operates as a multi-stage analysis pipeline that translates between two distinct type systems. The process is linear and stateless between calls, though it maintains significant internal state during a single verification run.

The workflow begins with **Grok Parsing**, where the module utilizes [pkg/parser](../parser/design.md) to transform a target `.grok` file into a [pkg/ast](../ast/design.md) representation. Once the Grok AST is available, the verifier enters the **Go Source Discovery** phase, inspecting the `source:` annotations within each `grok` block to identify the relevant Go files or directories.

The core of the analysis happens during **Go Type Extraction**. The verifier leverages the Go standard library's `go/parser` and `go/ast` packages to parse the identified source code. It traverses the resulting Go AST to populate an internal `goTypeInfo` structure, which serves as a comprehensive registry of all structs, interfaces, functions, and type definitions found across the specified sources. This aggregation allows the verifier to resolve types that may be spread across multiple files within the same package.

Finally, the module performs a **Structural Comparison**. It recursively walks the Grok AST, comparing each declaration against the aggregated Go type information. This phase involves complex normalization, mapping Grok's snake_case identifiers to Go's PascalCase or camelCase conventions and converting Grok's recursive type expressions into a format compatible with Go's type system. Discrepancies are accumulated into a `Result` object containing a list of `Finding` records.

## Interface Implementations

The `verifier` module does not implement any external interfaces defined within the Grok project. Instead, it acts as a high-level consumer of the [pkg/parser](../parser/design.md) and [pkg/ast](../ast/design.md) contracts. It also functions as a client to the Go standard library's AST interfaces, specifically consuming `*ast.File` and `*ast.GenDecl` nodes to build its internal model of the implementation.

## Public API

The public API is centered around a single, powerful entry point designed for integration into developer workflows and CI/CD pipelines.

- **Verify(grokPath string) (*Result, error)**: This is the primary "front door" of the module. It accepts a path to a `.grok` file, orchestrates the entire parsing and comparison pipeline, and returns a `Result` object. It handles file I/O and coordinates the interaction between the Grok parser and the Go source analyzer.
- **Result**: A container for the findings of a verification run. It provides an `ErrorCount()` method, allowing callers to quickly determine if any critical architectural drift was detected that should fail a build.
- **Finding**: A detailed record of a single discrepancy. Each finding includes a `Severity` (Error, Warning, or Info), the paths to the involved files, and a descriptive message explaining the nature of the drift.
- **Severity**: An enumeration that classifies the impact of a finding. `Error` indicates a fundamental mismatch (e.g., a missing type or wrong field type), `Warning` indicates minor drift (e.g., extra fields in Go not documented in Grok), and `Info` provides context (e.g., additional methods in Go that are omitted from the Grok interface).

## Implementation Details

### Go Type Extraction

The extraction process is handled by `extractGoTypes`, which performs a deep traversal of the Go AST. It populates the `goTypeInfo` structure, which is the central repository of implementation facts.
- **Structs**: The verifier captures field names and their types as strings. It also associates methods with their respective structs by inspecting function receivers.
- **Interfaces**: It extracts method signatures, including parameter types and return values, to ensure full interface compliance.
- **Functions**: Top-level functions are captured with their full signatures.
- **Type Definitions**: Simple type aliases (e.g., `type MyInt int`) are tracked to allow the verifier to recognize them as valid types even if they don't have complex internal structure.

### Type Mapping and Comparison

Bridging the gap between Grok and Go requires sophisticated normalization logic:
- **Naming Conventions**: The `snakeToPascal` and `snakeToCamel` functions are used to match Grok's idiomatic snake_case identifiers to Go's PascalCase (for exported fields/methods) or camelCase (for unexported ones).
- **Type Normalization**: The `grokTypeToGoString` function recursively converts Grok `TypeExpr` nodes into their Go string equivalents. For example, it maps Grok's optional types (`T?`) to Go pointers (`*T`) and Grok sequences (`[T]`) to Go slices (`[]T`).
- **Fuzzy Matching**: The `typesMatch` function handles nuances such as the equivalence of `any` and `interface{}`. Crucially, it implements `stripPackagePrefix`, which allows the verifier to match unqualified types in Grok (e.g., `File`) to qualified types in Go (e.g., `ast.File`), facilitating cross-package verification.

### Verification Logic

The verification logic is partitioned by declaration type, each with its own specific rules:
- **Structs and Classes**: The verifier checks for field existence and type compatibility. It also identifies "extra" fields in the Go implementation that are missing from the Grok declaration, reporting them as warnings to encourage complete documentation. For classes, it additionally verifies that constructor parameters correspond to struct fields.
- **Interfaces**: It ensures that every method declared in the Grok interface exists in the Go implementation with a matching signature.
- **Functions**: It performs positional comparison of parameters and return types. It specifically handles Grok's `self` parameter in class methods, excluding it from the positional count when comparing against Go's receiver-based methods.

## Dependencies

- [pkg/ast](../ast/design.md): Provides the foundational data structures for representing the Grok language.
- [pkg/parser](../parser/design.md): Used to transform Grok source text into the AST consumed by the verifier.
- **Go Standard Library**: The verifier depends heavily on `go/ast`, `go/parser`, and `go/token` for its analysis of the Go implementation.

## Technical Debt and Future Work

- **Expression Verification**: The current implementation focuses on structural types. Future work should include verifying the `requires` and `ensures` contracts by parsing and analyzing the expressions within those blocks.
- **Pointer Nuances**: While the verifier handles basic optional-to-pointer mapping, more robust handling of pointer vs. value receivers and complex indirection would reduce false positives in edge cases.
- **Import Resolution**: The current `stripPackagePrefix` approach is a heuristic. A more robust system would resolve imports to ensure that a Grok type `File` actually refers to the same `ast.File` being used in the Go code.
- **Performance Optimization**: For large projects, the verifier currently re-parses Go files for every `grok` block. Implementing a cache for parsed Go type information would significantly improve performance in large-scale applications.
