### USER (gemini)

# Task: Synthesize Project Architecture

Please synthesize a high-level `project-design.md` by analyzing the individual `design.md` files from all modules. This document will be our primary map for navigating this project in the future.

## Project Context

- Project Path: /Users/bill/projects/grok
- Total Modules: 3

## Modules to Analyze

- ast: pkg/ast/design.md
- parser: pkg/parser/design.md
- verifier: pkg/verifier/design.md

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
[ID: tool_3f49ea0464b76733]
File: /Users/bill/projects/grok/pkg/ast/design.md (65 lines)

# AST Module Design

## Executive Summary

The `ast` module defines the Abstract Syntax Tree (AST) for the Grok language. It serves as the common representation for both `.grok` (declaration-only) and `.gk` (full implementation) files. The AST is designed to be expressive enough to capture complex type systems, relations, and semantic annotations while maintaining precise source location information for error reporting and tooling.

## File Inventory

- [ast.go](ast.go): Contains the Go type definitions for all AST nodes, including declarations for classes, structs, enums, interfaces, and functions.
- [ast.grok](ast.grok): A self-describing Grok specification of the AST, providing a high-level overview of the data structures and their relationships.

## Architecture and Data Flow

The AST is organized hierarchically, starting from a `File` node which contains one or more `GrokBlock` nodes. Each `GrokBlock` represents a logical grouping of declarations, including imports, documentation, invariants, and various type declarations.

Data typically flows into the AST from a parser. The AST nodes are designed to be the primary exchange format between the parser, type checker, and subsequent analysis or code generation phases.

A key architectural pattern in this module is the use of a tagged union for type expressions. The `TypeExpr` struct uses a `Kind` field to discriminate between different types of expressions (such as Named, Optional, Union, or Sequence) and a `Data` field to hold the specific payload for that kind. This allows for a flexible and recursive representation of the Grok type system.

## Interface Implementations

The `ast` module primarily defines data structures and does not implement external interfaces. It provides the core types that other modules in the Grok ecosystem consume and manipulate.

## Public API

The public API consists of the exported structs and enums defined in `ast.go`. The primary entry point for representing a parsed source file is the `File` struct.

### Core Structures

- **File**: The root node representing a source file, containing a collection of `GrokBlock` nodes.
- **GrokBlock**: A container for declarations within a `grok { ... }` block, including structs, enums, interfaces, classes, functions, and relations.
- **TypeExpr**: A versatile structure for representing any Grok type expression. It uses `TypeExprKind` to identify the specific type (e.g., `TypeNamed`, `TypeOptional`, `TypeUnion`).

### Declarations

- **StructDecl**: Defines a named tuple or record type.
- **EnumDecl**: Defines a sum type with multiple variants.
- **InterfaceDecl**: Defines a set of method signatures that a type can implement.
- **ClassDecl**: Defines a class with fields, methods, and constructor parameters.
- **FuncDecl**: Defines a function or method, including its parameters, return type, and semantic annotations.
- **RelationDecl**: Models relationships between types, such as ownership or reference associations.

### Metadata and Positioning

- **Annotations**: Captures rich metadata for functions, such as concurrency requirements, lock management, and formal contracts (preconditions and postconditions).
- **Pos and Span**: Provide precise source location information (file, line, column) for every major AST node, enabling accurate error reporting.

## Implementation Details

The implementation of `TypeExpr` is particularly noteworthy. By using a `Kind` enum and an `any` data field, it can represent complex, nested types like `map[string][(int, bool)?]` in a flat structure. The specific types held in the `Data` field (e.g., `NamedType`, `UnionType`, `SequenceType`) are also defined within `ast.go`.

The `Annotations` struct is designed to support Grok's focus on safety and concurrency. It includes fields for tracking whether a function is `Concurrent`, which locks it `RequiresLock` or `ExcludesLock`, and whether it is `Pure` or `Spawns` new tasks.

Relations are defined using `RelationDecl`, which specifies the `Parent` and `Child` sides of a relationship, the `Kind` (Owns or Refs), and whether it is a one-to-many relationship (`IsMany`).

## Dependencies

The `ast` module is a leaf package and does not depend on any other modules within the Grok project. It relies solely on the Go standard library.

## Technical Debt and Future Work

