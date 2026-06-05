# LIR Module Design

## Executive Summary

The `lir` (Low-level Intermediate Representation) module serves as the critical bridge between the high-level, semantically rich Abstract Syntax Tree (AST) and the backend-specific code generators in the Forge compiler. Its primary mission is to resolve all semantic complexity of the Forge language into a simplified, flat representation that makes backend implementation a straightforward exercise in syntax mapping. 

The LIR is built on several foundational design principles that distinguish it from traditional compiler IRs. First, it preserves **structured control flow** (if, while, for, switch) rather than decomposing logic into basic blocks and jumps. This preservation is essential for generating idiomatic high-level target code, particularly for the Go backend. Second, it enforces **flat expressions**, where every sub-expression is assigned to a unique, named temporary variable. This SSA-like flattening ensures that backends never have to deal with nested expression trees. Third, it performs **semantic resolution**, lowering high-level Forge features like optional types, tagged unions (enums), method dispatch, and closures into explicit LIR constructs. Finally, the LIR is **backend agnostic**, providing a common foundation for multiple targets while offering optional passes like monomorphization for backends that require it.

## File Inventory

*   [lir.go](lir.go): Defines the core data structures for the LIR, including the type system (`LType`), operands (`LValue`), expressions (`LExpr`), and structured statements (`LStmt`).
*   [lower.go](lower.go): Implements the `Lowerer`, the primary engine that transforms a type-checked AST into an LIR program. This file contains the bulk of the semantic lowering logic.
*   [optimize.go](optimize.go): Performs post-lowering LIR-to-LIR optimizations, such as side-effect temporary elimination, multi-return destructuring, and unused variable removal.
*   [monomorphize.go](monomorphize.go): Implements a monomorphization pass that specializes generic functions, classes, and structs into concrete versions for backends like C that do not support generics natively.
*   [go_backend.go](go_backend.go): The Go code generator, which transforms LIR into idiomatic Go code, leveraging Go's native support for generics, interfaces, and concurrency.
*   [c_backend.go](c_backend.go): The C code generator, which transforms monomorphized LIR into C code, utilizing a dedicated runtime library for features like slices and strings.
*   [lir.forge](lir.forge): A Forge specification file that formally defines the LIR module's structure and interfaces, used by the verifier to ensure implementation integrity.
*   [lower_test.go](lower_test.go): Comprehensive unit tests for the lowering pass, verifying the transformation of various AST constructs.
*   [mono_test.go](mono_test.go): Unit tests for the monomorphization pass, ensuring correct specialization and name mangling.
*   [c_backend_test.go](c_backend_test.go): Tests for the C code generator, verifying the emission of valid C code.
*   [e2e_test.go](e2e_test.go): End-to-end tests that exercise the entire pipeline from lowering through optimization and code generation.
*   [dump_c_test.go](dump_c_test.go): A utility test for dumping generated C code for manual inspection.

## Architecture and Data Flow

The `lir` module operates as a sophisticated transformation pipeline that converts high-level architectural intent into executable source code. The flow of data through the module follows a strictly defined sequence of stages.

The pipeline begins with the **Lowering** stage. The `Lowerer` consumes a type-checked `ast.File` from the [pkg/ast](../ast/design.md) module and a `checker.Registry` from the [pkg/checker](../checker/design.md) module. It performs a recursive walk of the AST, flattening every nested expression into a sequence of single-operation assignments to named temporaries. During this process, high-level constructs are desugared into their LIR equivalents; for instance, a `match` statement on an enum is lowered into a `switch` on an integer variant tag, with explicit field extractions for any bound variables.

Once the initial LIR is generated, it enters the **Optimization** stage. The `Optimize` function runs several passes over the LIR to clean up the output of the lowerer. These passes are essential because the lowerer often produces redundant temporaries or patterns that are difficult for backends to emit efficiently. Key optimizations include fusing void-context calls into bare statements (side-effect elimination) and destructuring multi-return values from single temporaries into separate variables.

For backends that require it, such as C, the pipeline includes an optional **Monomorphization** stage. The `Monomorphize` function scans the LIR program for all instantiations of generic types and functions. It then creates specialized, non-generic copies for each unique set of type arguments and rewrites all call sites and type references to use these specialized versions. This ensures that the final LIR contains no type variables before it reaches the backend.

The final stage is **Code Generation**. A backend-specific emitter, either the Go backend or the C backend, walks the optimized (and potentially monomorphized) LIR tree and produces the final source code string. Because the LIR has already resolved all semantic complexity, these backends are primarily responsible for syntax mapping and utilizing their respective runtime libraries.

## Interface Implementations

The `lir` module does not explicitly implement interfaces defined in other packages, as its primary role is data transformation. However, it is a heavy consumer of the data structures and semantic information provided by:
*   [pkg/ast](../ast/design.md): The module provides the input `ast.File` and all its constituent nodes.
*   [pkg/checker](../checker/design.md): The module provides the `checker.Registry` and the `checker.Type` information used to resolve names and types during the lowering process.

## Public API

The `lir` module exposes a clean, functional API for interacting with the LIR pipeline. The primary entry points are:

