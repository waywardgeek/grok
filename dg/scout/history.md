### USER (gemini)

# Task: Synthesize Project Architecture

Please synthesize a high-level `project-design.md` by analyzing the individual `design.md` files from all modules. This document will be our primary map for navigating this project in the future.

## Project Context

- Project Path: /Users/bill/projects/forge
- Total Modules: 6

## Modules to Analyze

- ast: pkg/ast/design.md
- checker: pkg/checker/design.md
- lir: pkg/lir/design.md
- parser: pkg/parser/design.md
- verifier: pkg/verifier/design.md
- runtime: runtime/design.md

## Your Strategy

1. **Read Designs**: Read the `design.md` file for every module listed above.
2. **Map Interfaces**: Identify the primary interfaces that define the system's boundaries and how modules satisfy them.
3. **Synthesize Narrative**: Create a cohesive narrative at the project root focusing on module interactions, shared patterns, and global data flow.
5. **Module Map**: Include a **categorized tree view** of all modules, with each entry being a **clickable markdown link** to its `design.md` file (e.g., `[pkg/scanner](pkg/scanner/design.md)`).
6. **Use Tools**: Use `write_file` and `edit_file` to build the document in sections. Do NOT output the whole document at once.

When finished, confirm: 'Project documentation complete.'


---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/ast/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_29bada5c50adf8d7]
File: /Users/bill/projects/forge/pkg/ast/design.md (88 lines)

# AST Module Design

The `ast` module defines the Abstract Syntax Tree (AST) for the Forge language. It provides the data structures used by the parser to represent the structure of Forge source code, including declarations, expressions, statements, and types. Additionally, it includes a desugaring phase that transforms high-level language constructs into more fundamental AST nodes before semantic analysis.

## Executive Summary

The `ast` module is the backbone of the Forge compiler's frontend. It serves as the intermediate representation between the raw source code and the semantic checker. The module is designed to be simple and data-centric, using a "Kind + Data" pattern to represent polymorphic nodes like expressions and statements. A key feature of this module is its desugaring capability, which automates the expansion of complex features like relations and interface fields into standard class members and implementation blocks.

## File Inventory

- [ast.go](ast.go): Defines the core AST nodes for declarations (functions, classes, structs, enums, interfaces, relations, etc.) and source location tracking (`Pos` and `Span`).
- [expr.go](expr.go): Defines AST nodes for expressions (literals, calls, operators, etc.), statements (variable declarations, control flow, etc.), and patterns used in match expressions.
- [desugar.go](desugar.go): Implements transformations that simplify the AST, such as expanding relations into fields and methods, and extracting default interface implementations.

## Architecture and Data Flow

The AST is the central data structure through which information flows during the compilation process.

1.  **Parsing**: The parser reads Forge source files and constructs a `File` AST node, which contains one or more `ForgeBlock` nodes.
2.  **Desugaring**: Before type checking, the AST undergoes several desugaring passes defined in `desugar.go`. These passes transform high-level "sugar" into explicit declarations. For example, a `relation` declaration is expanded into concrete fields in the involved classes and `impl` blocks that satisfy the interface requirements.
3.  **Semantic Analysis**: The checker traverses the desugared AST to perform name resolution and type checking. It stores the results of this analysis directly in the AST nodes using `ResolvedType` fields (e.g., in `Expr` and `Pattern`).
4.  **Lowering/Codegen**: The fully checked and desugared AST is then used by the backend to generate target code (such as LLVM IR or C++).

## Interface Implementations

The AST nodes are primarily data structures and do not implement complex interfaces. However, they are designed to be easily traversable. The `File` type provides a `DocComment` method to retrieve comments associated with specific lines, which is useful for generating documentation or processing annotations.

## Public API

### Core Types

- **`File`**: The root of the AST for a single source file. It contains a list of `ForgeBlock` nodes and all comments found in the file.
- **`ForgeBlock`**: Represents a `forge { ... }` block, which is the primary container for declarations in Forge.
- **`TypeExpr`**: A polymorphic container for type expressions (e.g., named types, optionals, unions, functions).
- **`Expr`**: A polymorphic container for all expression types. It includes a `ResolvedType` field used by the checker.
- **`Stmt`**: A polymorphic container for all statement types.
- **`Block`**: A sequence of statements, typically representing a function body or a scoped block of code.

### Desugaring Functions

The module exports several functions to perform AST transformations:

- **`DesugarInterfaceFields`**: Converts `field` declarations within interfaces into getter and setter method signatures.
- **`DesugarRelations`**: The most significant transformation. It processes `relation` declarations by injecting the necessary fields into concrete classes and generating `impl` blocks that map interface methods to these fields.
- **`DesugarDefaultImpls`**: Moves methods with bodies from interfaces to the top-level as standalone functions, adding the appropriate type parameters and `where` clauses to preserve their relational context.
- **`DesugarDestructors`**: Generates `destroy` methods for classes involved in `owns` relations, incorporating destructor logic defined in interfaces.

## Implementation Details

### Polymorphism via Kind and Data

To avoid complex interface hierarchies and deep nesting, the AST uses a consistent pattern for polymorphic nodes:

```go
type Expr struct {
    Kind ExprKind
    Data any
    Span Span
    ResolvedType any
}
```

The `Kind` field (an enum) identifies the specific type of the node, and the `Data` field holds the corresponding struct (e.g., `*BinaryExpr`, `*CallExpr`). This allows for efficient switching and processing without the overhead of Go interfaces for every node type.

### Source Mapping

Every major AST node includes a `Span` field, which stores the start and end `Pos` (file, line, column) in the original source code. This is critical for providing high-quality error messages during checking and for supporting IDE features like "go to definition."

### Desugaring Logic

The desugaring process is designed to be idempotent and order-dependent:

1.  **Interface Fields**: First, fields in interfaces are turned into method signatures.
2.  **Relations**: Next, relations are processed. This step is "generative" as it creates new fields in classes and new `impl` blocks. It uses the `rewriteFieldType` helper to map interface type parameters to the concrete types specified in the relation.
3.  **Default Impls**: Finally, default implementations are extracted. By moving them to the top level, the rest of the compiler can treat them as regular generic functions.

## Dependencies

The `ast` module is a leaf node in the package hierarchy; it does not depend on other internal Forge packages. This prevents circular dependencies, especially since the `checker` and `parser` both depend heavily on the `ast` module.

The `ResolvedType` and `InferredTypeArgs` fields use the `any` type to avoid a direct dependency on the `checker` package, which would create a circular import.

## Technical Debt and Future Work

- **Type Safety**: The use of `any` for `Data` and `ResolvedType` fields sacrifices compile-time type safety for flexibility and cycle avoidance. Future iterations might explore using more specific interfaces or generics if the Go version allows.
- **In-place Transformation**: Desugaring modifies the AST in-place. While efficient, this loses the original "sugar" representation. If the compiler needs to support refactoring tools or high-fidelity source-to-source transformations, a non-destructive approach might be required.
- **Visitor Pattern**: Currently, traversal is done via manual switches. Implementing a standard Visitor pattern might simplify complex AST walks in the checker and codegen phases.
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/checker/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_b0cf5c0396572ca3]
File: /Users/bill/projects/forge/pkg/checker/design.md (87 lines)

# Checker Module Design

The `checker` module implements the semantic analysis phase of the Forge compiler. It is responsible for type checking, type inference, name resolution, and enforcing language-specific constraints on the desugared Abstract Syntax Tree (AST) produced by the `ast` module.