- **Function Bodies**: The `FuncDecl` struct currently lacks a field for the function body. This will be necessary to support full implementation files (`.gk`).
- **Semantic Validation**: While the module defines the structure of the AST, it does not currently include logic for semantic validation (e.g., checking for duplicate field names or invalid type combinations).
- **Serialization**: Adding serialization support (e.g., JSON tags) would be beneficial for tools that need to export or cache the AST.
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
[ID: tool_989a34e264952bdb]
File: /Users/bill/projects/grok/pkg/parser/design.md (62 lines)

# Parser Module Design

## Executive Summary

The `parser` module is the front-end of the Grok compiler. It is responsible for transforming Grok source code (typically `.grok` or `.gk` files) into a structured Abstract Syntax Tree (AST). The module employs a two-phase architecture consisting of a hand-written lexer and a recursive descent parser that follows Parsing Expression Grammar (PEG) principles. It is designed to be expressive, handling Grok's rich type system, relationship modeling, and extensive semantic annotations while providing precise source location information for all constructs.

## File Inventory

- [lexer.go](lexer.go): Implements the `Lexer`, which tokenizes Grok source text into a stream of `Token` objects. It handles UTF-8 input, various literal types (including triple-quoted strings), and significant newlines.
- [parser.go](parser.go): Implements the `Parser`, a recursive descent parser that consumes tokens from the `Lexer` and constructs the AST. It handles the core Grok grammar, including blocks, type declarations, functions, and relations.
- [parser.grok](parser.grok): A self-describing Grok specification of the parser's own structure and the AST it produces. It serves as both documentation and a formal model of the module.

## Architecture and Data Flow

The data flow within the module is linear and synchronous. Source text is provided to the `Lexer`, which produces a stream of tokens on demand. The `Parser` consumes these tokens, using lookahead where necessary to make branching decisions.

1. **Lexical Analysis**: The `Lexer` scans the input character by character. It identifies keywords, identifiers, literals, and punctuation. A key feature of the Grok lexer is its handling of newlines; while mostly ignored, they are significant in certain contexts, such as separating declarations or marking the end of certain annotation blocks. The lexer also collapses multiple newlines to simplify the parser's job.
2. **Syntactic Analysis**: The `Parser` uses the token stream to build the AST. It is structured as a set of mutually recursive functions, each corresponding to a grammatical construct (e.g., `parseStruct`, `parseFunc`, `parseTypeExpr`).
3. **AST Construction**: As the parser recognizes constructs, it instantiates the corresponding nodes from the [pkg/ast](../ast/design.md) module. Every major node is tagged with an `ast.Span` representing its exact location in the source file.

## Interface Implementations

- **ParseError**: Implements the standard Go `error` interface. It includes an `ast.Span` to provide context-aware error messages that include the filename, line, and column of the syntax error.

## Public API

The primary entry points for the module are simple functional interfaces that hide the complexity of the lexer and parser state.

- **ParseFile(source, filename string) (*ast.File, error)**: The main entry point for parsing a complete Grok source file. It returns the root `ast.File` node or a `ParseError`.
- **ParseString(source string) (*ast.File, error)**: A convenience wrapper around `ParseFile` that uses `"<string>"` as the filename, useful for testing or dynamic parsing.

The `Lexer` and `Parser` types are also exported, allowing for more granular control or integration into other tools (like language servers), though they are typically used via the high-level functions above.

## Implementation Details

### Lexer Logic

The `Lexer` is a stateful scanner. It supports:
- **Triple-Quoted Strings**: `"""..."""` for multi-line documentation or content, with automatic trimming of leading/trailing whitespace lines.
- **Annotation Keywords**: Keywords like `why`, `doc`, and `requires` are recognized as special tokens. The parser uses contextual information to distinguish between these keywords and their use as identifiers (e.g., a field named `why`).
- **Newline Management**: The `scan` method explicitly handles `\n`, collapsing sequences of newlines and whitespace to ensure the parser sees a clean stream of significant breaks.

### Parser Logic

The `Parser` is a recursive descent implementation of the Grok grammar. Notable aspects include:
- **Identifier Flexibility**: The `expectIdentLike` method allows certain keywords and annotation keywords to be used as identifiers (e.g., field or parameter names), providing more flexibility in the language's surface syntax.
- **Type Expression Parsing**: `parseTypeExpr` handles a recursive and complex type system, including unions (`A | B`), sequences (`[T]`), maps (`map[K]V`), optionals (`T?`), and function types (`A -> B`). It uses a "base type" approach to handle precedence and nesting.
- **Lookahead and State Restoration**: In cases where the grammar is not LL(1), the parser uses a simple lookahead mechanism by saving and restoring the `Lexer` state (e.g., in `isWhyAnnotation` and `parseTupleField`).
- **Annotation Parsing**: Grok's rich metadata (preconditions, postconditions, concurrency constraints) is parsed into an `ast.Annotations` struct. Some annotations, like `requires` and `ensures`, capture the remaining text on a line as a raw string for later analysis by the verifier.
- **Relation Declarations**: The parser handles the unique `relation` syntax, which includes optional hints (like `DoublyLinked`) and labeled sides (`Parent:label owns Child:label`).