*   `NewLowerer()`: This function creates a new instance of the `Lowerer`, initializing the internal state required for AST-to-LIR transformation.
*   `Lowerer.Lower(file *ast.File) *LProgram`: This is the "front door" to the module. It takes a type-checked AST file and returns a complete `LProgram` structure representing the lowered code.
*   `Optimize(prog *LProgram)`: This function runs the full suite of LIR-to-LIR optimization passes on a program, mutating it in place to improve code quality and backend compatibility.
*   `Monomorphize(prog *LProgram)`: This function specializes all generic declarations in the program, removing all type variables. It is a prerequisite for the C backend.
*   `EmitGo(prog *LProgram) string`: This function takes an LIR program and generates a complete Go source code string, handling all necessary imports and formatting.
*   `EmitC(prog *LProgram) string`: This function takes a monomorphized LIR program and generates a complete C source code string, relying on the `forge_runtime.h` contract.

## Implementation Details

### The Lowering Pass

The `Lowerer` is the most complex component of the module, responsible for the heavy lifting of semantic resolution. It maintains a stateful mapping of AST nodes to LIR constructs and tracks visibility, imports, and type information. A central feature of the lowerer is **expression flattening**. Every sub-expression in the AST is assigned to a unique temporary variable (`LValTemp`). This ensures that all `LExpr` instances in the LIR reference only simple operands, never nested expressions. For example, `a + b * c` is lowered into `%0 = b * c` followed by `%1 = a + %0`.

Control flow lowering in Forge LIR is **structured**. Instead of decomposing logic into basic blocks and jumps, the LIR uses high-level statements like `LStmtIf` and `LStmtWhile`. The `LStmtWhile` is particularly noteworthy; it contains a `CondBlock`—a list of statements executed before each iteration to compute the loop condition. This elegantly handles complex conditions that might involve side effects or multiple steps, ensuring they are re-evaluated correctly on every iteration without duplicating code.

The `Lowerer` also handles the desugaring of Forge's high-level types:
*   **Enums**: These are lowered to `LTyTaggedUnion`. Match statements are converted to switches on the variant tag, with the lowerer generating the necessary logic to extract variant fields into local bindings.
*   **Classes**: These are lowered to `LTyClassHandle`, representing a pointer to a heap-allocated struct. Field access is converted to explicit `LExprClassGet` and `LStmtClassSet` operations.
*   **Generics**: The lowerer preserves type parameters as `LTyTypeVar`. This allows the Go backend to emit native Go generics.
*   **Impl Blocks**: The lowerer generates forwarding wrapper methods for `impl` blocks. This enables classes to satisfy interfaces even when method names or signatures do not match exactly, by creating a bridge between the interface's expected signature and the class's actual implementation.
*   **Try Operator**: The `?` operator is lowered into a sequence of operations: extracting the error from a result, checking if it is non-null, and performing an early return if an error is present, followed by extracting the success value.

### The Optimization Pass

The optimizer runs after lowering to refine the LIR. One of its most critical tasks is **side-effect temporary elimination**. The lowerer often assigns the result of every call to a temporary, even if the result is immediately discarded (e.g., in a void-context call). The optimizer identifies these patterns and fuses them into `LStmtSideEffect` statements, which backends can emit as bare expressions.

Another vital optimization is **multi-return destructuring**. Forge allows functions to return multiple values, such as `(T, error)`. The lowerer initially represents this as a single temporary of type `LTyErrorResult`. The optimizer identifies patterns where this temporary is immediately destructured into separate variables and replaces them with a single `LStmtMultiAssign`. This maps directly to Go's multi-assignment syntax and simplifies C's handling of result structures.

### Monomorphization

For backends like C that lack native generic support, monomorphization is mandatory. The pass operates in four distinct phases. First, it indexes all generic declarations in the program. Second, it performs a deep walk of all function bodies to collect every unique instantiation of generic functions, classes, and structs based on the concrete type arguments used at call sites. Third, it generates specialized copies of these declarations, mangling their names (e.g., `Stack_i32`) to ensure uniqueness. Finally, it rewrites all call sites and type references throughout the program to use these specialized, non-generic versions.

### Backends

The **Go Backend** is designed as a "thin" emitter. It leverages Go's native support for generics, interfaces, and concurrency. It uses a `strings.Builder` to accumulate code and maintains a map of required imports, which are automatically added to the file header. It handles Forge-specific features like generators by emitting a goroutine and channel pattern that matches Forge's lazy iteration semantics.

The **C Backend** is significantly more involved due to C's lack of high-level features. It relies on a small runtime header (`forge_runtime.h`) that defines macros and structures for slices, optional types, and error results. It hoists lambdas to top-level static functions and uses `malloc` for class allocations. It also implements a sophisticated type inference mechanism to resolve `LTyAny` types into concrete C types where possible, ensuring type safety in the generated code.

## Dependencies

*   [pkg/ast](../ast/design.md): Provides the source AST and node definitions.
*   [pkg/checker](../checker/design.md): Provides the type registry and resolved type information.
*   [runtime](../../runtime/design.md): Provides the C runtime contract for the C backend.

## Technical Debt and Future Work

*   **Map Support in C**: The C backend currently lacks implementation for map types.
*   **Concurrency in C**: Goroutines and channels are not yet supported in the C backend.
*   **Advanced Optimizations**: Future improvements could include constant folding, dead code elimination beyond unused temporaries, and function inlining.
*   **Relational Constraints**: While basic support exists, more complex scenarios involving multi-class interfaces may require further refinement in the Go backend to ensure perfect compatibility with Go's type system.
