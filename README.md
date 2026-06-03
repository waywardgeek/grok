# Grok

Structural understanding files for AI-assisted codebases — describe your architecture, verify it hasn't drifted.

## What is Grok?

Grok is a file format (`.grok`) and verification tool for maintaining compressed architectural descriptions of codebases. A `.grok` file captures data structures, APIs, and interfaces — the cross-file concepts an AI or human needs to understand and review code without reading every source file.

The AI writes `.grok` files after implementation. The human reviews them. The verifier checks they haven't drifted from the source.

## Quick Example

```
// pkg/verifier/verifier.grok — lives next to the code it describes

grok Verifier {
  why: "Compares .grok files against Go source, reporting structural drift."

  enum Severity { Error Warning Info }

  struct Finding {
    Severity: Severity
    GrokFile: string
    GoFile:   string
    Message:  string
  }

  class Result() {
    Findings: [Finding]
    func ErrorCount(self) -> int
  }

  func Verify(grokPath: string) -> (Result?, error)

  source: ["verifier.go"]
}
```

Then verify it:
```bash
$ grok-verify pkg/verifier/verifier.grok
0 errors, 0 warnings
```

If the code drifts from the understanding:
```
[ERROR] verifier.grok ↔ verifier.go: function Verify: param count mismatch: .grok=2, Go=1
```

## Key Principles

- **`.grok` files live next to the code** they describe (`pkg/ast/ast.grok` alongside `pkg/ast/ast.go`)
- **Cross-file concepts only** — data structures, APIs, interfaces. Not single-file implementation details.
- **Adopt the implementation language's conventions** — PascalCase for Go, snake_case for Python
- **AI writes, human reviews** — the natural workflow is implement first, write `.grok` after, human validates architecture
- **Source paths are relative** to the `.grok` file's directory

## Documentation

- [Grok-Driven Development](https://coderhapsody.ai/docs/grok-driven-development) — the methodology
- [Grok Language Specification](https://coderhapsody.ai/docs/grok-language) — the full type system and syntax

## Installation

```bash
go install github.com/waywardgeek/grok/cmd/grok-verify@latest
```

Or build from source:
```bash
git clone git@github.com:waywardgeek/grok.git
cd grok
go build -o grok-verify ./cmd/grok-verify/
```

## Usage

```bash
# Verify a single .grok file
grok-verify pkg/parser/parser.grok

# Verify all .grok files in a project
find . -name '*.grok' -exec grok-verify {} \;
```

## Project Structure

```
cmd/grok-verify/    CLI tool
pkg/ast/            AST node types + ast.grok
pkg/parser/         PEG parser for .grok files + parser.grok
pkg/verifier/       Structural drift detector + verifier.grok
```

The project is self-referential: every package has its own `.grok` file, verified by the tool it contains.

## Status

Early MVP. The parser and verifier work. The verifier checks:
- Type/struct/class/enum/interface existence
- Field names and types
- Method existence and signatures (param count, param types, return types)
- Package-prefix stripping for cross-package type references

## License

Apache 2.0 — see [LICENSE](LICENSE).

## Authors

Bill Cox & [CodeRhapsody](https://coderhapsody.ai)
