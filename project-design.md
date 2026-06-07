# Forge Project Architecture

## Executive Summary

The Forge project is a comprehensive compiler and toolchain for the Forge programming language. It is designed to provide a high-level, expressive language with advanced features such as relations, interfaces, generic type inference, and structured control flow, while maintaining the capability to compile down to multiple backends, currently supporting Go and C. The project exists to bridge the critical gap between high-level architectural intent and low-level implementation. By capturing design specifications in dedicated `.forge` files, the system ensures structural integrity and actively prevents architectural drift. Its primary goals are to deliver a robust, multi-stage compilation pipeline—spanning parsing, semantic analysis, low-level intermediate representation, and code generation—while maintaining a strong, verifiable connection between the documented design and the evolving codebase through its unique verification engine.

## System Architecture

The Forge compiler follows a classic pipeline architecture, augmented by a parallel verification track that acts as an architectural guardian. The architecture is fundamentally layered and data-centric, designed to isolate complexity at each distinct stage of transformation. The primary compilation flow is linear, progressing from raw source text to an Abstract Syntax Tree (AST), through semantic checking, into a Low-level Intermediate Representation (LIR), and finally to target code generation.

The philosophy of the system is to resolve specific classes of problems in dedicated modules. The parser handles all syntax and grammatical ambiguities, employing a hybrid strategy of recursive descent and Pratt parsing to produce a structural representation. The AST module acts as the central data hub, providing a clean representation of the program using a highly efficient "Kind + Data" pattern. Crucially, it features a generative desugaring engine that expands high-level syntactic sugar—such as relations and interface fields—into fundamental constructs before semantic analysis. 

The checker assumes responsibility for all semantic rules, acting as the semantic truth of the compiler. It performs type inference, scope management, module import resolution, and validation, ultimately annotating the AST with resolved types. The LIR module then takes this semantically rich tree and bridges the gap to the backends. It flattens the tree into a simplified sequence of operations while preserving structured control flow. This crucial step removes the burden of semantic resolution from the backends, handling complex transformations like monomorphization for targets that lack native generics. Finally, the backends act as thin emitters that translate the simplified LIR into idiomatic target source code.

In parallel to the compilation pipeline, the verifier module operates as a standalone structural integrity engine. It parses `.forge` specification files and performs a deep structural comparison against the actual Go implementation, utilizing the Go standard library's AST tools to extract type information. This ensures that the documented design and the codebase remain perfectly synchronized, enforcing the philosophy that documentation should be an active, verifiable contract rather than a passive artifact.

## Interface & Contract Map

The system's boundaries are primarily defined by the rich data structures passed between modules rather than traditional Go interfaces. These data structures act as the fundamental contracts that govern module interactions.

The most critical contract is the Abstract Syntax Tree. The `ast.File` and its constituent nodes representing declarations, expressions, and statements form the central nervous system of the compiler. The parser module acts as the initial producer of this contract, transforming source code into the raw AST. The AST module itself acts as both a consumer and producer, taking the raw AST and producing a desugared version. The checker module consumes this desugared AST for semantic analysis and mutates it by attaching resolved type information. The LIR module consumes the fully type-checked AST to generate the intermediate representation. Additionally, the verifier module consumes the AST of `.forge` files to compare against Go source code.

The Type Registry contract, defined by the `checker.Registry` and `checker.Type` structures, represents the semantic truth of the program. The checker module builds and populates this registry during its declaration and checking passes. The LIR module is the primary consumer of this contract, relying heavily on the type registry and resolved types to correctly lower high-level constructs, such as generics, classes, and enums, into low-level operations.

The LIR contract, represented by the `lir.LProgram`, defines the fully resolved and flattened program state. The LIR module's lowerer produces this representation. Internal optimization and monomorphization passes within the LIR module consume and mutate this structure. Ultimately, the Go and C backends consume the optimized `LProgram` to emit the final source code.

The Runtime contract is defined by the C runtime header (`forge_runtime.h`), which establishes the execution environment for C-compiled Forge programs. The runtime module provides these foundational macros and inline functions. The C backend within the LIR module acts as the consumer, emitting code that relies entirely on this contract for memory layout and operational semantics, such as dynamic slices, optional types, and error results.

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

The **Parser** module is the foundational entry point for the compiler frontend, responsible for transforming raw Forge source code into an Abstract Syntax Tree. It manages internal state through a hand-written lexical scanner that tokenizes the input, carefully handling significant newlines and collecting comments. The parser employs a hybrid strategy, utilizing recursive descent for high-level declarations and a Pratt parser for expressions, allowing it to efficiently handle complex operator precedence without grammar explosion. It implements the primary data contract by producing the `ast.File` consumed by the rest of the system, and it seamlessly handles complex features like f-string interpolation and contextual keywords.

The **AST** module defines the core data structures that represent the structure of Forge code, acting as the central data hub. It manages state primarily as a tree of polymorphic nodes utilizing a "Kind + Data" pattern to avoid deep interface hierarchies, making it highly efficient for recursive traversals. Crucially, it includes a generative desugaring engine that transforms high-level constructs, such as relations, interface fields, and default implementations, into fundamental AST nodes before semantic analysis occurs. It also handles the selective merging of the standard library based on reachability analysis. It implements the primary data contract consumed by the checker and LIR modules.