## Dependencies

- [pkg/ast](../ast/design.md): The parser is tightly coupled to the AST definition, as it is the primary producer of these structures.

## Technical Debt and Future Work

- **Error Recovery**: The current parser implementation is "fail-fast"—it returns the first error it encounters. Implementing error recovery (e.g., skipping to the next `grok` block or semicolon) would improve the user experience in IDEs.
- **Function Bodies**: While the AST and parser support function declarations, they do not yet handle function bodies (statements and expressions) required for full `.gk` implementation files.
- **Expression Parsing**: The `requires` and `ensures` annotations currently capture raw text. A full expression parser will be needed to turn these into a structured representation for the verifier.
- **Performance**: While sufficient for current needs, the use of lexer state restoration for lookahead could be optimized if parsing performance becomes a bottleneck for very large files.
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
[ID: tool_bffe28873b5c9888]
File: /Users/bill/projects/grok/pkg/verifier/design.md (72 lines)

# Verifier Module Design

## Executive Summary

The `verifier` module is the structural integrity engine of the Grok system. Its primary purpose is to detect "architectural drift"—the divergence between the high-level design declared in `.grok` files and the actual implementation in Go source code. By comparing the Abstract Syntax Tree (AST) of a Grok specification against the AST of the corresponding Go code, the verifier ensures that the developer's mental model (as captured in Grok) remains synchronized with the reality of the codebase. It reports mismatches in types, field names, function signatures, and interface implementations as errors, warnings, or informational findings.

## File Inventory

- [verifier.go](verifier.go): The core implementation of the verification logic. It handles Go source parsing, type extraction, and the recursive comparison of Grok declarations against Go implementations.
- [verifier_test.go](verifier_test.go): Comprehensive tests for the verifier, including integration tests that run the verifier against the project's own `.grok` files and unit tests for naming convention conversions.

## Architecture and Data Flow

The verifier operates in a multi-stage pipeline that bridges the gap between the Grok and Go type systems.

1.  **Grok Parsing**: The process begins by using the [pkg/parser](../parser/design.md) to transform a `.grok` file into a [pkg/ast](../ast/design.md) representation.
2.  **Go Source Discovery**: For each `grok` block in the file, the verifier identifies the associated Go source files or directories specified in the `source:` annotations.
3.  **Go Type Extraction**: The verifier uses the standard library's `go/parser` and `go/ast` packages to parse the referenced Go code. It then traverses the Go AST to build an internal `goTypeInfo` map, which aggregates all structs, interfaces, functions, and type definitions found across the specified sources.
4.  **Structural Comparison**: The verifier performs a deep, recursive comparison between the Grok AST nodes and the extracted Go type information. It maps Grok's snake_case identifiers to Go's PascalCase or camelCase conventions and converts Grok's type expressions into a normalized string format for comparison.
5.  **Finding Generation**: Any discrepancies found during comparison are recorded as `Finding` objects, categorized by `Severity` (Error, Warning, or Info).

## Interface Implementations

The `verifier` module does not currently implement any external interfaces. It provides a functional API for performing verification and returning a structured `Result`.

## Public API

The public API is designed for simplicity and integration into build pipelines or CI/CD workflows.

- **Verify(grokPath string) (*Result, error)**: The primary entry point. It parses the specified `.grok` file, locates the corresponding Go source code, performs the verification, and returns a `Result` containing all findings.
- **Result**: A container for the findings of a verification run. It provides an `ErrorCount()` method to quickly determine if any critical drift was detected.
- **Finding**: Represents a single instance of drift. It includes the `Severity`, the paths to the involved files, and a descriptive message.
- **Severity**: An enumeration (`Error`, `Warning`, `Info`) that classifies the impact of a finding.

## Implementation Details

### Go Type Extraction

The `extractGoTypes` function is the heart of the Go analysis phase. It processes `*ast.File` nodes and populates a `goTypeInfo` structure. It specifically handles:
-   **Structs**: Capturing field names and types.
-   **Interfaces**: Capturing method signatures.
-   **Functions**: Capturing top-level functions and their signatures.
-   **Methods**: Identifying receiver types to associate methods with their respective structs.
-   **Type Definitions**: Tracking simple type aliases.

