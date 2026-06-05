# Grok

A typed language for design and implementation — describe your architecture, verify it hasn't drifted, compile it to Go.

**Repository:** [github.com/waywardgeek/grok](https://github.com/waywardgeek/grok)

## What is Grok?

Grok has two modes:

**`.grok` files — understandings.** Declaration-only design artifacts: data structures, APIs, interfaces, annotations, doc blocks, invariants, ownership relations. No function bodies. The AI writes them after implementation; the human reviews them. A structural verifier checks the exported API and type structure against Go source. Narrative annotations such as docs, invariants, and relations are design context; they are parsed, but not semantically proven by the verifier. This is the core of [Grok-Driven Development (GDD)](https://coderhapsody.ai/docs/grok-driven-development).

**`.gk` files — code.** Full Grok with function bodies and executable semantics. Compiles to Go. An existence proof that the language design is sound: if the notation is precise enough to verify against real implementations, function bodies are all that's missing to make it a real language.

## Why?

The real bottleneck in AI-assisted software development is **human review**, not code generation. As AI generates code faster, reviewers drown in PRs. A `.grok` file contains *only* the decisions that matter — data structures, API boundaries, type relationships, concurrency contracts — at 5-10x the information density of source code. The reviewer validates architecture, not syntax. The verifier confirms that the source matches the verifiable structural declarations.

See [Grok-Driven Development](https://coderhapsody.ai/docs/grok-driven-development) for the full methodology.

---

## Quick Start

### Installation

Build the unified CLI from source:

```bash
git clone https://github.com/waywardgeek/grok.git
cd grok
go build -o grok ./cmd/grok/
# Optionally move to your PATH:
# mv grok /usr/local/bin/
```

Or install individual tools:

```bash
go install github.com/waywardgeek/grok/cmd/grok-verify@latest
go install github.com/waywardgeek/grok/cmd/grok-compile@latest
```

### Verify a `.grok` file

```bash
$ grok verify pkg/parser/parser.grok
[WARNING] pkg/parser/parser.grok ↔ lexer.go, parser.go, expr_parser.go: class Lexer: ctor param Source not found as field
...
0 errors, 3 warnings
```

Warnings and informational findings can appear when a `.grok` file intentionally omits implementation details or when constructor/field conventions do not line up exactly.

If the code drifts:

```
[ERROR] parser.grok ↔ parser.go: function ParseString: param count mismatch: .grok=2, Go=1
[WARNING] parser.grok ↔ parser.go: exported type Config not documented in .grok
```

### Compile a `.gk` file

```bash
$ grok compile testdata/demo.gk
wrote demo.go
$ go run demo.go
Task Manager Demo
Added: Buy groceries (priority 2)
...
```

### Generate a `.grok` file from Go source

```bash
$ grok gen pkg/ast/        # scaffolds ast.grok from Go source
$ grok update ast.grok     # auto-adds missing exported symbols
$ grok fmt ast.grok        # formats to canonical style
```

---

## The Compiler

The `.gk` compiler is a full-stack transpiler: parser → type checker → Go code generator.

```
// demo.gk

grok task_demo {
  enum Priority { Low Medium High Critical }

  struct Task {
    name: string
    priority: Priority
    done: bool
  }

  class TaskManager<T>() {
    tasks: [T]

    pub fn add(mut self, task: T) {
      self.tasks = append(self.tasks, task)
    }

    pub fn count(self) -> i32 {
      return <i32>len(self.tasks)
    }
  }

  func main() {
    let mgr = TaskManager<Task>()
    mgr.add(Task { name: "Ship it", priority: Priority.High, done: false })
    println(f"Tasks: {mgr.count()}")
  }
}
```

### Language Features

**Type system:** Generics with constraints and inference, interfaces (structural subtyping), enums (tagged unions with fields), optionals (`T?`, `!` unwrap, `isnull()`), union types (`T | U`), tuples `(T, U)`, type aliases.

**Error handling:** `(T, error)` tuples with Rust-style `?` operator for concise error propagation — works in any expression position:

```
func process(s: string) -> (i32, error) {
    let result = double(parse(s)?)      // nested ? in function args
    let sum = parse("a")? + parse("b")? // multiple ? in one expression
    return (result + sum, null)
}
```

**Concurrency:** `spawn` (goroutines), typed channels (`make_channel<T>`, send/receive/close), `select` with receive binding, `lock(mu) { ... }` (scoped mutex).

**Concurrency safety:** `guarded_by(mu)` annotations are enforced at compile time — accessing a guarded field outside its lock scope is a checker error:

```
class Counter() {
    count: i32 guarded_by(mu)  // must access within lock(mu)
    mu: lock

    pub fn increment(mut self) {
        lock(self.mu) {
            self.count = self.count + 1  // OK
        }
    }

    pub fn bad_read(self) -> i32 {
        return self.count  // ERROR: field "count" is guarded by "mu"
    }
}
```

**Pattern matching:** Enum/union type switches, nested patterns (`Some(Circle(r)) =>`), guard clauses (`x if x > 0 =>`), exhaustiveness warnings, tuple destructuring.

**Other:** Lambdas, f-strings, visibility (`pub`/private), built-in methods on string/list/map/channel, numeric types (i8–i256, u8–u256, f32–f128), type casts, modules (`import X from "file.gk"`), `cascade` (defer).

---

## The Toolchain

| Command | Description |
|---|---|
| `grok compile file.gk` | Compile `.gk` to `.go` |
| `grok verify file.grok` | Check `.grok` against Go source for structural drift |
| `grok update file.grok` | Auto-add missing exported symbols, regenerate function index |
| `grok update --prune file.grok` | Also remove stale declarations not in Go source |
| `grok gen pkg/dir/` | Scaffold a new `.grok` file from Go source |
| `grok fmt file.grok` | Format `.grok` to canonical style |

---

## Key Principles

- **`.grok` files live next to the code** they describe (`pkg/ast/ast.grok` alongside `pkg/ast/ast.go`)
- **Cross-file concepts only** — data structures, APIs, interfaces. Not single-file implementation details.
- **Adopt the implementation language's conventions** — PascalCase for Go, snake_case for Python
- **AI writes, human reviews** — implement first, write `.grok` after, human validates architecture
- **Completeness checking** — the verifier warns about exported Go symbols not documented in `.grok`
- **Self-referential** — every compiler package has its own `.grok` file, verified by the tool it contains

---

## Project Structure

```
cmd/grok/              Unified CLI (compile, verify, update, gen, fmt)
cmd/grok-compile/      Standalone compiler CLI
cmd/grok-verify/       Standalone verifier CLI
cmd/grok-gen/          Standalone scaffolding CLI
cmd/grok-update/       Standalone update CLI
pkg/ast/               AST node types + ast.grok
pkg/parser/            PEG parser for .grok and .gk files + parser.grok
pkg/checker/           Type checker with inference + checker.grok
pkg/transpiler/        Go code generator + transpiler.grok
pkg/verifier/          Structural drift detector + verifier.grok
testdata/              34 top-level .gk sample files (all compile; generated Go passes go test)
testdata/modules/      Module system test files
```

---

## Test Status

234 tests across parser, checker, transpiler, and verifier. 34 top-level `.gk` sample files all compile, and the generated Go for those samples passes `go test`. The project/package `.grok` files currently verify with 0 errors and 3 warnings; informational findings report intentionally omitted internal methods.

```bash
$ go test ./...
ok  github.com/waywardgeek/grok/pkg/checker     0.018s
ok  github.com/waywardgeek/grok/pkg/parser       0.006s
ok  github.com/waywardgeek/grok/pkg/transpiler   0.005s
ok  github.com/waywardgeek/grok/pkg/verifier     0.004s
```

### Known Issues

- `features.gk`: `go vet` warns about unreachable code after exhaustive match
- i128/i256 types silently downcast to int64/uint64 (math/big support planned)
- Verification is structural: `requires`, `ensures`, `invariant`, `relation`, and most narrative annotations are not semantically checked
- Some complex or unconvertible type forms are matched permissively to avoid false positives
- No LSP server yet (planned)

---

## Documentation

- [Grok Language Specification](https://coderhapsody.ai/docs/grok-language) — full type system, syntax, and examples
- [Grok-Driven Development](https://coderhapsody.ai/docs/grok-driven-development) — the methodology

---

## License

Apache 2.0 — see [LICENSE](LICENSE).

## Authors

Bill Cox & [CodeRhapsody](https://coderhapsody.ai)

*"grok" is a 60-year-old word from Heinlein's Stranger in a Strange Land meaning deep, complete understanding. We are reclaiming it.*
