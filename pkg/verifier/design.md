# Verifier Module Design

The `verifier` module serves as the structural integrity engine for the Forge project, acting as a guardian against architectural drift. It performs a deep, recursive comparison between the high-level architectural intent declared in `.forge` files and the actual implementation in Go source code. By bridging two distinct type systems, it ensures that the developer's mental model—captured in documentation and specifications—remains synchronized with the evolving codebase.

## Executive Summary

The `verifier` module provides a multi-stage analysis pipeline that validates Go source code against Forge specifications. It parses `.forge` files, discovers referenced Go source files, extracts their type information using the Go standard library's AST tools, and performs a structural comparison. The verifier identifies mismatches in fields, methods, function signatures, and naming conventions, reporting them with varying severity levels (Error, Warning, Info). A key feature is its "completeness check," which ensures that all exported Go symbols are documented in the corresponding Forge block, preventing the specification from becoming an incomplete or outdated subset of the implementation. This self-correcting mechanism ensures that the documentation is an active, verifiable contract rather than a passive artifact.

## File Inventory

- [verifier.go](verifier.go): The primary implementation file containing the verification logic, Go type extraction, and structural comparison algorithms. It defines the `Verify` entry point and the `Result` and `Finding` types.
- [verifier_test.go](verifier_test.go): Comprehensive test suite that validates drift detection, naming convention mapping, and completeness checks against both synthetic examples and the project's own specifications.
- [verifier.forge](verifier.forge): The module's own architectural specification, providing a self-documented model of its internal logic and public API.

## Architecture and Data Flow

The verification process is designed as a stateless operation from the perspective of the caller, although it maintains significant internal state during a single run to aggregate information across multiple files. The workflow begins by delegating to the [pkg/parser](../parser/design.md) module to produce a `forgeast.File` from the provided `.forge` source path. The verifier then iterates through each `forge` block within the file, inspecting the `source:` annotations to locate the relevant Go files or directories. These paths are resolved relative to the directory containing the `.forge` file itself, ensuring that the specification and its implementation are co-located or predictably linked.

For each discovered source, the verifier utilizes the `go/parser` and `go/ast` packages from the Go standard library to extract detailed type information. This data is populated into a `goTypeInfo` registry, which serves as a comprehensive database of all structs, interfaces, functions, and type definitions found across all specified sources for a given block. This aggregation is a critical design choice, as a single Forge block's implementation may span multiple Go files within the same package. The `mergeGoInfo` function handles the complex task of combining information from multiple files, ensuring that methods defined in different files for the same struct are correctly associated and that field definitions are unified.

Once the Go type registry is fully populated, the verifier recursively walks the Forge AST, comparing each declaration—including structs, classes, interfaces, enums, and functions—against the aggregated Go types. It normalizes naming conventions and type expressions on the fly to enable direct comparison between the two systems. After validating the declared Forge symbols, the verifier performs a reverse check via `verifyCompleteness`. This function walks all exported Go symbols in the source files to identify any that are missing from the Forge specification, ensuring the documentation remains a complete representation of the public API. Findings are collected into a `Result` object, which classifies each discrepancy by severity and provides detailed messages and file locations.

## Interface Implementations

The `verifier` module is a standalone utility and does not implement interfaces defined in other packages. It acts as a consumer of the AST produced by [pkg/ast](../ast/design.md) and [pkg/parser](../parser/design.md) and provides its own reporting structures for use by CLI tools and CI pipelines.

## Public API

The primary entry point to the module is the `Verify(forgePath string) (*Result, error)` function. This function orchestrates the entire pipeline, from parsing the Forge file and discovering Go sources to extracting types and performing the final comparison. It returns a `Result` containing all findings or an error if the verification process itself fails due to file system issues or parsing errors.

The `Result` type acts as a container for the findings of a verification run. It provides an `ErrorCount()` method, which is the primary metric used by external tools to determine if the verification should be considered a failure. Each individual discrepancy is represented by a `Finding` object, which includes a `Severity` level, the paths to the involved Forge and Go files, and a descriptive message.