## Executive Summary

The `checker` module ensures that a Forge program is semantically valid before it is passed to the backend for code generation. It performs a comprehensive pass over the AST, resolving type expressions into concrete `Type` objects, tracking variable scopes, and verifying that all expressions and statements adhere to Forge's type system rules. Key features include structural subtyping for interfaces, numeric widening, automatic optional wrapping, and support for generic type inference. The checker also handles module imports, ensuring that symbols are correctly resolved across file boundaries while detecting circular dependencies.

## File Inventory

- [checker.go](checker.go): The primary implementation of the type checker. It defines the core `Type` system, `Scope` management, `Registry` for user-defined types, and the `Checker` struct which contains the logic for traversing and validating the AST.
- [checker_test.go](checker_test.go): A comprehensive suite of tests that verify the checker's behavior across a wide range of Forge language features, including basic arithmetic, control flow, function calls, struct/class definitions, and generic type inference.
- [checker.forge](checker.forge): A Forge-language specification of the checker's own architecture and logic, serving as both documentation and a formal model of the module's behavior.

## Architecture and Data Flow

The `checker` module operates on the AST after it has been parsed and desugared.

1.  **Initialization**: A new `Checker` instance is created, initializing the global scope with builtin functions (like `println`, `len`, `append`) and builtin types (like `i32`, `string`, `bool`).
2.  **Module Loading**: When `CheckFile` or `CheckModuleFile` is called, the checker begins traversing the AST. If it encounters an `import` statement, it recursively loads and checks the imported module, caching the results to avoid redundant work and using a stack to detect import cycles.
3.  **Type Registration**: The checker first performs a "declaration pass" where it registers all user-defined types (structs, classes, enums, interfaces) in a global `Registry`. This allows for forward references within a module.
4.  **Semantic Validation**: The checker then performs a deep traversal of the AST:
    - **Scope Management**: It maintains a stack of `Scope` objects to handle nested blocks, function parameters, and local variables.
    - **Type Resolution**: `TypeExpr` nodes in the AST are resolved into `Type` objects.
    - **Expression Inference**: For every expression, the checker infers its type and stores it in the `ResolvedType` field of the `ast.Expr` node.
    - **Statement Checking**: Statements are verified for correctness (e.g., ensuring `if` conditions are boolean, verifying return types match function signatures).
5.  **Error Reporting**: Any semantic violations are captured as `CheckError` objects, which include a descriptive message and the source location (`Span`) where the error occurred.

## Interface Implementations

The `checker` module does not explicitly implement external interfaces, but it provides the core logic that satisfies the semantic requirements of the Forge language. It consumes the AST nodes defined in [pkg/ast](../ast/design.md) and uses the parser from `pkg/parser` to handle imports.

## Public API

### Core Types

- **`Type`**: Represents a resolved type in the Forge system. It uses a `Kind` (e.g., `TyInt`, `TyFunc`, `TyStruct`) and additional metadata (e.g., bit width, element types, field lists) to describe the type.
- **`Checker`**: The main entry point for semantic analysis. It provides methods like `CheckFile(file *ast.File)` and `Errors() []error`.
- **`ModuleExports`**: A structure that holds the public types and functions exported by a module, used for cross-module symbol resolution.

### Primary Methods

- **`New()`**: Constructs a new `Checker` with initialized builtins.
- **`CheckFile(file *ast.File)`**: Performs type checking on the provided AST. This is the main "front door" for checking a single file.
- **`CheckModuleFile(importPath string, fromSpan ast.Span)`**: Resolves, parses, and checks a module at the given path, returning its exported symbols.
- **`Errors()`**: Returns the list of all errors encountered during the checking process.


## Implementation Details

### Type Representation and Equality

The `Type` struct is the heart of the module. It uses a recursive structure to represent complex types like lists, maps, and functions. Structural equality is implemented in the `Equal` method, which ensures that two types are identical in their composition.

### Subtyping and Assignability

Forge supports several forms of implicit type compatibility:
- **Numeric Widening**: Smaller integer types can be implicitly converted to larger ones (e.g., `i8` to `i32`), and platform-sized `int`/`uint` types are compatible with their fixed-size counterparts.
- **Optional Wrapping**: A type `T` is always assignable to `T?`.
- **Structural Subtyping**: A class or struct is considered to implement an interface if it provides all the methods defined by that interface with matching signatures. This is checked dynamically during the `assignableTo` call.
- **Union Types**: A type `T` is assignable to a union type `U` if `T` is assignable to any of the variants of `U`. Conversely, a union type `U` is assignable to `T` only if all variants of `U` are assignable to `T`.

### Generic Type Inference

The checker implements a basic form of unification for generic functions. When a generic function is called, the `inferTypeArgs` method walks the parameter types and actual argument types in parallel. It binds type variables (e.g., `T`) to concrete types from the arguments. If a type variable is used in multiple positions, the first match wins, and subsequent uses must be compatible.

### Concurrency and Guarded Fields

The checker enforces safety rules for concurrent access. It tracks "held locks" within a scope and verifies that fields annotated with `guarded_by` are only accessed when the corresponding lock is held (e.g., via a `lock` statement).

### Error Handling

The checker is designed to be resilient. When it encounters an error, it logs the error and returns a sentinel `TypeError` or `TypeUnknown`. This allows the checker to continue processing the rest of the file to find as many errors as possible in a single pass.

## Dependencies

- **[pkg/ast](../ast/design.md)**: The checker consumes the AST and stores results back into it.
- **`pkg/parser`**: Used to parse imported module files.
- **Standard Library**: Uses `fmt`, `os`, `path/filepath`, and `strings` for file I/O and string manipulation.

## Technical Debt and Future Work

- **Advanced Inference**: The current type inference is relatively simple. More complex scenarios involving nested generics or circular constraints may require a more robust constraint-solver-based approach (like Hindley-Milner).
- **Method Resolution**: Method lookup currently relies on a simple name-based search in the `TypeInfo`. Support for method overloading or more complex trait-based dispatch might be needed in the future.
- **Performance**: For very large projects, the recursive module checking and deep AST traversal could become a bottleneck. Implementing incremental checking or a more parallelized approach could improve performance.
- **Circular Type Definitions**: While module cycles are detected, circular type definitions (e.g., two structs containing each other directly) need careful handling to avoid infinite recursion during size calculation or equality checks.
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/lir/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_4d732416f503b1eb]
File: /Users/bill/projects/forge/pkg/lir/design.md (93 lines)

# LIR Module Design

## Executive Summary

The `lir` (Low-level Intermediate Representation) module serves as the bridge between the high-level, type-checked Abstract Syntax Tree (AST) and the backend-specific code generators in the Forge compiler. Its primary purpose is to resolve all semantic complexity of the Forge language into a simplified, flat representation that makes backend implementation straightforward.

The LIR is designed around several key principles:
- **Structured Control Flow**: Unlike many IRs that use basic blocks and jumps, Forge LIR preserves structured control flow (if, while, for, switch). This simplifies the generation of high-level target languages like Go and C.
- **Flat Expressions**: All nested expressions are flattened into a sequence of single-operation assignments to named temporaries (SSA-like). This eliminates the need for backends to handle complex expression trees.
- **Semantic Resolution**: All high-level language features—such as optional types, tagged unions (enums), method dispatch, and closures—are lowered into explicit LIR constructs.
- **Backend Agnostic**: The LIR provides a common foundation for multiple backends. It preserves type information (including generics for the Go backend) while providing a monomorphization pass for backends that require it (like C).