The **Checker** module implements the semantic analysis phase of the compiler, ensuring that a Forge program is semantically valid before code generation. It maintains complex internal state, including a stack of scopes for variable tracking and a global registry for user-defined types. It consumes the desugared AST, resolves all type expressions, infers generic types, enforces concurrency safety through guarded annotations, and validates the exhaustiveness of match statements. It acts as the semantic truth of the compiler, annotating the AST nodes with `ResolvedType` information, fulfilling the semantic contract required by the subsequent lowering phase.

The **LIR** module is the architectural bridge between the high-level AST and the backend code generators. It resolves all semantic complexity into a simplified, flat representation while preserving structured control flow. It manages state during the lowering process by mapping AST nodes to LIR constructs and assigning all sub-expressions to unique temporary variables, ensuring that backends never encounter nested expression trees. It includes sophisticated optimization passes for side-effect elimination and multi-return destructuring, as well as a monomorphization pass to specialize generic types for backends like C. It consumes the type-checked AST and produces the `LProgram` contract, which is then consumed by its internal Go and C backends to emit target code.

The **Verifier** module is a standalone structural integrity engine designed to detect architectural drift by comparing high-level `.forge` specifications against the actual Go implementation. It maintains state during a run to aggregate type information extracted from Go source files using the standard library's AST tools. It consumes the AST produced by the parser for `.forge` files and performs deep structural comparisons, normalizing naming conventions and mapping types across the two systems. It also performs a reverse completeness check, ensuring every exported Go symbol is documented in the Forge specification, and aggregates any mismatches into a detailed report.

The **Runtime** module provides the foundational C runtime library required by the Forge C backend. It is implemented as a passive, header-only library that establishes the memory layout and operational semantics for Forge's high-level constructs in a C environment. It does not maintain internal state but provides a suite of macros and inline functions for dynamic slices, optional types, error results, tagged unions, and concurrency primitives. These foundational elements are consumed by the C code generated by the LIR module, bridging the gap between Forge's expressive type system and C's low-level primitives.

## Integration Patterns & Workflows

The system relies on clear, sequential workflows to transform data across module boundaries. Two primary workflows define the operation of the Forge toolchain: the compilation pipeline and the architectural verification process.

The compilation pipeline workflow traces the journey of a Forge source file from raw text to executable code. The process begins when a file is read from disk and passed to the parser module. The lexer tokenizes the text, handling significant newlines, and the parser constructs an initial `ast.File` using recursive descent and Pratt parsing. This AST is immediately passed to the AST module's desugaring functions, which expand complex features like relations into standard classes and implementation blocks, and selectively merge required standard library components. The desugared AST is then handed to the checker module. The checker performs a declaration pass to populate its type registry, followed by a deep traversal to resolve types, infer generics, enforce concurrency safety, and validate semantics, annotating the AST with `ResolvedType` fields. Once validated, the AST and the type registry are passed to the LIR module's lowerer. The lowerer flattens expressions into temporaries and converts high-level constructs into LIR statements while preserving structured control flow. The resulting `LProgram` undergoes optimization passes to remove redundant operations and destructure multi-returns. If the target is C, a monomorphization pass specializes generic types. Finally, the backend emitter traverses the LIR and generates the final source code string, which is written to disk.

The architectural verification workflow operates independently to ensure design fidelity. The process starts when the verifier module is invoked on a `.forge` specification file. It uses the parser module to generate an AST of the specification. The verifier then inspects the source annotations within the AST to locate the corresponding Go source files. It uses the Go standard library to extract all exported types, interfaces, and function signatures from these Go files, building a comprehensive internal registry that aggregates information across multiple files. The verifier then recursively walks the Forge AST, translating Forge type expressions into Go string representations and comparing them against the extracted Go types. It normalizes naming conventions on the fly to facilitate this comparison. Finally, it performs a reverse completeness check, ensuring every exported Go symbol is documented in the Forge specification, and aggregates any mismatches into a detailed report.

## Dependency Overview

The project exhibits a strict, unidirectional dependency graph that prevents circular imports and enforces a clear separation of concerns. This layering is critical for maintaining the maintainability of the compiler.

At the base of the hierarchy is the AST module, which acts as a leaf node containing pure data structures. The parser module depends solely on the AST module to construct its output, ensuring that parsing logic is decoupled from semantic analysis. The checker module depends on both the AST module for input and annotation, and the parser module to handle imported files during its semantic analysis phase. The LIR module depends on the AST and checker modules to perform its lowering process, relying on the semantic truth established by the checker. 

The verifier module depends on the AST and parser modules to read specifications, but operates entirely independently of the compiler's semantic and backend phases. This isolation allows the verifier to act as an objective observer of the codebase. The runtime module is completely independent of all Go code, serving only as a target dependency for the generated C code. This layered architecture ensures that the frontend remains decoupled from backend concerns, and that data structures are defined independently of the logic that manipulates them.