Severity levels are used to classify findings into three categories. `Error` represents fundamental mismatches such as missing types, incorrect field types, or parameter and return count mismatches. `Warning` indicates minor drift, such as extra Go fields not documented in Forge or constructor parameter mismatches. `Info` provides contextual information, such as additional Go methods that are not part of the Forge interface specification, which is expected for specifications that capture only high-level public concepts.

## Implementation Details

### Deep Type Comparison

The core of the verifier's logic resides in the `forgeTypeToGoString` function, which recursively converts Forge `TypeExpr` nodes into a Go-compatible string format. This transformation allows the verifier to compare types across the two systems using string equality. Key mappings include converting Forge optional types (`T?`) to Go pointers (`*T`), sequences (`[T]`) to Go slices (`[]T`), and maps directly to their Go equivalents. Forge tuples are used to represent Go's multiple return values and are unpacked during comparison to be checked element-wise against the Go function's return list. Forge `unit` represents a void return or an empty parameter list.

The `typesMatch` function serves as the final arbiter of type equality. It handles the equivalence of `any` and `interface{}` and employs a `stripPackagePrefix` helper to heuristically remove package qualifiers, such as converting `ast.File` to `File`. This allows Forge files to use unqualified names while the Go implementation uses fully qualified types. To avoid false positives, the function returns true if either side contains a "?" character, which represents an unconvertible or unknown type.

### Name Mapping and Normalization

Forge typically uses `snake_case` for identifiers, while Go uses `PascalCase` for exported symbols and `camelCase` for unexported ones. The verifier bridges this gap by attempting three-way matching in functions like `findGoName` and `findGoFieldType`. It checks for the `PascalCase` version (via `snake_to_pascal`) for exported fields and methods, the `camelCase` version (via `snake_to_camel`) for unexported identifiers, and finally an exact match. Forge primitive names are also mapped to their Go equivalents, such as `i32` to `int32` and `u8` to `uint8`.

### Verification Rules

The verifier applies specific rules based on the type of declaration encountered. For structs and classes, it checks for field existence and type matching. For classes, it additionally verifies method signatures, comparing parameter counts (excluding the `self` parameter) and ensuring that positional types and return types match. Interface verification ensures that every method declared in Forge exists in the Go interface with a matching signature. Enums are validated against three common Go patterns: typedefs, structs, or interfaces, and the verifier also checks for the common `XxxKind` naming pattern. Functions undergo positional parameter and return type comparison, with multi-return Go functions mapped to Forge tuple returns.

### Completeness Check

The `verifyCompleteness` function ensures that the Forge specification is a complete model of the public API. It identifies all exported Go symbols—types and functions—in the referenced source files and reports an error for any that are not documented in the Forge block. It uses the `isExported` helper, which checks for an uppercase first letter, to distinguish between the public API and internal implementation details.

## Dependencies

- **[pkg/ast](../ast/design.md)**: Provides the `ForgeBlock` and `TypeExpr` structures used to represent the Forge specification.
- **[pkg/parser](../parser/design.md)**: Used to parse the `.forge` file into an AST.
- **Go Standard Library**:
    - `go/ast` and `go/parser`: Used to analyze the Go source code and extract type information.
    - `go/token`: Used for position tracking in Go files.
    - `os` and `path/filepath`: Used for file system operations and path resolution.

## Technical Debt and Future Work

- **Expression Verification**: Currently, `requires` and `ensures` clauses are treated as raw strings and are not parsed or verified against the Go implementation. Future versions could use a Go expression parser to validate these constraints.
- **Complex Pointer Indirection**: While basic optional-to-pointer mapping works, deeply nested or complex pointer structures may occasionally cause false positives or be represented as `?`.
- **Import Resolution**: The `stripPackagePrefix` mechanism is heuristic and does not perform full symbol resolution, meaning it cannot distinguish between types with the same name in different packages.
- **Performance**: The verifier re-parses Go files for every Forge block. For large projects, a caching mechanism for parsed Go ASTs and `goTypeInfo` would significantly improve performance.
- **Complex Function Types**: Complex function types, such as maps containing functions, cannot currently be fully expressed in Forge and may be typed as `any` or `?`, leading to known false-positive errors in some edge cases.
- **Generic Constraints**: While basic generic types are supported, complex constraints are not currently compared.
