# Checker Module Design

The `checker` module implements the semantic analysis phase of the Forge compiler. It is responsible for type checking, type inference, name resolution, and enforcing language-specific constraints on the desugared Abstract Syntax Tree (AST) produced by the `ast` module.

## Executive Summary

The `checker` module ensures that a Forge program is semantically valid before it is passed to the backend for code generation. It performs a comprehensive pass over the AST, resolving type expressions into concrete `Type` objects, tracking variable scopes, and verifying that all expressions and statements adhere to Forge's type system rules. Key features include structural subtyping for interfaces, numeric widening, automatic optional wrapping, and support for generic type inference. The checker also handles module imports, ensuring that symbols are correctly resolved across file boundaries while detecting circular dependencies. It also enforces concurrency safety through `guarded_by` annotations and validates the exhaustiveness of `match` statements for enums and unions.

## File Inventory

- [checker.go](checker.go): The primary implementation of the type checker. It defines the core `Type` system, `Scope` management, `Registry` for user-defined types, and the `Checker` struct which contains the logic for traversing and validating the AST.
- [checker_test.go](checker_test.go): A comprehensive suite of tests that verify the checker's behavior across a wide range of Forge language features, including basic arithmetic, control flow, function calls, struct/class definitions, and generic type inference.
- [checker.forge](checker.forge): A Forge-language specification of the checker's own architecture and logic, serving as both documentation and a formal model of the module's behavior.

## Architecture and Data Flow

The `checker` module operates on the AST after it has been parsed and desugared. It follows a multi-pass approach to handle forward references and cross-module dependencies.

1.  **Initialization**: A new `Checker` instance is created, initializing the global scope with builtin functions (like `println`, `len`, `append`, `isnull`, `make_channel`) and builtin types (like `i32`, `string`, `bool`, `error`).
2.  **Module Loading**: When `CheckFile` or `CheckModuleFile` is called, the checker begins traversing the AST. If it encounters an `import` statement for a `.fg` file, it recursively loads and checks the imported module, caching the results in a `modules` map to avoid redundant work and using a `checking` stack to detect import cycles.
3.  **Type Registration (Pass 1)**: The checker performs a declaration pass where it registers all user-defined types (structs, classes, enums, interfaces) in a global `Registry`. This allows for forward references within a module. Interfaces are registered first, followed by other types, to ensure that `implements` clauses can be validated.
4.  **Semantic Validation (Pass 2)**: The checker then performs a deep traversal of the AST:
    - **Scope Management**: It maintains a stack of `Scope` objects to handle nested blocks, function parameters, and local variables.
    - **Type Resolution**: `TypeExpr` nodes in the AST are resolved into `Type` objects.
    - **Expression Inference**: For every expression, the checker infers its type and stores it in the `ResolvedType` field of the `ast.Expr` node.
    - **Statement Checking**: Statements are verified for correctness (e.g., ensuring `if` conditions are boolean, verifying return types match function signatures, enforcing mutability rules).
    - **Concurrency Safety**: The checker tracks "held locks" and verifies that fields annotated with `guarded_by` are only accessed within a corresponding `lock` block.
5.  **Error Reporting**: Any semantic violations are captured as `CheckError` objects, which include a descriptive message and the source location (`Span`) where the error occurred. The checker is designed to be resilient, logging errors and continuing where possible to find multiple issues in one pass.

## Interface Implementations

The `checker` module does not explicitly implement external Go interfaces, but it provides the core logic that satisfies the semantic requirements of the Forge language. It consumes the AST nodes defined in [pkg/ast](../ast/design.md) and uses the parser from [pkg/parser](../parser/design.md) to handle imports.

## Public API

### Core Types

- **`Type`**: Represents a resolved type in the Forge system. It uses a `Kind` (e.g., `TyInt`, `TyFunc`, `TyStruct`, `TyUnion`) and additional metadata (e.g., bit width, element types, field lists, type parameters) to describe the type.
- **`Checker`**: The main entry point for semantic analysis. It provides methods like `CheckFile(file *ast.File)` and `Errors() []error`.
- **`Registry`**: A collection of `TypeInfo` objects representing all named types known to the checker.
- **`Scope`**: A hierarchical mapping of identifiers to their resolved types and mutability status.
- **`ModuleExports`**: A structure that holds the public types and functions exported by a module, used for cross-module symbol resolution.

