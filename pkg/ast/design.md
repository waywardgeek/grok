# AST Module Design

## Executive Summary

The `ast` module provides the foundational data model for the Grok language. It defines the Abstract Syntax Tree (AST), which serves as the common representation for both `.grok` (declaration-only) and `.gk` (full implementation) files. The AST is designed to be highly expressive, capturing complex type systems, semantic relations, and rich metadata while maintaining precise source location information. As a leaf package in the Grok hierarchy, it manages no internal state and provides the structural contracts consumed by the parser and verifier.

## File Inventory

- [ast.go](ast.go): The primary source file containing the Go type definitions for all AST nodes, including declarations for classes, structs, enums, interfaces, and functions.
- [ast.grok](ast.grok): A self-describing Grok specification of the AST, providing a high-level overview of the data structures and their relationships in the Grok language itself.

## Architecture and Data Flow

The AST is organized as a strict hierarchy, rooted in the `File` structure. A `File` contains one or more `GrokBlock` nodes, each representing a logical grouping of declarations within a `grok { ... }` block. This structure allows a single source file to contain multiple architectural modules or namespaces.

Data flows into the AST from the [parser](../parser/design.md), which transforms raw source text into this structured representation. Once constructed, the AST becomes the "source of truth" for the rest of the toolchain. The [verifier](../verifier/design.md) traverses this tree to compare the declarative model against actual Go source code.

A central architectural pattern in this module is the use of a tagged union for type expressions. The `TypeExpr` structure uses a `Kind` field to discriminate between different types of expressions and a `Data` field to hold the specific payload. This recursive design allows the AST to represent deeply nested types, such as `map[string][(int, bool)?]`, with a consistent and manageable interface.

## Interface Implementations

The `ast` module is a pure data-modeling package and does not implement any external interfaces. It defines the concrete types that serve as the interface between the parser and the verifier.

## Public API

The public API consists of exported structs and enums that define the grammar of the Grok language.

### Core Structures

The `File` struct is the entry point for any parsed Grok source. It aggregates `GrokBlock` nodes, which are the primary containers for declarations. Each `GrokBlock` can include imports, documentation blocks, invariants, and the full suite of Grok type declarations.

### Type Expressions

The `TypeExpr` struct is the most versatile component of the API. It uses `TypeExprKind` to identify the specific nature of a type, such as `TypeNamed` for standard types, `TypeOptional` for nullable types, or `TypeUnion` for sum types. The associated data structures like `NamedType`, `UnionType`, and `SequenceType` provide the specific details for each kind.

### Declarations

The module defines several declaration types that map to Grok's high-level constructs:
- **StructDecl**: Represents named records or tuples.
- **EnumDecl**: Represents sum types with multiple variants.
- **InterfaceDecl**: Defines sets of method signatures and supports interface composition.
- **ClassDecl**: Models complex objects with fields, methods, and constructor parameters.
- **FuncDecl**: Defines functions or methods, including parameters, return types, and semantic annotations.
- **RelationDecl**: Explicitly models relationships between types, such as ownership or reference associations, which is a core feature of Grok's architectural modeling.

### Metadata and Positioning

Every major node in the AST includes a `Span` (composed of `Pos` start and end points). This ensures that every element in the tree can be traced back to its exact location in the source file, enabling high-quality error reporting and IDE integration.

## Implementation Details

The `TypeExpr` implementation is a critical piece of the module's logic. By using an `any` field for data, it avoids the complexity of a deep interface hierarchy while maintaining type safety through the `Kind` discriminator. This pattern is particularly effective for the recursive nature of type systems.

The `Annotations` struct is another vital component, reflecting Grok's focus on safety and concurrency. It captures metadata such as whether a function is `Pure`, whether it `Spawns` new tasks, and its requirements for locking (`RequiresLock`, `ExcludesLock`). These annotations are not just comments; they are first-class members of the AST intended for formal verification.

Relations are handled via `RelationDecl`, which specifies the `Parent` and `Child` sides of a relationship. It supports different relationship kinds (`Owns` vs `Refs`) and cardinality (`IsMany`), allowing the architect to declare the structural constraints of the system's object graph.

## Dependencies

The `ast` module is a leaf package and has no dependencies on other modules within the Grok project. It relies exclusively on the Go standard library.

## Technical Debt and Future Work

- **Function Bodies**: The `FuncDecl` struct currently lacks a field for the function body. While the comments indicate it will hold a `*Block` for `.gk` files, the field has not yet been implemented.
- **Semantic Validation**: The module currently only defines the structure of the AST. It does not provide any methods for semantic validation, such as checking for circular interface inheritance or duplicate field names.
- **Serialization**: There is currently no built-in support for serializing the AST to formats like JSON, which would be useful for external tooling or caching.
- **Visitor Pattern**: As the number of AST node types grows, implementing a Visitor pattern might simplify the traversal logic in the verifier and other future consumers.
