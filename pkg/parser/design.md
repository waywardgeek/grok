# Parser Module Design

The `parser` module is the foundational entry point for the Forge compiler frontend. Its primary responsibility is to transform raw Forge source code—whether in declarative `.forge` files or imperative `.fg` files—into a structured Abstract Syntax Tree (AST). It employs a hand-written lexical scanner and a hybrid parsing strategy that combines recursive descent for high-level structures with a Pratt (precedence climbing) parser for expressions. This architecture ensures that the parser is both highly performant and capable of providing precise error reporting.

## Executive Summary

The `parser` module serves as the "front door" of the compilation pipeline. It takes raw source text and produces a comprehensive representation of the program's structure as defined in the [pkg/ast](../ast/design.md) module. The module is designed to handle the unique syntactic requirements of the Forge language, including significant newlines, complex f-string interpolation, and a rich set of declarative constructs for architectural specification. By producing a unified AST, it allows subsequent stages of the compiler, such as the checker and the lowerer, to operate on a consistent data model regardless of the original source file's purpose.

## File Inventory

- [lexer.go](lexer.go): Implements a stateful, hand-written lexical scanner. It tokenizes the source text into a stream of `Token` objects, handles significant newlines, and collects line comments for inclusion in the final AST.
- [parser.go](parser.go): Contains the core `Parser` implementation and the recursive descent logic for top-level declarations. It manages the parsing of `forge` blocks, classes, interfaces, enums, and complex type expressions.
- [expr_parser.go](expr_parser.go): Implements the expression engine using the Pratt algorithm and the statement parser using recursive descent. It also manages specialized logic for match expressions, lambdas, and the recursive parsing of interpolated f-strings.
- [parser.forge](parser.forge): A Forge specification file that provides a high-level architectural overview and self-documentation of the parser module.

## Architecture and Data Flow

The transformation from raw text to an AST follows a well-defined pipeline within the module. The process begins with the `Lexer`, which reads the source string character by character. Unlike many traditional lexers, the Forge lexer treats newlines as significant tokens (`TNewline`) when they serve as statement or declaration separators, while collapsing multiple consecutive newlines to simplify the parser's job. Comments are captured during this phase and stored in a dedicated slice, allowing them to be attached to the root `ast.File` node without cluttering the token stream.

The `Parser` consumes tokens from the `Lexer` and uses recursive descent to navigate the language's grammar. When the parser identifies a top-level construct, such as a `class` or `func`, it recursively calls specialized methods to parse the constituent parts. For expressions, the parser delegates to the Pratt parsing logic in `expr_parser.go`. This algorithm uses a precedence table to efficiently handle operator precedence and associativity, allowing for complex expressions like method call chains and nested indexing without the "grammar explosion" often seen in pure recursive descent parsers.

As the parser matches grammatical rules, it instantiates the corresponding nodes from the `pkg/ast` module. These nodes are linked together to form a tree. A notable feature of this process is the handling of f-strings: when the lexer encounters an interpolated string, it captures the raw content; the parser then identifies the `{expr}` boundaries and recursively invokes a new parser instance to process the embedded expressions, merging the results into a `StringInterpExpr` node.

## Interface Implementations

The `parser` module does not implement external interfaces. It is designed as a pure producer of the `ast.File` structure. This structure serves as the primary data contract consumed by the [pkg/checker](../checker/design.md) and the [pkg/lir](../lir/design.md) modules.

## Public API

### Entry Points

The module provides two primary entry points for external callers:

- **`ParseFile(source, filename string) (*ast.File, error)`**: This is the standard entry point for parsing a Forge source file. It initializes the lexer and parser, executes the parsing logic, and returns the resulting AST. If parsing fails, it returns a `ParseError` containing detailed location information.
- **`ParseString(source string) (*ast.File, error)`**: A convenience wrapper around `ParseFile` that uses `"<string>"` as the filename, typically used for testing or parsing small snippets of code.

### Core Types

- **`Parser`**: The central parsing engine. While it can be instantiated manually, it is typically used through the package-level entry points. It maintains the lexer state and a list of accumulated errors.
- **`Lexer`**: The stateful scanner responsible for tokenization. It provides `Peek` and `Next` methods for the parser to consume tokens.
- **`ParseError`**: A specialized error type that implements the `error` interface. It includes a descriptive message and an `ast.Span` that identifies the exact location of the error in the source code.

## Implementation Details

### Lexical Scanning and Significant Newlines

The lexer is a stateful scanner that uses a `peek`/`advance` pattern. It is responsible for identifying keywords, literals, and punctuation. A critical aspect of the Forge lexer is its handling of newlines. Newlines are preserved as tokens to separate statements, but the lexer automatically collapses multiple consecutive newlines and skips whitespace that does not contribute to the structure. This allows the parser to remain simple while supporting an ergonomic, newline-sensitive syntax.

### Disambiguation and Backtracking

The Forge grammar contains several ambiguities that require lookahead to resolve. The parser handles these cases using a simple but effective backtracking mechanism: it saves the current state of the `Lexer` (which is a small, copyable struct), attempts to parse a specific construct, and restores the saved state if the attempt fails. This is used in two primary scenarios:
- **Struct Literals vs. Blocks**: When the parser encounters an identifier followed by `{`, it peeks ahead to see if the content looks like a field initializer (`name: value`) to distinguish a struct literal from a block of code.
- **Generic Calls vs. Comparisons**: The `<` operator can start a generic argument list or a comparison. The parser attempts to parse a type argument list and backtracks if it fails or if the following token is not an opening parenthesis.

### Pratt Parsing for Expressions

The expression parser in `expr_parser.go` implements the Pratt algorithm, also known as precedence climbing. This approach is particularly well-suited for languages with many levels of operator precedence. The parser maintains a table of precedence levels and associativity for each operator. When parsing an expression, it recursively climbs the precedence ladder, ensuring that operators are grouped correctly. Postfix operators, such as member access (`.`), function calls (`()`), and indexing (`[]`), are handled in a loop after the primary expression is parsed, allowing for arbitrary chaining of operations.

### Contextual Keywords

To maintain a clean and ergonomic syntax, Forge treats many keywords (such as `why`, `doc`, and `source`) as "contextual." The `expectIdentLike` method allows these tokens to be used as identifiers in positions where they are not ambiguous, such as field names or parameter names. This prevents the language from having an overly restrictive set of reserved words.

### Statement and Block Parsing

Statements are parsed using recursive descent, recognizing keywords like `let`, `if`, `for`, `while`, `match`, and `return`. If no keyword is matched, the parser attempts to parse an expression statement or an assignment. Assignments desugar compound operators (like `+=`) into their binary equivalents during the parsing phase, simplifying the work for later compiler stages.

## Dependencies

- **[pkg/ast](../ast/design.md)**: The parser is entirely dependent on the `ast` module for its output data structures. It uses `ast.Pos` and `ast.Span` for location tracking and constructs various `ast.Node` types to build the tree.

## Technical Debt and Future Work

- **Error Recovery**: The current implementation often stops at the first encountered error. Implementing a synchronization mechanism (e.g., skipping to the next newline or closing brace) would allow the parser to report multiple errors in a single pass, improving the developer experience.
- **Performance of Backtracking**: While copying the `Lexer` state is simple, it can be inefficient for deeply nested ambiguities. A more formal lookahead buffer or a GLR-based approach might be considered if parsing performance becomes a bottleneck.
- **F-String Interpolation**: The recursive invocation of the parser for f-strings is powerful but creates overhead. Pre-tokenizing the interpolated parts during the initial lexing phase or using a more integrated approach could improve efficiency.