## File Inventory

- [lir.go](lir.go): Defines the core data structures for the LIR, including types (`LType`), operands (`LValue`), expressions (`LExpr`), and statements (`LStmt`).
- [lower.go](lower.go): Implements the `Lowerer`, which transforms a type-checked AST into an LIR program. This is where most of the semantic lowering logic resides.
- [optimize.go](optimize.go): Performs post-lowering LIR→LIR optimizations, such as side-effect temp elimination, multi-return destructuring, and unused variable removal.
- [monomorphize.go](monomorphize.go): Implements a monomorphization pass that specializes generic functions, classes, and structs into concrete versions for backends that do not support generics.
- [go_backend.go](go_backend.go): The Go code generator. It transforms LIR into idiomatic Go, handling Forge-specific features like generators and enums using Go-native constructs.
- [c_backend.go](c_backend.go): The C code generator. It transforms monomorphized LIR into C code, utilizing a small runtime library for features like slices and strings.
- [lower_test.go](lower_test.go): Unit tests for the lowering pass.
- [mono_test.go](mono_test.go): Unit tests for the monomorphization pass.
- [e2e_test.go](e2e_test.go): End-to-end tests that lower, optimize, and emit code to verify the entire pipeline.

## Architecture and Data Flow

The `lir` module operates as a pipeline that transforms high-level Forge code into target source code. The flow of data through the module follows these stages:

1.  **Lowering**: The `Lowerer` in [lower.go](lower.go) takes a type-checked `ast.File` (from [pkg/ast](../ast/design.md)) and a `checker.Registry` (from [pkg/checker](../checker/design.md)). It performs a recursive walk of the AST, flattening expressions into temporaries and converting high-level constructs into their LIR equivalents. For example, a `match` statement on an enum is lowered into a `switch` on the variant tag, with explicit field extractions.
2.  **Optimization**: The `Optimize` function in [optimize.go](optimize.go) runs several passes over the generated LIR. These passes clean up the output of the lowerer, which often produces redundant temporaries or patterns that are difficult for backends to emit efficiently. Key optimizations include fusing void-context calls into bare statements and destructuring multi-return values.
3.  **Monomorphization (Optional)**: For backends like C, the `Monomorphize` function in [monomorphize.go](monomorphize.go) is called. It scans the program for all instantiations of generic types and functions, creating specialized, non-generic copies for each unique set of type arguments. It then rewrites all call sites to use these specialized versions.
4.  **Code Generation**: Finally, a backend-specific emitter ([go_backend.go](go_backend.go) or [c_backend.go](c_backend.go)) walks the LIR tree and produces the final source code. Because the LIR has already resolved all semantic complexity, these backends are primarily responsible for syntax mapping and utilizing their respective runtime libraries.

## Interface Implementations

The `lir` module does not explicitly implement interfaces defined in other packages, as it primarily serves as a data transformation layer. However, it consumes the interfaces and data structures defined in:
- [pkg/ast](../ast/design.md): The input to the lowering pass.
- [pkg/checker](../checker/design.md): Provides type information and the symbol registry used during lowering.

## Public API

The primary entry points for the `lir` module are:

- `NewLowerer()`: Creates a new instance of the AST-to-LIR transformer.
- `Lowerer.Lower(file *ast.File) *LProgram`: Transforms a type-checked AST file into an LIR program.
- `Optimize(prog *LProgram)`: Runs the optimization suite on an LIR program.
- `Monomorphize(prog *LProgram)`: Specializes generics for backends that require it.
- `EmitGo(prog *LProgram) string`: Generates Go source code from an LIR program.
- `EmitC(prog *LProgram) string`: Generates C source code from a monomorphized LIR program.


## Implementation Details

### Lowering Pass

The `Lowerer` is the most complex component of the module. It maintains a stateful mapping of AST nodes to LIR constructs. A critical aspect of lowering is **expression flattening**. Every sub-expression in the AST is assigned to a unique temporary variable (`LValTemp`). This ensures that all `LExpr` instances in the LIR reference only simple operands, never nested expressions.

Control flow lowering is **structured**. Instead of basic blocks and jumps, the LIR uses high-level statements like `LStmtIf` and `LStmtWhile`. This is particularly important for the Go backend, as it allows for the generation of idiomatic Go code. The `LStmtWhile` is unique in that it contains a `CondBlock`—a list of statements that are executed before each iteration to compute the loop condition. This elegantly handles complex conditions that might involve side effects or multiple steps.

The `Lowerer` also handles the desugaring of Forge's high-level types:
- **Enums**: Lowered to `LTyTaggedUnion`. Match statements are converted to switches on the variant tag.
- **Classes**: Lowered to `LTyClassHandle`, which represents a pointer to a heap-allocated struct. Field access is converted to explicit `LExprClassGet` and `LStmtClassSet` operations.
- **Generics**: The `Lowerer` preserves type parameters as `LTyTypeVar`. This allows the Go backend to emit native Go generics.
- **Impl Blocks**: The lowerer generates forwarding wrapper methods for `impl` blocks, enabling classes to satisfy interfaces even when method names or signatures don't match exactly.

### Optimization Pass

The optimizer runs after lowering to clean up the LIR. One of its most important tasks is **side-effect temp elimination**. The lowerer often assigns the result of every call to a temporary, even if the result is immediately discarded. The optimizer fuses these into `LStmtSideEffect` statements.

Another critical optimization is **multi-return destructuring**. Forge allows functions to return multiple values (e.g., `(T, error)`). The lowerer initially represents this as a single temporary of type `LTyErrorResult`. The optimizer identifies patterns where this temporary is immediately destructured into separate variables and replaces them with a single `LStmtMultiAssign`, which maps directly to Go's multi-assignment syntax.

### Monomorphization

For the C backend, monomorphization is mandatory. The pass works in two phases. First, it performs a deep walk of all function bodies to collect every unique instantiation of generic functions, classes, and structs. Second, it generates specialized copies of these declarations, mangling their names (e.g., `Stack_i32`) to ensure uniqueness. Finally, it rewrites all call sites and type references to use these specialized versions.

### Backends

The **Go Backend** is designed to be a "thin" emitter. It leverages Go's native support for generics, interfaces, and concurrency. It uses a `strings.Builder` to accumulate the generated code and maintains a map of required imports, which are automatically added to the file header.

The **C Backend** is more involved due to C's lack of high-level features. It relies on a small runtime header (`forge_runtime.h`) that defines macros and structures for slices, optional types, and error results. It hoists lambdas to top-level static functions and uses `malloc` for class allocations.

## Dependencies

- [pkg/ast](../ast/design.md): Provides the source AST for lowering.
- [pkg/checker](../checker/design.md): Provides the type information and symbol registry required to resolve names and types during lowering.

## Technical Debt and Future Work

- **Map Support in C**: The C backend currently lacks support for map types.
- **Concurrency in C**: Goroutines and channels are not yet implemented in the C backend.
- **Advanced Optimizations**: Future work could include constant folding, dead code elimination beyond unused temporaries, and inlining.
- **Relational Constraints**: While basic support for relational constraints exists, more complex scenarios involving multi-class interfaces may require further refinement in the Go backend.
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/parser/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_ed6603baff815f3b]
File: /Users/bill/projects/forge/pkg/parser/design.md (82 lines)