### Type Mapping and Comparison

Grok and Go have different type systems and naming conventions. The verifier bridges these using several strategies:
-   **Naming Conventions**: The `snakeToPascal` and `snakeToCamel` helpers allow the verifier to match a Grok field like `user_id` to a Go field `UserID` or `userId`.
-   **Type Normalization**: The `grokTypeToGoString` function converts complex Grok `TypeExpr` nodes (like `[string]?`) into their Go equivalents (like `*[]string`).
-   **Fuzzy Matching**: The `typesMatch` function handles Go-specific nuances, such as the equivalence of `any` and `interface{}` and the presence of package prefixes (e.g., matching `ast.File` in Go to `File` in Grok).

### Verification Logic

The verification is partitioned by declaration type:
-   **Structs/Classes**: Checks for field existence and type compatibility. It also reports "extra" fields in Go that are missing from Grok as warnings, encouraging complete documentation.
-   **Interfaces**: Ensures that all methods declared in the Grok interface exist in the Go interface with matching signatures.
-   **Functions**: Compares parameter counts, parameter types, and return types. It handles Grok's "self" parameter (used in class methods) by excluding it from the positional comparison with Go parameters.

## Dependencies

-   [pkg/ast](../ast/design.md): Provides the AST structures for the Grok language.
-   [pkg/parser](../parser/design.md): Used to parse `.grok` files.
-   **Go Standard Library**: Relies heavily on `go/ast`, `go/parser`, and `go/token` for analyzing Go source code.

## Technical Debt and Future Work

-   **Expression Verification**: Currently, the verifier only checks structural types. Future versions should verify the `requires` and `ensures` contracts by parsing the expressions and potentially using static analysis or symbolic execution.
-   **Pointer Nuances**: The current type comparison is somewhat simplified regarding pointer indirection. More robust handling of pointer vs. value receivers and types would reduce false positives/negatives.
-   **Import Resolution**: The verifier currently assumes that unqualified types in Grok match types in the Go package being analyzed. Support for cross-package type verification would be a significant improvement.
-   **Performance**: For very large projects, parsing all Go files for every `grok` block might become slow. Implementing a cache for parsed Go type information would improve performance.
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "content": "# Grok Project Architecture\n\n## Executive Summary\n\nThe Grok project is a structural integrity engine and language toolchain designed to bridge the gap between high-level architectural design and concrete implementation. It introduces the Grok language, which allows developers to declare the structural and semantic contracts of their systems—such as types, relations, and concurrency constraints—in `.grok` files. The primary goal of the project is to detect \"architectural drift\" by comparing these declarative models against the actual Go source code implementation. By ensuring that the developer's mental model remains synchronized with the reality of the codebase, Grok acts as a continuous verification system for software architecture.\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/grok/project-design.md`
**Size**: 748 bytes
**Permissions**: -rw-r--r--
**Lines**: 6
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "append": true,
  "content": "\n## System Architecture\n\nThe architecture of the Grok project follows a classic compiler frontend pipeline integrated with a static analysis engine. The system is composed of three primary modules: the Abstract Syntax Tree (`ast`), the `parser`, and the `verifier`. The philosophy of the architecture is centered around a single source of truth for the structural model, which is the AST. Data flows linearly from source text into the parser, which constructs the AST. This AST then serves as the foundational data structure for the verifier. The verifier acts as a bridge between two distinct type systems, extracting type information from the target Go source code and performing a deep, recursive comparison against the Grok AST. This decoupled design ensures that the parsing logic is entirely independent of the verification logic, allowing the AST to potentially be used for other tooling, such as code generation or language servers, in the future.\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/grok/project-design.md`
**Size**: 1704 bytes
**Permissions**: -rw-r--r--
**Lines**: 5
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "append": true,
  "content": "\n## Interface \u0026 Contract Map\n\nWhile the Grok project currently relies on concrete data structures rather than abstract Go interfaces for its internal boundaries, the system is defined by several critical contracts that govern module interactions.\n\n- **The AST Contract**: The `ast.File` structure defines the structural contract for the entire system. The `parser` module acts as the sole producer of this contract, transforming raw source text into a valid syntax tree. The `verifier` module acts as the primary consumer, traversing the `ast.File` to drive its analysis of the Go source code.\n- **The Verification Result Contract**: The `verifier` module exposes a `Result` structure containing a collection of `Finding` objects. This serves as the external contract for any build pipeline, CI/CD system, or IDE integration that consumes the output of the Grok toolchain.\n- **The Go AST Interface**: The `verifier` heavily consumes the Go standard library's `go/ast` and `go/parser` interfaces to extract type information from the target codebase, acting as a client to the Go compiler's frontend.\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/grok/project-design.md`