### Primary Methods

- **`New()`**: Constructs a new `Checker` with initialized builtins.
- **`CheckFile(file *ast.File)`**: Performs type checking on the provided AST. This is the main "front door" for checking a single file.
- **`CheckModuleFile(importPath string, fromSpan ast.Span)`**: Resolves, parses, and checks a module at the given path, returning its exported symbols.
- **`Errors()`**: Returns the list of all errors encountered during the checking process.

## Implementation Details

### Type Representation and Equality

The `Type` struct is the heart of the module. It uses a recursive structure to represent complex types like lists, maps, functions, and unions. Structural equality is implemented in the `Equal` method, which ensures that two types are identical in their composition.

### Subtyping and Assignability

Forge supports several forms of implicit type compatibility implemented in the `assignableTo` method:
- **Numeric Widening**: Smaller integer types can be implicitly converted to larger ones (e.g., `i8` to `i32`), and platform-sized `int`/`uint` types are compatible with their fixed-size counterparts.
- **Optional Wrapping**: A type `T` is always assignable to `T?`.
- **Structural Subtyping**: A class or struct is considered to implement an interface if it provides all the methods defined by that interface with matching signatures. This is checked dynamically during the `assignableTo` call.
- **Union Types**: A type `T` is assignable to a union type `U` if `T` is assignable to any of the variants of `U`. Conversely, a union type `U` is assignable to `T` only if all variants of `U` are assignable to `T`.

### Generic Type Inference

The checker implements a unification-based inference for generic functions. When a generic function is called without explicit type arguments, the `inferTypeArgs` method walks the parameter types and actual argument types in parallel. It binds type variables (e.g., `T`) to concrete types from the arguments. If a type variable is used in multiple positions, the first match wins, and subsequent uses must be compatible. The inferred types are then substituted into the function's return type.

### Relational Constraints

Forge supports relational constraints like `where Graph<G, N, E>`. When such a constraint is encountered in a function's `where` clause, the checker looks up the corresponding interface and "grants" its methods to the involved type variables. This allows methods like `G.nodes()` or `N.out_edges()` to be called on variables of type `G` or `N` within the function body.

### Concurrency and Guarded Fields

The checker enforces safety rules for concurrent access. It tracks "held locks" within a scope (populated by `lock` statements) and verifies that fields annotated with `guarded_by` are only accessed when the corresponding lock is held. The lock name is extracted from the expression (e.g., `self.mu`).

### Exhaustiveness Checking

For `match` statements operating on `enum` or `union` types, the checker verifies that all possible variants or member types are covered by the match arms. If a wildcard pattern (`_`) is not present, the checker reports an error listing the missing variants.

### Built-in Methods and Functions

The checker provides specialized resolution for methods on built-in types:
- **`string`**: `len`, `contains`, `has_prefix`, `has_suffix`, `to_upper`, `to_lower`, `trim`, `replace`, `split`, `index_of`, `repeat`.
- **`list`**: `len`, `push`, `pop`, `contains`, `reverse`, `join`.
- **`map`**: `len`, `contains_key`, `keys`, `values`.
- **`channel`**: `send`, `receive`, `close`.

Built-in functions like `println`, `len`, `append`, `isnull`, and `make_channel` are pre-defined in the global scope.

## Dependencies

- **[pkg/ast](../ast/design.md)**: The checker consumes the AST and stores results back into it.
- **[pkg/parser](../parser/design.md)**: Used to parse imported module files.
- **Standard Library**: Uses `fmt`, `os`, `path/filepath`, and `strings` for file I/O and string manipulation.

## Technical Debt and Future Work

- **Advanced Inference**: The current type inference is relatively simple. More complex scenarios involving nested generics or circular constraints may require a more robust constraint-solver-based approach (like Hindley-Milner).
- **Method Resolution**: Method lookup currently relies on a simple name-based search in the `TypeInfo`. Support for method overloading or more complex trait-based dispatch might be needed in the future.
- **Performance**: For very large projects, the recursive module checking and deep AST traversal could become a bottleneck. Implementing incremental checking or a more parallelized approach could improve performance.
- **Circular Type Definitions**: While module cycles are detected, circular type definitions (e.g., two structs containing each other directly) need careful handling to avoid infinite recursion during size calculation or equality checks.