# Parser Module Design

The `parser` module is responsible for transforming Forge source code into an Abstract Syntax Tree (AST). It implements a hand-written lexer and a hybrid parser that combines recursive descent for high-level structures and statements with Pratt parsing for expressions. The parser is designed to handle both the declarative `forge { ... }` blocks and the imperative logic found in function bodies.

## Executive Summary

The `parser` module serves as the entry point for the Forge compiler's frontend. It takes raw source text and produces a structured representation defined in the [pkg/ast](../ast/design.md) module. The module is split into a lexer that tokenizes the input and a parser that enforces the language's grammar. Key features include support for significant newlines, complex f-string interpolation via recursive parsing, and a robust expression engine that handles operator precedence and associativity.

## File Inventory

- [lexer.go](lexer.go): Implements a hand-written lexer that tokenizes Forge source code. It handles literals, keywords, punctuation, and significant newlines, while also collecting comments for inclusion in the AST.
- [parser.go](parser.go): Contains the primary recursive descent logic for parsing top-level declarations, `forge` blocks, and complex types. It manages the parser state and error reporting.
- [expr_parser.go](expr_parser.go): Implements expression parsing using the Pratt (precedence climbing) algorithm and statement parsing via recursive descent. It also handles specialized constructs like match expressions, lambdas, and f-string interpolation.
- [parser.forge](parser.forge): A Forge specification file that provides a high-level architectural overview and self-documentation of the parser module.

## Architecture and Data Flow

The parsing process follows a linear progression from raw text to a fully formed AST:

1.  **Lexical Analysis**: The `Lexer` reads the source string character by character, producing a stream of `Token` objects. It handles whitespace by skipping it except for newlines, which are preserved as `TNewline` tokens when they are syntactically significant. Comments are collected into a slice within the `Lexer` to be attached to the final `ast.File`.
2.  **Syntactic Analysis**: The `Parser` consumes tokens from the `Lexer`. It uses recursive descent to match high-level constructs like `class`, `interface`, and `func` declarations.
3.  **Expression Parsing**: When the parser encounters an expression, it hand-offs control to the Pratt parsing logic in `expr_parser.go`. This allows for efficient handling of operator precedence (e.g., multiplication before addition) and complex postfix operations like method calls and indexing.
4.  **AST Construction**: As the parser matches grammatical rules, it constructs the corresponding nodes from the `pkg/ast` module. These nodes are linked together to form a tree representing the entire source file.
5.  **Recursive Interpolation**: A unique aspect of the parser is its handling of f-strings. When an f-string is encountered, the lexer captures the raw content. The parser then identifies the `{expr}` boundaries and recursively invokes a new parser instance to process the embedded expressions, merging the results into a `StringInterpExpr`.

## Interface Implementations

The `parser` module does not implement external interfaces. Instead, it acts as a producer for the `ast.File` structure, which is then consumed by the [pkg/checker](../checker/design.md) and backend modules.

## Public API

### Entry Points

- **`ParseFile(source, filename string) (*ast.File, error)`**: The primary entry point for parsing a Forge source file. It initializes the lexer and parser and returns the resulting AST or a `ParseError`.
- **`ParseString(source string) (*ast.File, error)`**: A convenience wrapper around `ParseFile` that uses `"<string>"` as the filename.

### Types

- **`Parser`**: The core parsing engine. While it can be instantiated manually, it is typically used through the package-level `Parse` functions. It maintains a reference to the `Lexer` and a list of accumulated errors.
- **`ParseError`**: A specialized error type that includes a descriptive message and an `ast.Span` indicating the location of the error in the source code.

## Implementation Details

### Lexical Scanning

The lexer is a stateful scanner that uses a `peek`/`advance` pattern. It is responsible for:
- **Significant Newlines**: Newlines are treated as tokens to separate statements and declarations, but multiple consecutive newlines are collapsed to simplify parsing.
- **Comment Collection**: Line comments (`//`) are captured and stored in the `Lexer.Comments` slice, allowing the parser to attach them to the `ast.File` for use in documentation generation.
- **F-String Raw Capture**: Unlike standard strings, f-strings are scanned as a single token containing the raw text between the quotes. This allows the parser to handle the interpolation logic.

### Recursive Descent and Backtracking

The parser uses recursive descent for most of its grammar. In cases where the grammar is ambiguous, it employs limited backtracking by saving and restoring the lexer state:
- **Struct Literals vs. Blocks**: When the parser sees an identifier followed by `{`, it peeks ahead to see if the content looks like a field initializer (`name: value`) to distinguish a struct literal from a block of code.
- **Generic Calls vs. Comparisons**: The `<` operator can start a generic argument list or a comparison. The parser attempts to parse a type argument list and backtracks if it fails or if the following token is not an opening parenthesis.

### Pratt Parsing for Expressions

The expression parser uses a table of precedence levels to drive the Pratt algorithm. This ensures that operators are grouped correctly according to standard mathematical and logical rules. Postfix operators (like `.`, `()`, `[]`, `!`, and `?`) are handled in a loop after the primary expression is parsed, allowing for arbitrary chaining.

## Key Design Decisions

- **Hand-written Recursive Descent**: The choice of a hand-written parser over a generator (like Yacc or Antlr) provides full control over error messages and allows for easy integration of the Pratt algorithm for expressions. It also facilitates the state-saving lookahead mechanism used for disambiguation.
- **Pratt Parsing for Expressions**: Using the Pratt algorithm (precedence climbing) allows for a clean and efficient way to handle 12 levels of operator precedence without the "grammar explosion" typical of pure recursive descent.
- **Contextual Keywords**: To keep the language ergonomic, many keywords (like `why`, `doc`, `source`) are treated as "contextual." The `expectIdentLike` method allows these tokens to be used as identifiers in positions where they are not ambiguous (e.g., as field or parameter names).
- **Unified AST**: Both declarative `.forge` files and imperative `.fg` files share the same AST structures. A function in a `.forge` file simply has a `nil` body, while the same function in a `.fg` file includes a `Block` node.
- **State Restoration for Disambiguation**: The parser uses a simple but effective lookahead strategy by copying the `Lexer` struct (which is small and contains only primitive types and a string reference). This allows it to "try" parsing a construct (like a generic call) and backtrack if it fails, without complex state management.

### Statement Parsing

Statements are parsed using recursive descent. The parser recognizes keywords like `let`, `if`, `for`, `while`, `match`, `return`, `break`, `continue`, `spawn`, `select`, and `lock`. If no keyword is matched, it attempts to parse an expression statement or an assignment.

## Dependencies

- **[pkg/ast](../ast/design.md)**: The parser is entirely dependent on the `ast` module for its output data structures. It uses `ast.Pos` and `ast.Span` for location tracking and constructs various `ast.Node` types.

## Technical Debt and Future Work

- **Error Recovery**: Currently, the parser often stops at the first error. Implementing a synchronization mechanism (e.g., skipping to the next newline or brace) would allow it to report multiple errors in a single pass.
- **Performance**: The use of lexer state restoration for backtracking is functional but can be inefficient for deeply nested ambiguities. A more formal lookahead mechanism or a GLR approach might be considered if performance becomes a bottleneck.
- **Interpolation Complexity**: The recursive invocation of the parser for f-strings is powerful but creates overhead. Pre-tokenizing the interpolated parts during the initial lexing phase might be more efficient.
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/verifier/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_089db34959b10946]
File: /Users/bill/projects/forge/pkg/verifier/design.md (95 lines)

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