**Size**: 2803 bytes
**Permissions**: -rw-r--r--
**Lines**: 9
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "append": true,
  "content": "\n## Module Map\n\n- **Core Data Structures**\n  - [pkg/ast](pkg/ast/design.md)\n- **Frontend Toolchain**\n  - [pkg/parser](pkg/parser/design.md)\n- **Analysis Engine**\n  - [pkg/verifier](pkg/verifier/design.md)\n\n### AST Module\nThe `ast` module is the foundational data model for the Grok language. It defines the Abstract Syntax Tree, capturing the hierarchical structure of `.grok` files, including blocks, type declarations, functions, and complex relations. A critical aspect of its design is the use of a tagged union pattern for type expressions, allowing for a flexible and recursive representation of Grok's rich type system. It manages no internal state, serving purely as a collection of data structures. It implements no external interfaces but provides the core types consumed by the parser and verifier.\n\n### Parser Module\nThe `parser` module is responsible for lexical and syntactic analysis, transforming raw Grok source code into the structured AST. It employs a two-phase architecture with a hand-written, stateful lexer and a recursive descent parser. The lexer manages state to handle complex tokens like triple-quoted strings and significant newlines, while the parser manages the state of the token stream, occasionally using lookahead and state restoration to resolve ambiguities. The parser consumes raw text and produces the `ast.File` contract.\n\n### Verifier Module\nThe `verifier` module is the structural integrity engine that detects architectural drift. It manages a complex internal state during execution, specifically the `goTypeInfo` map, which aggregates all structs, interfaces, and functions extracted from the target Go source code. The verifier consumes the `ast.File` contract produced by the parser and the Go AST produced by the standard library. It bridges the gap between these two models by normalizing naming conventions and type expressions, ultimately producing a structured `Result` containing any detected discrepancies.\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/grok/project-design.md`
**Size**: 4765 bytes
**Permissions**: -rw-r--r--
**Lines**: 19
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "append": true,
  "content": "\n## Integration Patterns \u0026 Workflows\n\nThe most critical cross-module workflow is the end-to-end verification process. When a user invokes the verifier on a `.grok` file, the workflow begins in the `verifier` module, which immediately delegates to the `parser` module. The `parser` instantiates its lexer, tokenizes the source, and constructs an `ast.File` tree, returning it to the verifier. The verifier then inspects the source annotations within the AST to locate the corresponding Go files. It invokes the Go standard library parser to build a Go AST and extracts the relevant type information into its internal state. Finally, the verifier recursively traverses the Grok AST, comparing each declaration against the extracted Go types, normalizing names and types on the fly, and accumulating findings into a final result.\n\nA complex micro-workflow occurs during the structural comparison phase. When the verifier encounters a Grok type expression, it must compare this against a Go type. The verifier traverses the Grok type expression AST node, recursively building a normalized Go string representation. It simultaneously applies naming convention transformations, converting Grok's snake case identifiers to Go's Pascal case or camel case, ensuring that the conceptual model accurately maps to the idiomatic implementation.\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/grok/project-design.md`
**Size**: 6097 bytes
**Permissions**: -rw-r--r--
**Lines**: 7
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "append": true,
  "content": "\n## Dependency Overview\n\nThe dependency graph of the Grok project is strictly layered and unidirectional, preventing circular dependencies and ensuring a clean separation of concerns. At the base of the graph is the `ast` module, which is a leaf package depending only on the Go standard library. The `parser` module sits above the `ast`, depending on it to construct the syntax tree. At the top of the hierarchy is the `verifier` module, which depends on both the `parser` to read `.grok` files and the `ast` to traverse the resulting model. The verifier also introduces a heavy dependency on the Go standard library's `go/ast` and `go/parser` packages. This layered architecture constrains the flow of information: the AST has no knowledge of how it is parsed, and the parser has no knowledge of how the AST will be verified.\n",
  "file_path": "project-design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/grok/project-design.md`
**Size**: 6925 bytes
**Permissions**: -rw-r--r--
**Lines**: 5
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
[ID: tool_ce24a520f97688f6]
File: /Users/bill/projects/grok/project-design.md (46 lines)

# Grok Project Architecture

