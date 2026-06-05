# Forge

A typed language for design and implementation — describe your architecture, verify it hasn't drifted, compile it to Go.

**Repository:** [github.com/waywardgeek/forge](https://github.com/waywardgeek/forge)

## What is Forge?

Forge has two modes:

**`.forge` files — understandings.** Declaration-only design artifacts: data structures, APIs, interfaces, annotations, doc blocks, invariants, ownership relations. No function bodies. The AI writes them after implementation; the human reviews them. A structural verifier checks they haven't drifted from the source. This is the core of [Forge-Driven Development (GDD)](https://coderhapsody.ai/docs/forge-driven-development).

**`.fg` files — code.** Full Forge with function bodies and executable semantics. Compiles to Go. An existence proof that the language design is sound: if the notation is precise enough to verify against real implementations, function bodies are all that's missing to make it a real language.

## Why?

The real bottleneck in AI-assisted software development is **human review**, not code generation. As AI generates code faster, reviewers drown in PRs. A `.forge` file contains *only* the decisions that matter — data structures, API boundaries, type relationships, concurrency contracts — at 5-10x the information density of source code. The reviewer validates architecture, not syntax. The verifier confirms the source matches.

See [Forge-Driven Development](https://coderhapsody.ai/docs/forge-driven-development) for the full methodology.

---

## Quick Start

### Installation

Build the unified CLI from source:

```bash
git clone https://github.com/waywardgeek/forge.git
cd forge
go build -o forge ./cmd/forge/
# Optionally move to your PATH:
# mv forge /usr/local/bin/
```

Or install individual tools:

```bash
go install github.com/waywardgeek/forge/cmd/forge-verify@latest
go install github.com/waywardgeek/forge/cmd/forge-compile@latest
```

### Verify a `.forge` file

```bash
$ forge verify pkg/parser/parser.forge
0 errors, 0 warnings
```

If the code drifts:

```
[ERROR] parser.forge ↔ parser.go: function ParseString: param count mismatch: .forge=2, Go=1
[WARNING] parser.forge ↔ parser.go: exported type Config not documented in .forge
```

### Compile a `.fg` file

```bash
$ forge compile testdata/demo.fg
wrote demo.go
$ go run demo.go
Task Manager Demo
Added: Buy groceries (priority 2)
...
```

### Generate a `.forge` file from Go source

```bash
$ forge gen pkg/ast/        # scaffolds ast.forge from Go source
$ forge update ast.forge     # auto-adds missing exported symbols
$ forge fmt ast.forge        # formats to canonical style
```

---

## The Compiler

The `.fg` compiler is a full-stack transpiler: parser → type checker → Go code generator.

```
// demo.fg

forge task_demo {
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

**Other:** Lambdas, f-strings, visibility (`pub`/private), built-in methods on string/list/map/channel, numeric types (i8–i256, u8–u256, f32–f128), type casts, modules (`import X from "file.fg"`), `cascade` (defer).

---

## The Toolchain

| Command | Description |
|---|---|
| `forge compile file.fg` | Compile `.fg` to `.go` |
| `forge verify file.forge` | Check `.forge` against Go source for structural drift |
| `forge update file.forge` | Auto-add missing exported symbols, regenerate function index |
| `forge update --prune file.forge` | Also remove stale declarations not in Go source |
| `forge gen pkg/dir/` | Scaffold a new `.forge` file from Go source |
| `forge fmt file.forge` | Format `.forge` to canonical style |

---

## Key Principles

- **`.forge` files live next to the code** they describe (`pkg/ast/ast.forge` alongside `pkg/ast/ast.go`)
- **Cross-file concepts only** — data structures, APIs, interfaces. Not single-file implementation details.
- **Adopt the implementation language's conventions** — PascalCase for Go, snake_case for Python
- **AI writes, human reviews** — implement first, write `.forge` after, human validates architecture
- **Completeness checking** — the verifier warns about exported Go symbols not documented in `.forge`
- **Self-referential** — every compiler package has its own `.forge` file, verified by the tool it contains

---

## Project Structure

```
cmd/forge/              Unified CLI (compile, verify, update, gen, fmt)
cmd/forge-compile/      Standalone compiler CLI
cmd/forge-verify/       Standalone verifier CLI
cmd/forge-gen/          Standalone scaffolding CLI
cmd/forge-update/       Standalone update CLI
pkg/ast/               AST node types + ast.forge
pkg/parser/            PEG parser for .forge and .fg files + parser.forge
pkg/checker/           Type checker with inference + checker.forge
pkg/transpiler/        Go code generator + transpiler.forge
pkg/verifier/          Structural drift detector + verifier.forge
testdata/              36 .fg test files (all compile, 33 go-build clean)
testdata/modules/      Module system test files
```

---

## Test Status

233 tests across parser, checker, transpiler, and verifier. 36 `.fg` test files all compile; 33 produce Go that passes `go build` (1 known issue: `typealias.fg`). 6 self-referential `.forge` files verify clean (0 errors, 0 warnings).

```bash
$ go test ./...
ok  github.com/waywardgeek/forge/pkg/checker     0.018s
ok  github.com/waywardgeek/forge/pkg/parser       0.006s
ok  github.com/waywardgeek/forge/pkg/transpiler   0.005s
ok  github.com/waywardgeek/forge/pkg/verifier     0.004s
```

### Known Issues

- `typealias.fg`: optional type alias wrapping generates Go that doesn't build in all cases
- `features.fg`: `go vet` warns about unreachable code after exhaustive match
- i128/i256 types silently downcast to int64/uint64 (math/big support planned)
- No LSP server yet (planned)

---

## Documentation

- [Forge Language Specification](https://coderhapsody.ai/docs/forge-language) — full type system, syntax, and examples
- [Forge-Driven Development](https://coderhapsody.ai/docs/forge-driven-development) — the methodology

---

## License

Apache 2.0 — see [LICENSE](LICENSE).

## Authors

Bill Cox & [CodeRhapsody](https://coderhapsody.ai)

*"forge" is a 60-year-old word from Heinlein's Stranger in a Strange Land meaning deep, complete understanding. We are reclaiming it.*