1.  **Forge Parsing**: The process begins by delegating to [pkg/parser](../parser/design.md) to produce an `ast.File` from the provided `.forge` source path.
2.  **Go Source Discovery**: The verifier inspects the `source:` annotations within each `forge` block to locate the relevant Go files. These paths are resolved relative to the directory containing the `.forge` file.
3.  **Go Type Extraction**: For each discovered source (which can be a file or a directory), the verifier uses `go/parser` and `go/ast` to extract type information. This data is populated into a `goTypeInfo` registry, which serves as a comprehensive database of all structs, interfaces, functions, and type definitions found across all specified sources for a given block.
4.  **Structural Comparison**: The verifier recursively walks the Forge AST, comparing each declaration (struct, class, interface, enum, function) against the aggregated Go types. It normalizes naming conventions and type expressions on the fly to enable direct comparison.
5.  **Completeness Analysis**: After validating the declared Forge symbols, the verifier performs a reverse check, walking all exported Go symbols in the source files to identify any that are missing from the Forge specification.
6.  **Result Accumulation**: Findings are collected into a `Result` object, which classifies each discrepancy by severity and provides detailed messages and file locations.

## Interface Implementations

The `verifier` module does not implement external interfaces defined in other packages. It is a standalone utility module that consumes the AST produced by [pkg/parser](../parser/design.md) and produces a `Result` report.

## Public API

### Primary Entry Point

- **`Verify(forgePath string) (*Result, error)`**: This is the "front door" to the module. It orchestrates the entire pipeline: parsing the Forge file, discovering Go sources, extracting types, and performing the comparison. It returns a `Result` containing all findings or an error if the verification process itself fails (e.g., file not found, parse error).

### Result and Findings

- **`Result`**: A container for the findings of a verification run. It provides an `ErrorCount()` method, which is typically used by CLI tools or CI pipelines to determine if the verification should be considered a failure.
- **`Finding`**: Represents a single discrepancy. It includes a `Severity` (Error, Warning, Info), the paths to the involved Forge and Go files, and a descriptive message.
- **`Severity`**: An enumeration used to classify findings:
    - **`Error`**: Fundamental mismatches such as missing types, incorrect field types, or parameter/return count mismatches.
    - **`Warning`**: Minor drift, such as extra Go fields not documented in Forge or constructor parameter mismatches.
    - **`Info`**: Contextual information, such as additional Go methods that are not part of the Forge interface specification.

## Implementation Details

### Deep Type Comparison

The core of the verifier is the `forgeTypeToGoString` function, which recursively converts Forge `TypeExpr` nodes into a Go-compatible string format. This allows the verifier to compare types across the two systems:

- **Optional Types**: `T?` in Forge is mapped to `*T` in Go.
- **Sequences**: `[T]` in Forge is mapped to `[]T` in Go.
- **Maps**: `map[K]V` maps directly to `map[K]V`.
- **Tuples**: Forge tuples are used to represent Go's multiple return values and are compared element-wise.
- **Equivalence**: The verifier recognizes that `any` and `interface{}` are equivalent in modern Go.

To handle cross-package references, the `stripPackagePrefix` function heuristically removes package qualifiers (e.g., `ast.File` becomes `File`). This allows Forge files to use unqualified names while the Go implementation uses fully qualified types.

### Name Mapping and Normalization

Forge typically uses `snake_case` for identifiers, while Go uses `PascalCase` for exported symbols and `camelCase` for unexported ones. The verifier bridges this gap by attempting three-way matching:
1.  **`snakeToPascal`**: (e.g., `foo_bar` → `FooBar`) for exported fields, methods, and types.
2.  **`snakeToCamel`**: (e.g., `foo_bar` → `fooBar`) for unexported identifiers.
3.  **Exact Match**: For cases where the names already align.

### Verification Rules

The verifier applies specific rules based on the type of declaration:

- **Structs and Classes**: Checks for field existence and type matching. For classes, it also verifies method signatures (parameter counts, positional types, and return types). It excludes the `self` parameter from Forge method signatures when comparing to Go methods.
- **Interfaces**: Ensures every method declared in the Forge interface exists in the Go interface with a matching signature.
- **Enums**: Validates that a matching Go type exists, supporting common Go patterns like `type X int` or `type XKind int`.
- **Functions**: Performs positional parameter and return type comparison.

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
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "runtime/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_8ca1c91ab4efdd5e]
File: /Users/bill/projects/forge/runtime/design.md (84 lines)

# Runtime Module Design

## Executive Summary

The `runtime` module provides the minimal C runtime library required by the Forge compiler's C backend. It is implemented as a header-only library ([forge_runtime.h](forge_runtime.h)) that provides the low-level primitives necessary to support Forge's high-level features in a C environment. This includes dynamic slices, optional types, error results, string manipulation, and tagged unions. By providing these abstractions, the runtime enables the C backend to emit code that is both readable and semantically equivalent to the original Forge source.

## File Inventory

- [forge_runtime.h](forge_runtime.h): The core runtime header. It contains all macro definitions and inline functions for the C-based Forge execution environment.

## Architecture and Data Flow

The `runtime` module is a passive component that provides a set of types and utilities used by the code generated by the [LIR module](../pkg/lir/design.md). It does not maintain internal state or have an active execution flow of its own. Instead, it defines the memory layout and operational semantics for Forge constructs when compiled to C.

The architecture is centered around C macros that allow for a form of "template-based" type generation. Since C lacks generics, the Forge C backend uses these macros to define concrete struct types for every unique instantiation of a generic Forge type (e.g., `Slice_int32_t`, `Opt_string`).

## Interface Implementations

As a C header library, this module does not implement any Go interfaces. It serves as a target for the C code generator in the [LIR module](../pkg/lir/design.md).

## Public API

The "Public API" of the runtime consists of the macros and functions intended to be called by Forge-generated C code.

### Dynamic Slices
Forge slices are implemented as dynamic arrays with automatic resizing.
- `FORGE_SLICE_DEF(ElemType, SliceName)`: Defines a struct for a slice of a specific type.
- `forge_slice_empty(SliceName)`: Initializes an empty slice.
- `forge_push(slice_ptr, elem, SliceName)`: Appends an element, growing the capacity by 2x if necessary.
- `forge_pop(slice_ptr)`: Removes and returns the last element.
- `forge_slice_lit(SliceName, ElemType, ...)`: Creates a slice from a C initializer list.
- `forge_contains(slice, elem)`: Performs a linear search for an element.

### Optional Types
Optionals (`T?`) are implemented as a struct containing a boolean flag and the value.
- `FORGE_OPT_DEF(ElemType, OptName)`: Defines an optional type.
- `forge_some(value, OptName)`: Creates an optional containing a value.
- `forge_none(OptName)`: Creates an empty optional.
- `forge_isnull(opt)`: Checks if the optional is empty.
- `forge_unwrap(opt)`: Returns the value (caller must ensure it is not null).

### Error Results
Forge's `(T, error)` return pattern is mapped to a result struct.
- `FORGE_RESULT_DEF(ElemType, ResultName)`: Defines a result type.
- `forge_ok(val, ResultName)`: Creates a successful result.
- `forge_err(msg, ResultName)`: Creates an error result with a message.
- `forge_is_err(r)`: Checks if the result represents an error.