## Executive Summary

The Grok project is a structural integrity engine and language toolchain designed to bridge the gap between high-level architectural design and concrete implementation. It introduces the Grok language, which allows developers to declare the structural and semantic contracts of their systems—such as types, relations, and concurrency constraints—in `.grok` files. The primary goal of the project is to detect "architectural drift" by comparing these declarative models against the actual Go source code implementation. By ensuring that the developer's mental model remains synchronized with the reality of the codebase, Grok acts as a continuous verification system for software architecture.

## System Architecture

The architecture of the Grok project follows a classic compiler frontend pipeline integrated with a static analysis engine. The system is composed of three primary modules: the Abstract Syntax Tree (`ast`), the `parser`, and the `verifier`. The philosophy of the architecture is centered around a single source of truth for the structural model, which is the AST. Data flows linearly from source text into the parser, which constructs the AST. This AST then serves as the foundational data structure for the verifier. The verifier acts as a bridge between two distinct type systems, extracting type information from the target Go source code and performing a deep, recursive comparison against the Grok AST. This decoupled design ensures that the parsing logic is entirely independent of the verification logic, allowing the AST to potentially be used for other tooling, such as code generation or language servers, in the future.

## Interface & Contract Map

While the Grok project currently relies on concrete data structures rather than abstract Go interfaces for its internal boundaries, the system is defined by several critical contracts that govern module interactions.

- **The AST Contract**: The `ast.File` structure defines the structural contract for the entire system. The `parser` module acts as the sole producer of this contract, transforming raw source text into a valid syntax tree. The `verifier` module acts as the primary consumer, traversing the `ast.File` to drive its analysis of the Go source code.
- **The Verification Result Contract**: The `verifier` module exposes a `Result` structure containing a collection of `Finding` objects. This serves as the external contract for any build pipeline, CI/CD system, or IDE integration that consumes the output of the Grok toolchain.
- **The Go AST Interface**: The `verifier` heavily consumes the Go standard library's `go/ast` and `go/parser` interfaces to extract type information from the target codebase, acting as a client to the Go compiler's frontend.

## Module Map

- **Core Data Structures**
  - [pkg/ast](pkg/ast/design.md)
- **Frontend Toolchain**
  - [pkg/parser](pkg/parser/design.md)
- **Analysis Engine**
  - [pkg/verifier](pkg/verifier/design.md)

### AST Module
The `ast` module is the foundational data model for the Grok language. It defines the Abstract Syntax Tree, capturing the hierarchical structure of `.grok` files, including blocks, type declarations, functions, and complex relations. A critical aspect of its design is the use of a tagged union pattern for type expressions, allowing for a flexible and recursive representation of Grok's rich type system. It manages no internal state, serving purely as a collection of data structures. It implements no external interfaces but provides the core types consumed by the parser and verifier.

### Parser Module
The `parser` module is responsible for lexical and syntactic analysis, transforming raw Grok source code into the structured AST. It employs a two-phase architecture with a hand-written, stateful lexer and a recursive descent parser. The lexer manages state to handle complex tokens like triple-quoted strings and significant newlines, while the parser manages the state of the token stream, occasionally using lookahead and state restoration to resolve ambiguities. The parser consumes raw text and produces the `ast.File` contract.

### Verifier Module
The `verifier` module is the structural integrity engine that detects architectural drift. It manages a complex internal state during execution, specifically the `goTypeInfo` map, which aggregates all structs, interfaces, and functions extracted from the target Go source code. The verifier consumes the `ast.File` contract produced by the parser and the Go AST produced by the standard library. It bridges the gap between these two models by normalizing naming conventions and type expressions, ultimately producing a structured `Result` containing any detected discrepancies.

## Integration Patterns & Workflows

The most critical cross-module workflow is the end-to-end verification process. When a user invokes the verifier on a `.grok` file, the workflow begins in the `verifier` module, which immediately delegates to the `parser` module. The `parser` instantiates its lexer, tokenizes the source, and constructs an `ast.File` tree, returning it to the verifier. The verifier then inspects the source annotations within the AST to locate the corresponding Go files. It invokes the Go standard library parser to build a Go AST and extracts the relevant type information into its internal state. Finally, the verifier recursively traverses the Grok AST, comparing each declaration against the extracted Go types, normalizing names and types on the fly, and accumulating findings into a final result.

