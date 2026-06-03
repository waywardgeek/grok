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