### String Helpers
A collection of inline functions for high-level string operations.
- `forge_str_contains`, `forge_str_index_of`: Search operations.
- `forge_str_has_prefix`, `forge_str_has_suffix`: Prefix/suffix checks.
- `forge_str_replace`, `forge_str_repeat`, `forge_str_join`: String transformation and construction.
- `forge_sprintf`: A heap-allocating version of `sprintf`.

### Tagged Unions
Used for ad-hoc union types (e.g., `string | i32`).
- `ForgeUnion`: A generic tagged union struct.
- `forge_union_i32`, `forge_union_string`, etc.: Constructor functions for different types.

## Implementation Details

### Memory Management
The runtime currently relies on standard C `malloc` and `realloc`. A significant implementation detail is that the runtime does not yet include a garbage collector. Consequently, heap-allocated objects—including slices, strings created by helpers, and class instances—are currently leaked. This is a known limitation of the current C backend implementation.

### Slice Growth
The `forge_push` macro implements a simple geometric growth strategy. When the length reaches capacity, the capacity is doubled (starting from 4 if the slice was empty). This ensures amortized O(1) push operations.

### String Handling
Strings in the Forge C runtime are represented as standard null-terminated `const char*`. The helper functions are designed to mimic the behavior of Forge's built-in string methods, often allocating new memory for the result to ensure immutability-like behavior.

## Dependencies

- **Standard C Library**: The runtime depends on `<stdio.h>`, `<stdlib.h>`, `<stdint.h>`, `<stdbool.h>`, `<string.h>`, and `<stdarg.h>`.
- **LIR Module**: This module is a critical dependency for the C backend in [pkg/lir](../pkg/lir/design.md).

## Technical Debt and Future Work

- **Garbage Collection**: Implementing or integrating a garbage collector (e.g., Boehm-Demers-Weiser) is the highest priority for making the C backend viable for long-running programs.
- **Concurrency Support**: The C runtime currently lacks support for Forge's concurrency primitives (`spawn`, channels, `select`).
- **Map Implementation**: A hash map implementation is needed to support Forge's `map<K, V>` type in C.
- **Error Handling Refinement**: The current error result uses a `const char*` for the error message. This should be expanded to support more complex error types if Forge's error system evolves.
- **Thread Safety**: The current runtime primitives (especially slice operations) are not thread-safe.
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "content": "# Forge Project Architecture\n\n## Executive Summary\n\nThe Forge project is a comprehensive compiler and toolchain for the Forge programming language. It is designed to provide a high-level, expressive language with advanced features such as relations, interfaces, generic type inference, and structured control flow, while maintaining the capability to compile down to multiple backends, currently supporting Go and C. The project exists to bridge the critical gap between high-level architectural intent and low-level implementation. By capturing design specifications in dedicated `.forge` files, the system ensures structural integrity and actively prevents architectural drift. Its primary goals are to deliver a robust, multi-stage compilation pipeline—spanning parsing, semantic analysis, low-level intermediate representation, and code generation—while maintaining a strong, verifiable connection between the documented design and the evolving codebase through its unique verification engine.\n\n## System Architecture\n\nThe Forge compiler follows a classic pipeline architecture, augmented by a parallel verification track that acts as an architectural guardian. The architecture is fundamentally layered and data-centric, designed to isolate complexity at each distinct stage of transformation. The primary compilation flow is linear, progressing from raw source text to an Abstract Syntax Tree (AST), through semantic checking, into a Low-level Intermediate Representation (LIR), and finally to target code generation.\n\nThe philosophy of the system is to resolve specific classes of problems in dedicated modules. The parser handles all syntax and grammatical ambiguities, producing a structural representation. The AST module provides a clean, desugared representation of the program, expanding high-level syntactic sugar into fundamental constructs. The checker assumes responsibility for all semantic rules, performing type inference, scope management, and validation, ultimately annotating the AST with resolved types. The LIR module then takes this semantically rich tree and flattens it into a simplified, SSA-like sequence of operations with structured control flow. This crucial step removes the burden of semantic resolution from the backends. Finally, the backends act as thin emitters that translate the simplified LIR into idiomatic target source code.\n\nIn parallel to the compilation pipeline, the Verifier module operates as a standalone structural integrity engine. It parses `.forge` specification files and performs a deep structural comparison against the actual Go implementation. This ensures that the documented design and the codebase remain perfectly synchronized, enforcing the philosophy that documentation should be an active, verifiable contract rather than a passive artifact.\n\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/forge/project-design.md`
**Size**: 2815 bytes
**Permissions**: -rw-r--r--
**Lines**: 15
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "append": true,
  "content": "\n## Interface \u0026 Contract Map\n\nThe system's boundaries are primarily defined by the rich data structures passed between modules rather than traditional Go interfaces. These data structures act as the fundamental contracts that govern module interactions.\n\nThe most critical contract is the Abstract Syntax Tree. The `ast.File` and its constituent nodes representing declarations, expressions, and statements form the central nervous system of the compiler. The parser module acts as the initial producer of this contract, transforming source code into the raw AST. The AST module itself acts as both a consumer and producer, taking the raw AST and producing a desugared version. The checker module consumes this desugared AST for semantic analysis and mutates it by attaching resolved type information. The LIR module consumes the fully type-checked AST to generate the intermediate representation. Additionally, the verifier module consumes the AST of `.forge` files to compare against Go source code.\n\nThe Type Registry contract, defined by the `checker.Registry` and `checker.Type` structures, represents the semantic truth of the program. The checker module builds and populates this registry during its declaration and checking passes. The LIR module is the primary consumer of this contract, relying heavily on the type registry and resolved types to correctly lower high-level constructs, such as generics, classes, and enums, into low-level operations.\n\nThe LIR contract, represented by the `lir.LProgram`, defines the fully resolved and flattened program state. The LIR module's lowerer produces this representation. Internal optimization and monomorphization passes within the LIR module consume and mutate this structure. Ultimately, the Go and C backends consume the optimized `LProgram` to emit the final source code.\n\nThe Runtime contract is defined by the C runtime header, which establishes the execution environment for C-compiled Forge programs. The runtime module provides these foundational macros and functions. The C backend within the LIR module acts as the consumer, emitting code that relies entirely on this contract for memory layout and operational semantics.\n\n## Module Map\n\n### Frontend\n- [pkg/parser](pkg/parser/design.md)\n- [pkg/ast](pkg/ast/design.md)\n\n### Semantic Analysis\n- [pkg/checker](pkg/checker/design.md)\n\n### Intermediate Representation \u0026 Backends\n- [pkg/lir](pkg/lir/design.md)\n- [runtime](runtime/design.md)\n\n### Tooling\n- [pkg/verifier](pkg/verifier/design.md)\n\nThe **Parser** module is the entry point for the compiler frontend, responsible for transforming raw Forge source code into an Abstract Syntax Tree. It manages internal state through a hand-written lexer that tokenizes the input, carefully handling significant newlines and collecting comments. The parser employs recursive descent for high-level structures and a Pratt parser for expressions, allowing it to efficiently handle complex operator precedence. It implements the primary data contract by producing the `ast.File` consumed by the rest of the system.\n\nThe **AST** module defines the core data structures that represent the structure of Forge code, acting as the central data hub. It manages state primarily as a tree of polymorphic nodes utilizing a \"Kind + Data\" pattern to avoid deep interface hierarchies. Crucially, it includes a desugaring phase that transforms high-level constructs, such as relations and interface fields, into fundamental AST nodes before semantic analysis occurs. It implements the primary data contract consumed by the checker and LIR modules.\n\nThe **Checker** module implements the semantic analysis phase of the compiler. It is responsible for type checking, generic type inference, name resolution, and enforcing all language constraints. It maintains complex internal state, including a stack of scopes for variable tracking and a global registry for user-defined types. It consumes the desugared AST, resolves all type expressions, and annotates the AST nodes with `ResolvedType` information, fulfilling the semantic contract required by the subsequent lowering phase.\n\nThe **LIR** module bridges the high-level AST and the backend code generators by resolving all semantic complexity into a simplified, flat representation with structured control flow. It manages state during the lowering process by mapping AST nodes to LIR constructs and assigning all sub-expressions to unique temporary variables. It also includes sophisticated optimization and monomorphization passes. It consumes the type-checked AST and produces the `LProgram` contract, which is then consumed by its internal Go and C backends to emit target code.\n\nThe **Verifier** module is a standalone structural integrity engine designed to detect architectural drift. It compares high-level `.forge` specifications against the actual Go implementation. It maintains state during a run to aggregate type information extracted from Go source files using the standard library's AST tools. It consumes the AST produced by the parser for `.forge` files and performs deep structural comparisons, reporting discrepancies to ensure the specification remains a complete and accurate model of the codebase.\n\nThe **Runtime** module provides the minimal C runtime library required by the Forge C backend. It is a passive, header-only library that defines the memory layout and operational semantics for Forge constructs like dynamic slices, optional types, and error results in a C environment. It does not maintain internal state but provides the foundational macros and functions consumed by the C code generated by the LIR module.\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/forge/project-design.md`