A complex micro-workflow occurs during the structural comparison phase. When the verifier encounters a Grok type expression, it must compare this against a Go type. The verifier traverses the Grok type expression AST node, recursively building a normalized Go string representation. It simultaneously applies naming convention transformations, converting Grok's snake case identifiers to Go's Pascal case or camel case, ensuring that the conceptual model accurately maps to the idiomatic implementation.

## Dependency Overview

The dependency graph of the Grok project is strictly layered and unidirectional, preventing circular dependencies and ensuring a clean separation of concerns. At the base of the graph is the `ast` module, which is a leaf package depending only on the Go standard library. The `parser` module sits above the `ast`, depending on it to construct the syntax tree. At the top of the hierarchy is the `verifier` module, which depends on both the `parser` to read `.grok` files and the `ast` to traverse the resulting model. The verifier also introduces a heavy dependency on the Go standard library's `go/ast` and `go/parser` packages. This layered architecture constrains the flow of information: the AST has no knowledge of how it is parsed, and the parser has no knowledge of how the AST will be verified.
```

---

### ASSISTANT (gemini)

**Tool Call: `edit_file`**
```json
{
  "file_path": "project-design.md",
  "new_text": "While the Grok project currently relies on concrete data structures rather than abstract Go interfaces for its internal boundaries, the system is defined by several critical contracts that govern module interactions.\n\nThe primary structural contract for the entire system is defined by the AST file structure. The parser module acts as the sole producer of this contract, transforming raw source text into a valid syntax tree. The verifier module acts as the primary consumer, traversing this file structure to drive its analysis of the Go source code.\n\nAnother critical contract is the verification result structure exposed by the verifier module. This structure contains a collection of finding objects, serving as the external contract for any build pipeline, continuous integration system, or integrated development environment that consumes the output of the Grok toolchain.\n\nFinally, the verifier heavily consumes the Go standard library's abstract syntax tree and parser interfaces to extract type information from the target codebase. In this capacity, the verifier acts as a client to the Go compiler's frontend, relying on these external interfaces to build its internal representation of the implementation's reality.",
  "old_text": "While the Grok project currently relies on concrete data structures rather than abstract Go interfaces for its internal boundaries, the system is defined by several critical contracts that govern module interactions.\n\n- **The AST Contract**: The `ast.File` structure defines the structural contract for the entire system. The `parser` module acts as the sole producer of this contract, transforming raw source text into a valid syntax tree. The `verifier` module acts as the primary consumer, traversing the `ast.File` to drive its analysis of the Go source code.\n- **The Verification Result Contract**: The `verifier` module exposes a `Result` structure containing a collection of `Finding` objects. This serves as the external contract for any build pipeline, CI/CD system, or IDE integration that consumes the output of the Grok toolchain.\n- **The Go AST Interface**: The `verifier` heavily consumes the Go standard library's `go/ast` and `go/parser` interfaces to extract type information from the target codebase, acting as a client to the Go compiler's frontend."
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Edited Successfully**

**File**: `/Users/bill/projects/grok/project-design.md`
**Changes**: Replaced 1068 characters with 1228 characters (+160)
**Line Change**: +2 lines
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
[ID: tool_8e1f081cd68709b7]
File: /Users/bill/projects/grok/project-design.md (48 lines)

# Grok Project Architecture

## Executive Summary

The Grok project is a structural integrity engine and language toolchain designed to bridge the gap between high-level architectural design and concrete implementation. It introduces the Grok language, which allows developers to declare the structural and semantic contracts of their systems—such as types, relations, and concurrency constraints—in `.grok` files. The primary goal of the project is to detect "architectural drift" by comparing these declarative models against the actual Go source code implementation. By ensuring that the developer's mental model remains synchronized with the reality of the codebase, Grok acts as a continuous verification system for software architecture.

## System Architecture

The architecture of the Grok project follows a classic compiler frontend pipeline integrated with a static analysis engine. The system is composed of three primary modules: the Abstract Syntax Tree (`ast`), the `parser`, and the `verifier`. The philosophy of the architecture is centered around a single source of truth for the structural model, which is the AST. Data flows linearly from source text into the parser, which constructs the AST. This AST then serves as the foundational data structure for the verifier. The verifier acts as a bridge between two distinct type systems, extracting type information from the target Go source code and performing a deep, recursive comparison against the Grok AST. This decoupled design ensures that the parsing logic is entirely independent of the verification logic, allowing the AST to potentially be used for other tooling, such as code generation or language servers, in the future.

## Interface & Contract Map

While the Grok project currently relies on concrete data structures rather than abstract Go interfaces for its internal boundaries, the system is defined by several critical contracts that govern module interactions.

The primary structural contract for the entire system is defined by the AST file structure. The parser module acts as the sole producer of this contract, transforming raw source text into a valid syntax tree. The verifier module acts as the primary consumer, traversing this file structure to drive its analysis of the Go source code.

Another critical contract is the verification result structure exposed by the verifier module. This structure contains a collection of finding objects, serving as the external contract for any build pipeline, continuous integration system, or integrated development environment that consumes the output of the Grok toolchain.

Finally, the verifier heavily consumes the Go standard library's abstract syntax tree and parser interfaces to extract type information from the target codebase. In this capacity, the verifier acts as a client to the Go compiler's frontend, relying on these external interfaces to build its internal representation of the implementation's reality.

## Module Map

- **Core Data Structures**
  - [pkg/ast](pkg/ast/design.md)
- **Frontend Toolchain**
  - [pkg/parser](pkg/parser/design.md)
- **Analysis Engine**
  - [pkg/verifier](pkg/verifier/design.md)

### AST Module
The `ast` module is the foundational data model for the Grok language. It defines the Abstract Syntax Tree, capturing the hierarchical structure of `.grok` files, including blocks, type declarations, functions, and complex relations. A critical aspect of its design is the use of a tagged union pattern for type expressions, allowing for a flexible and recursive representation of Grok's rich type system. It manages no internal state, serving purely as a collection of data structures. It implements no external interfaces but provides the core types consumed by the parser and verifier.

### Parser Module
The `parser` module is responsible for lexical and syntactic analysis, transforming raw Grok source code into the structured AST. It employs a two-phase architecture with a hand-written, stateful lexer and a recursive descent parser. The lexer manages state to handle complex tokens like triple-quoted strings and significant newlines, while the parser manages the state of the token stream, occasionally using lookahead and state restoration to resolve ambiguities. The parser consumes raw text and produces the `ast.File` contract.

### Verifier Module
The `verifier` module is the structural integrity engine that detects architectural drift. It manages a complex internal state during execution, specifically the `goTypeInfo` map, which aggregates all structs, interfaces, and functions extracted from the target Go source code. The verifier consumes the `ast.File` contract produced by the parser and the Go AST produced by the standard library. It bridges the gap between these two models by normalizing naming conventions and type expressions, ultimately producing a structured `Result` containing any detected discrepancies.

## Integration Patterns & Workflows

The most critical cross-module workflow is the end-to-end verification process. When a user invokes the verifier on a `.grok` file, the workflow begins in the `verifier` module, which immediately delegates to the `parser` module. The `parser` instantiates its lexer, tokenizes the source, and constructs an `ast.File` tree, returning it to the verifier. The verifier then inspects the source annotations within the AST to locate the corresponding Go files. It invokes the Go standard library parser to build a Go AST and extracts the relevant type information into its internal state. Finally, the verifier recursively traverses the Grok AST, comparing each declaration against the extracted Go types, normalizing names and types on the fly, and accumulating findings into a final result.

A complex micro-workflow occurs during the structural comparison phase. When the verifier encounters a Grok type expression, it must compare this against a Go type. The verifier traverses the Grok type expression AST node, recursively building a normalized Go string representation. It simultaneously applies naming convention transformations, converting Grok's snake case identifiers to Go's Pascal case or camel case, ensuring that the conceptual model accurately maps to the idiomatic implementation.

## Dependency Overview

The dependency graph of the Grok project is strictly layered and unidirectional, preventing circular dependencies and ensuring a clean separation of concerns. At the base of the graph is the `ast` module, which is a leaf package depending only on the Go standard library. The `parser` module sits above the `ast`, depending on it to construct the syntax tree. At the top of the hierarchy is the `verifier` module, which depends on both the `parser` to read `.grok` files and the `ast` to traverse the resulting model. The verifier also introduces a heavy dependency on the Go standard library's `go/ast` and `go/parser` packages. This layered architecture constrains the flow of information: the AST has no knowledge of how it is parsed, and the parser has no knowledge of how the AST will be verified.
```

---

### ASSISTANT (gemini)

Project documentation complete. I have successfully synthesized the architectural designs of the `ast`, `parser`, and `verifier` modules into a comprehensive `project-design.md` file at the root of the project. 

The document is written in continuous, flowing prose to ensure an optimal Text-to-Speech experience, with bulleted lists strictly reserved for the module inventory. It covers the executive summary, system architecture, interface and contract map, detailed module descriptions, integration workflows, and a dependency overview, providing a clear and deep mental model of the Grok project.

---

