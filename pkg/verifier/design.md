# Verifier Module Design

The `verifier` module is the structural integrity engine of the Forge system. It detects "architectural drift" by performing a deep comparison between the high-level architectural intent declared in `.forge` files and the actual implementation in Go source code. It bridges two distinct type systems to ensure that the developer's mental model, as captured in documentation and specifications, remains synchronized with the evolving codebase.

## Executive Summary

The `verifier` module provides a multi-stage analysis pipeline that validates Go source code against Forge specifications. It parses `.forge` files, discovers referenced Go source files, extracts their type information using the Go standard library's AST tools, and performs a structural comparison. The verifier identifies mismatches in fields, methods, function signatures, and naming conventions, reporting them with varying severity levels (Error, Warning, Info). A key feature is its "completeness check," which ensures that all exported Go symbols are documented in the corresponding Forge block, preventing the specification from becoming an incomplete or outdated subset of the implementation.

## File Inventory

- [verifier.go](verifier.go): The primary implementation file containing the verification logic, Go type extraction, and structural comparison algorithms.
- [verifier_test.go](verifier_test.go): Comprehensive test suite that validates drift detection, naming convention mapping, and completeness checks against both synthetic examples and the project's own specifications.
- [verifier.forge](verifier.forge): The module's own architectural specification, providing a self-documented model of its internal logic and public API.

## Architecture and Data Flow

The verification process is a stateless operation from the perspective of the caller, but it maintains significant internal state during a single run to aggregate information across multiple files.

The process begins by delegating to [pkg/parser](../parser/design.md) to produce an `ast.File` from the provided `.forge` source path. The verifier then inspects the `source:` annotations within each `forge` block to locate the relevant Go files. These paths are resolved relative to the directory containing the `.forge` file.

For each discovered source (which can be a file or a directory), the verifier uses `go/parser` and `go/ast` from the Go standard library to extract type information. This data is populated into a `goTypeInfo` registry, which serves as a comprehensive database of all structs, interfaces, functions, and type definitions found across all specified sources for a given block. This aggregation is a critical design choice, as a single Forge block's implementation may span multiple Go files within the same package.

Once the Go type registry is populated, the verifier recursively walks the Forge AST, comparing each declaration (struct, class, interface, enum, function) against the aggregated Go types. It normalizes naming conventions and type expressions on the fly to enable direct comparison. After validating the declared Forge symbols, the verifier performs a reverse check, walking all exported Go symbols in the source files to identify any that are missing from the Forge specification. Findings are collected into a `Result` object, which classifies each discrepancy by severity and provides detailed messages and file locations.

## Interface Implementations

The `verifier` module does not implement external interfaces defined in other packages. It is a standalone utility module that consumes the AST produced by [pkg/parser](../parser/design.md) and produces a `Result` report.

## Public API

The primary "front door" to the module is the `Verify(forgePath string) (*Result, error)` function. This function orchestrates the entire pipeline: parsing the Forge file, discovering Go sources, extracting types, and performing the comparison. It returns a `Result` containing all findings or an error if the verification process itself fails (e.g., file not found, parse error).

The `Result` type is a container for the findings of a verification run. It provides an `ErrorCount()` method, which is typically used by CLI tools or CI pipelines to determine if the verification should be considered a failure. Each individual discrepancy is represented by a `Finding` object, which includes a `Severity` (Error, Warning, Info), the paths to the involved Forge and Go files, and a descriptive message.

Severity levels are used to classify findings. `Error` represents fundamental mismatches such as missing types, incorrect field types, or parameter/return count mismatches. `Warning` indicates minor drift, such as extra Go fields not documented in Forge or constructor parameter mismatches. `Info` provides contextual information, such as additional Go methods that are not part of the Forge interface specification.

## Implementation Details

### Deep Type Comparison

The core of the verifier is the `forgeTypeToGoString` function, which recursively converts Forge `TypeExpr` nodes into a Go-compatible string format. This allows the verifier to compare types across the two systems. For example, optional types (`T?`) in Forge are mapped to pointers (`*T`) in Go, sequences (`[T]`) are mapped to slices (`[]T`), and maps map directly. Forge tuples are used to represent Go's multiple return values and are compared element-wise. The verifier also recognizes that `any` and `interface{}` are equivalent in modern Go.

To handle cross-package references, the `stripPackagePrefix` function heuristically removes package qualifiers (e.g., `ast.File` becomes `File`). This allows Forge files to use unqualified names while the Go implementation uses fully qualified types. The comparison logic in `typesMatch` is the final gate; if either side contains a "?" (representing an unconvertible type), it returns true to avoid reporting false positives for types the converter cannot yet handle.

### Name Mapping and Normalization

Forge typically uses `snake_case` for identifiers, while Go uses `PascalCase` for exported symbols and `camelCase` for unexported ones. The verifier bridges this gap by attempting three-way matching: `snakeToPascal` (e.g., `foo_bar` → `FooBar`) for exported fields, methods, and types; `snakeToCamel` (e.g., `foo_bar` → `fooBar`) for unexported identifiers; and exact matching for cases where the names already align. Forge primitive names are mapped to Go equivalents via `forgeNameToGo` (e.g., `i32` → `int32`, `f64` → `float64`).

### Verification Rules

The verifier applies specific rules based on the type of declaration. For structs and classes, it checks for field existence and type matching. For classes, it also verifies method signatures, including parameter counts (excluding the `self` parameter), positional types, and return types. Interfaces ensure every method declared in Forge exists in the Go interface with a matching signature. Enums are validated against three common Go patterns: typedefs (e.g., `type X int`), structs, or interfaces. Functions undergo positional parameter and return type comparison, with multi-return Go functions mapped to Forge tuple returns.

### Completeness Check

The `verifyCompleteness` function ensures the Forge specification is a "complete" model of the public API. It identifies all exported Go symbols (types and functions) in the referenced source files and reports an error for any that are not documented in the Forge block. Unexported symbols are ignored as they are considered internal implementation details.

## Dependencies

- **[pkg/ast](../ast/design.md)**: Used to represent the structure of the Forge file being verified.
- **[pkg/parser](../parser/design.md)**: Used to parse the `.forge` file into an AST.
- **Go Standard Library**:
    - `go/ast` and `go/parser`: Used to analyze the Go source code.
    - `go/token`: Used for position tracking in Go files.
    - `os` and `path/filepath`: Used for file system operations and path resolution.

## Technical Debt and Future Work

- **Expression Verification**: Currently, `requires` and `ensures` clauses are treated as raw strings and are not parsed or verified against the Go implementation.
- **Complex Pointer Indirection**: While basic optional-to-pointer mapping works, deeply nested or complex pointer structures may occasionally cause false positives.
- **Import Resolution**: The `stripPackagePrefix` mechanism is heuristic and does not perform full symbol resolution, which could lead to ambiguities in projects with many conflicting type names.
- **Performance**: The verifier re-parses Go files for every Forge block. For large projects, a caching mechanism for parsed Go ASTs would significantly improve performance.
- **Function Types**: Complex function types (e.g., `map[string]func(...)`) cannot currently be fully expressed in Forge and may be typed as `any`, leading to known false-positive errors in some edge cases.