**Size**: 8452 bytes
**Permissions**: -rw-r--r--
**Lines**: 41
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "append": true,
  "content": "\n## Integration Patterns \u0026 Workflows\n\nThe system relies on clear, sequential workflows to transform data across module boundaries. Two primary workflows define the operation of the Forge toolchain: the compilation pipeline and the architectural verification process.\n\nThe compilation pipeline workflow traces the journey of a Forge source file from raw text to executable code. The process begins when a file is read from disk and passed to the parser module. The lexer tokenizes the text, and the parser constructs an initial `ast.File`. This AST is immediately passed to the AST module's desugaring functions, which expand complex features like relations into standard classes and implementation blocks. The desugared AST is then handed to the checker module. The checker performs a declaration pass to populate its type registry, followed by a deep traversal to resolve types, infer generics, and validate semantics, annotating the AST with `ResolvedType` fields. Once validated, the AST and the type registry are passed to the LIR module's lowerer. The lowerer flattens expressions into temporaries and converts high-level constructs into LIR statements. The resulting `LProgram` undergoes optimization passes to remove redundant operations. If the target is C, a monomorphization pass specializes generic types. Finally, the backend emitter traverses the LIR and generates the final source code string, which is written to disk.\n\nThe architectural verification workflow operates independently to ensure design fidelity. The process starts when the verifier module is invoked on a `.forge` specification file. It uses the parser module to generate an AST of the specification. The verifier then inspects the source annotations within the AST to locate the corresponding Go source files. It uses the Go standard library to extract all exported types, interfaces, and function signatures from these Go files, building a comprehensive internal registry. The verifier then recursively walks the Forge AST, translating Forge type expressions into Go string representations and comparing them against the extracted Go types. It normalizes naming conventions on the fly to facilitate this comparison. Finally, it performs a reverse completeness check, ensuring every exported Go symbol is documented in the Forge specification, and aggregates any mismatches into a detailed report.\n\n## Dependency Overview\n\nThe project exhibits a strict, unidirectional dependency graph that prevents circular imports and enforces a clear separation of concerns. This layering is critical for maintaining the maintainability of the compiler.\n\nAt the base of the hierarchy is the AST module, which acts as a leaf node containing pure data structures. The parser module depends solely on the AST module to construct its output, ensuring that parsing logic is decoupled from semantic analysis. The checker module depends on both the AST module for input and annotation, and the parser module to handle imported files during its semantic analysis phase. The LIR module depends on the AST and checker modules to perform its lowering process, relying on the semantic truth established by the checker. \n\nThe verifier module depends on the AST and parser modules to read specifications, but operates entirely independently of the compiler's semantic and backend phases. This isolation allows the verifier to act as an objective observer of the codebase. The runtime module is completely independent of all Go code, serving only as a target dependency for the generated C code. This layered architecture ensures that the frontend remains decoupled from backend concerns, and that data structures are defined independently of the logic that manipulates them.\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/forge/project-design.md`
**Size**: 12181 bytes
**Permissions**: -rw-r--r--
**Lines**: 17
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_d9e61c511929a0d9]
File: /Users/bill/projects/forge/project-design.md (71 lines)

# Forge Project Architecture

## Executive Summary

The Forge project is a comprehensive compiler and toolchain for the Forge programming language. It is designed to provide a high-level, expressive language with advanced features such as relations, interfaces, generic type inference, and structured control flow, while maintaining the capability to compile down to multiple backends, currently supporting Go and C. The project exists to bridge the critical gap between high-level architectural intent and low-level implementation. By capturing design specifications in dedicated `.forge` files, the system ensures structural integrity and actively prevents architectural drift. Its primary goals are to deliver a robust, multi-stage compilation pipeline—spanning parsing, semantic analysis, low-level intermediate representation, and code generation—while maintaining a strong, verifiable connection between the documented design and the evolving codebase through its unique verification engine.

## System Architecture

The Forge compiler follows a classic pipeline architecture, augmented by a parallel verification track that acts as an architectural guardian. The architecture is fundamentally layered and data-centric, designed to isolate complexity at each distinct stage of transformation. The primary compilation flow is linear, progressing from raw source text to an Abstract Syntax Tree (AST), through semantic checking, into a Low-level Intermediate Representation (LIR), and finally to target code generation.

The philosophy of the system is to resolve specific classes of problems in dedicated modules. The parser handles all syntax and grammatical ambiguities, producing a structural representation. The AST module provides a clean, desugared representation of the program, expanding high-level syntactic sugar into fundamental constructs. The checker assumes responsibility for all semantic rules, performing type inference, scope management, and validation, ultimately annotating the AST with resolved types. The LIR module then takes this semantically rich tree and flattens it into a simplified, SSA-like sequence of operations with structured control flow. This crucial step removes the burden of semantic resolution from the backends. Finally, the backends act as thin emitters that translate the simplified LIR into idiomatic target source code.

In parallel to the compilation pipeline, the Verifier module operates as a standalone structural integrity engine. It parses `.forge` specification files and performs a deep structural comparison against the actual Go implementation. This ensures that the documented design and the codebase remain perfectly synchronized, enforcing the philosophy that documentation should be an active, verifiable contract rather than a passive artifact.


## Interface & Contract Map

The system's boundaries are primarily defined by the rich data structures passed between modules rather than traditional Go interfaces. These data structures act as the fundamental contracts that govern module interactions.

The most critical contract is the Abstract Syntax Tree. The `ast.File` and its constituent nodes representing declarations, expressions, and statements form the central nervous system of the compiler. The parser module acts as the initial producer of this contract, transforming source code into the raw AST. The AST module itself acts as both a consumer and producer, taking the raw AST and producing a desugared version. The checker module consumes this desugared AST for semantic analysis and mutates it by attaching resolved type information. The LIR module consumes the fully type-checked AST to generate the intermediate representation. Additionally, the verifier module consumes the AST of `.forge` files to compare against Go source code.

The Type Registry contract, defined by the `checker.Registry` and `checker.Type` structures, represents the semantic truth of the program. The checker module builds and populates this registry during its declaration and checking passes. The LIR module is the primary consumer of this contract, relying heavily on the type registry and resolved types to correctly lower high-level constructs, such as generics, classes, and enums, into low-level operations.

The LIR contract, represented by the `lir.LProgram`, defines the fully resolved and flattened program state. The LIR module's lowerer produces this representation. Internal optimization and monomorphization passes within the LIR module consume and mutate this structure. Ultimately, the Go and C backends consume the optimized `LProgram` to emit the final source code.

The Runtime contract is defined by the C runtime header, which establishes the execution environment for C-compiled Forge programs. The runtime module provides these foundational macros and functions. The C backend within the LIR module acts as the consumer, emitting code that relies entirely on this contract for memory layout and operational semantics.

## Module Map

### Frontend
- [pkg/parser](pkg/parser/design.md)
- [pkg/ast](pkg/ast/design.md)

### Semantic Analysis
- [pkg/checker](pkg/checker/design.md)

### Intermediate Representation & Backends
- [pkg/lir](pkg/lir/design.md)
- [runtime](runtime/design.md)

### Tooling
- [pkg/verifier](pkg/verifier/design.md)

The **Parser** module is the entry point for the compiler frontend, responsible for transforming raw Forge source code into an Abstract Syntax Tree. It manages internal state through a hand-written lexer that tokenizes the input, carefully handling significant newlines and collecting comments. The parser employs recursive descent for high-level structures and a Pratt parser for expressions, allowing it to efficiently handle complex operator precedence. It implements the primary data contract by producing the `ast.File` consumed by the rest of the system.

The **AST** module defines the core data structures that represent the structure of Forge code, acting as the central data hub. It manages state primarily as a tree of polymorphic nodes utilizing a "Kind + Data" pattern to avoid deep interface hierarchies. Crucially, it includes a desugaring phase that transforms high-level constructs, such as relations and interface fields, into fundamental AST nodes before semantic analysis occurs. It implements the primary data contract consumed by the checker and LIR modules.

The **Checker** module implements the semantic analysis phase of the compiler. It is responsible for type checking, generic type inference, name resolution, and enforcing all language constraints. It maintains complex internal state, including a stack of scopes for variable tracking and a global registry for user-defined types. It consumes the desugared AST, resolves all type expressions, and annotates the AST nodes with `ResolvedType` information, fulfilling the semantic contract required by the subsequent lowering phase.

The **LIR** module bridges the high-level AST and the backend code generators by resolving all semantic complexity into a simplified, flat representation with structured control flow. It manages state during the lowering process by mapping AST nodes to LIR constructs and assigning all sub-expressions to unique temporary variables. It also includes sophisticated optimization and monomorphization passes. It consumes the type-checked AST and produces the `LProgram` contract, which is then consumed by its internal Go and C backends to emit target code.

The **Verifier** module is a standalone structural integrity engine designed to detect architectural drift. It compares high-level `.forge` specifications against the actual Go implementation. It maintains state during a run to aggregate type information extracted from Go source files using the standard library's AST tools. It consumes the AST produced by the parser for `.forge` files and performs deep structural comparisons, reporting discrepancies to ensure the specification remains a complete and accurate model of the codebase.

The **Runtime** module provides the minimal C runtime library required by the Forge C backend. It is a passive, header-only library that defines the memory layout and operational semantics for Forge constructs like dynamic slices, optional types, and error results in a C environment. It does not maintain internal state but provides the foundational macros and functions consumed by the C code generated by the LIR module.

## Integration Patterns & Workflows

The system relies on clear, sequential workflows to transform data across module boundaries. Two primary workflows define the operation of the Forge toolchain: the compilation pipeline and the architectural verification process.

The compilation pipeline workflow traces the journey of a Forge source file from raw text to executable code. The process begins when a file is read from disk and passed to the parser module. The lexer tokenizes the text, and the parser constructs an initial `ast.File`. This AST is immediately passed to the AST module's desugaring functions, which expand complex features like relations into standard classes and implementation blocks. The desugared AST is then handed to the checker module. The checker performs a declaration pass to populate its type registry, followed by a deep traversal to resolve types, infer generics, and validate semantics, annotating the AST with `ResolvedType` fields. Once validated, the AST and the type registry are passed to the LIR module's lowerer. The lowerer flattens expressions into temporaries and converts high-level constructs into LIR statements. The resulting `LProgram` undergoes optimization passes to remove redundant operations. If the target is C, a monomorphization pass specializes generic types. Finally, the backend emitter traverses the LIR and generates the final source code string, which is written to disk.

The architectural verification workflow operates independently to ensure design fidelity. The process starts when the verifier module is invoked on a `.forge` specification file. It uses the parser module to generate an AST of the specification. The verifier then inspects the source annotations within the AST to locate the corresponding Go source files. It uses the Go standard library to extract all exported types, interfaces, and function signatures from these Go files, building a comprehensive internal registry. The verifier then recursively walks the Forge AST, translating Forge type expressions into Go string representations and comparing them against the extracted Go types. It normalizes naming conventions on the fly to facilitate this comparison. Finally, it performs a reverse completeness check, ensuring every exported Go symbol is documented in the Forge specification, and aggregates any mismatches into a detailed report.

## Dependency Overview

The project exhibits a strict, unidirectional dependency graph that prevents circular imports and enforces a clear separation of concerns. This layering is critical for maintaining the maintainability of the compiler.

At the base of the hierarchy is the AST module, which acts as a leaf node containing pure data structures. The parser module depends solely on the AST module to construct its output, ensuring that parsing logic is decoupled from semantic analysis. The checker module depends on both the AST module for input and annotation, and the parser module to handle imported files during its semantic analysis phase. The LIR module depends on the AST and checker modules to perform its lowering process, relying on the semantic truth established by the checker. 

The verifier module depends on the AST and parser modules to read specifications, but operates entirely independently of the compiler's semantic and backend phases. This isolation allows the verifier to act as an objective observer of the codebase. The runtime module is completely independent of all Go code, serving only as a target dependency for the generated C code. This layered architecture ensures that the frontend remains decoupled from backend concerns, and that data structures are defined independently of the logic that manipulates them.
```

---

### ASSISTANT (gemini)

Project documentation complete. I have successfully synthesized the `project-design.md` file at the root of the project. 

The document includes:
1. **Executive Summary**: An overview of the Forge compiler and its goals.
2. **System Architecture**: A deep narrative of the pipeline architecture and the parallel verification track.
3. **Interface & Contract Map**: A detailed explanation of the primary data structures (AST, Type Registry, LIR, Runtime) that act as contracts between modules.
4. **Module Map**: A categorized tree view with clickable links to each module's `design.md`, followed by a multi-paragraph summary of each module's role and state management.
5. **Integration Patterns & Workflows**: A detailed trace of the compilation pipeline and the architectural verification process.
6. **Dependency Overview**: A prose description of the strict, unidirectional dependency graph that enforces separation of concerns. 

The entire document has been written in continuous, flowing prose to ensure an optimal Text-to-Speech experience, avoiding bulleted lists for technical descriptions.

---

