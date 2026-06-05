# Multi-Class Interfaces & Relations Design

**Date**: 2026-06-05
**Status**: Design — not yet implemented

## Motivation

Traditional single-class interfaces can't express relationships between types.
A Graph isn't a property of Node or Edge alone — it's a property of how they
relate. Writing graph algorithms (min-cut, Dijkstra, topological sort) requires
either:
- A single concrete graph type (petgraph) — not generic
- C++ concept maps (Boost BGL) — unusable ceremony

Grok solves this with multi-class interfaces: an interface declares methods
across multiple participant types, and `impl` blocks wire them to concrete
classes via aliasing.

**Primary use cases**: graphs, doubly-linked lists, hashed relations, trees —
any data structure involving relationships between two or more types.

## Multi-Class Interface Syntax

The `func T.method(self)` syntax binds a method to a specific type parameter.
`self` is always the type before the dot.

```grok
interface Graph<G, N, E> {
    func G.nodes(self) -> gen N
    func G.edges(self) -> gen E
    func N.out_edges(self) -> gen E
    func N.in_edges(self) -> gen E
    func E.source(self) -> N
    func E.target(self) -> N
}
```

### Free Functions

Functions without a type prefix have no receiver:

```grok
interface Graph<G, N, E> {
    // ...methods above...
    func distance(a: N, b: N) -> f64   // no receiver, neither N is privileged
}
```

Use free functions only when there's no natural receiver.

### Default Implementations

Interfaces can provide default implementations — algorithms written once in
terms of the required methods:

```grok
interface Graph<G, N, E> {
    // Required
    func G.nodes(self) -> gen N
    func N.out_edges(self) -> gen E
    func E.target(self) -> N

    // Default — free for all implementors
    func count_nodes(graph: G) -> u64 {
        let mut count: u64 = 0
        for _ in graph.nodes() {
            count = count + 1
        }
        return count
    }
}
```

Dijkstra, min-cut, topological sort can all live as default functions.

## Bare Relational Constraints

Instead of `where G: Graph<G, N, E>` (redundant `G:`), use bare constraints:

```grok
func min_cut<G, N, E>(graph: G) -> (f64, ([N], [N]))
    where Graph<G, N, E>, E: Weighted
{
    for node in graph.nodes() {
        for edge in node.out_edges() {
            let w = edge.weight()
            let t = edge.target()
            // ...Stoer-Wagner
        }
    }
}
```

`where Graph<G, N, E>` asserts that G, N, E together satisfy Graph. This
grants all `func N.*` methods to values of type N, all `func E.*` methods to
values of type E, etc. The compiler resolves this from the interface declaration.

## impl Blocks with Aliasing

`impl` blocks map interface methods to concrete class methods/fields using
three forms:

### `=` — Method alias

```grok
impl Graph<CircuitGraph, Component, Wire> {
    G.nodes    = CircuitGraph.components
    G.edges    = CircuitGraph.wires
    N.out_edges = Component.outWires
    N.in_edges  = Component.inWires
    E.source   = Wire.driver
    E.target   = Wire.receiver
}
```

### `<->` — Field binding (generates getter + setter)

```grok
impl DoublyLinked<Folder, File> {
    P.first  <-> Folder.firstFile
    P.last   <-> Folder.lastFile
    C.parent <-> File.folder
    C.next   <-> File.nextFile
    C.prev   <-> File.prevFile
}
```

`<->` is shorthand: `P.first <-> Folder.firstFile` generates both
`P.first(self) -> C?` (getter) and `P.set_first(mut self, c: C?)` (setter)
mapped to the `firstFile` field.

### `{ body }` — Inline implementation

```grok
impl Hashed<Graph, Edge, (Node, Node)> {
    // ...other aliases...
    C.key(self) -> (Node, Node) {
        return (self.source, self.target)
    }
}
```

Use inline implementations when the mapping requires computation, not just
field access.

## Generators

`gen T` is a generator return type — lazy iteration via `yield`:

```grok
func G.nodes(self) -> gen N {
    let mut node = self.firstNode
    while !isnull(node) {
        yield node!
        node = node!.nextNode
    }
}
```

`for..in` works on both `[T]` (slices) and `gen T` (generators).

### Implementation Strategy

Under the hood, generators compile to an `Iterator<T>` state machine:

```grok
interface Iterator<T> {
    func next(mut self) -> T?
}
```

- `gen T` function → compiler-generated struct implementing `Iterator<T>`
- `[T]` automatically satisfies `Iterator<T>` (index-based)
- `for x in expr` desugars to calling `.next()` until null

## Full Examples

### DoublyLinked

```grok
interface DoublyLinked<P, C> {
    // Getters
    func P.first(self) -> C?
    func P.last(self) -> C?
    func C.parent(self) -> P?
    func C.next(self) -> C?
    func C.prev(self) -> C?

    // Setters
    func P.set_first(mut self, child: C?)
    func P.set_last(mut self, child: C?)
    func C.set_parent(mut self, parent: P?)
    func C.set_next(mut self, next: C?)
    func C.set_prev(mut self, prev: C?)

    // Default: append child to end of parent's list
    func append(parent: mut P, child: mut C) {
        match parent.last() {
            null -> {
                parent.set_first(child)
                parent.set_last(child)
                child.set_parent(parent)
                child.set_next(null)
                child.set_prev(null)
            }
            last -> insert_after(parent, last, child)
        }
    }

    // Default: insert new_child after existing
    func insert_after(parent: mut P, existing: mut C, new_child: mut C) {
        let old_next = existing.next()
        existing.set_next(new_child)
        new_child.set_prev(existing)
        new_child.set_next(old_next)
        new_child.set_parent(parent)
        match old_next {
            null -> parent.set_last(new_child)
            c    -> c.set_prev(new_child)
        }
    }

    // Default: remove child from parent's list
    func remove(parent: mut P, child: mut C) {
        match child.prev() {
            null -> parent.set_first(child.next())
            p    -> p.set_next(child.next())
        }
        match child.next() {
            null -> parent.set_last(child.prev())
            n    -> n.set_prev(child.prev())
        }
        child.set_parent(null)
        child.set_next(null)
        child.set_prev(null)
    }

    // Default: iterate children
    func children(parent: P) -> gen C {
        let mut child = parent.first()
        while !isnull(child) {
            let c = child!
            child = c.next()
            yield c
        }
    }
}
```

Usage with field bindings:

```grok
impl DoublyLinked<Folder, File> {
    P.first  <-> Folder.firstFile
    P.last   <-> Folder.lastFile
    C.parent <-> File.folder
    C.next   <-> File.nextFile
    C.prev   <-> File.prevFile
}

// Now append, remove, insert_after, children all work on Folder/File
let folder = Folder.new("src")
let f1 = File.new("main.gk")
let f2 = File.new("lib.gk")
append(folder, f1)
append(folder, f2)

for file in children(folder) {
    println(file.name)
}
```

### Hashed Relations

```grok
interface Hashed<P, C, K> where K: Hashable {
    func P.lookup(self, key: K) -> C?
    func P.insert(mut self, child: C)
    func P.remove(mut self, key: K) -> C?
    func P.contains(self, key: K) -> bool
    func P.entries(self) -> gen (K, C)
    func C.key(self) -> K
}
```

Usage:

```grok
impl Hashed<Registry, Handler, string> {
    P.lookup   = Registry.findHandler
    P.insert   = Registry.addHandler
    P.remove   = Registry.removeHandler
    P.contains = Registry.hasHandler
    P.entries  = Registry.allHandlers
    C.key      = Handler.name
}
```

### Graph with Generic Algorithms

```grok
interface Weighted {
    func weight(self) -> f64
}

interface Graph<G, N, E> {
    func G.nodes(self) -> gen N
    func G.edges(self) -> gen E
    func N.out_edges(self) -> gen E
    func N.in_edges(self) -> gen E
    func E.source(self) -> N
    func E.target(self) -> N
}

func dijkstra<G, N, E>(graph: G, start: N) -> Map<N, f64>
    where Graph<G, N, E>, E: Weighted, N: Hashable
{
    // Written once, works on any graph
    for node in graph.nodes() {
        for edge in node.out_edges() {
            let w = edge.weight()
            let t = edge.target()
            // ...relaxation
        }
    }
}

func min_cut<G, N, E>(graph: G) -> (f64, ([N], [N]))
    where Graph<G, N, E>, E: Weighted
{
    // Stoer-Wagner, written once
}

func topological_sort<G, N, E>(graph: G) -> [N]
    where Graph<G, N, E>
{
    // Kahn's algorithm, written once
}
```

Concrete implementation:

```grok
impl Graph<CircuitGraph, Component, Wire> {
    G.nodes     = CircuitGraph.components
    G.edges     = CircuitGraph.wires
    N.out_edges = Component.outWires
    N.in_edges  = Component.inWires
    E.source    = Wire.driver
    E.target    = Wire.receiver
}

let circuit = CircuitGraph.load("cpu.spice")
let order = topological_sort(circuit)
let (cut_val, (partA, partB)) = min_cut(circuit)
```

## Named impl Blocks (Disambiguation)

When a type participates in multiple relations of the same interface with
identical type signatures, `impl` blocks need labels to disambiguate:

```grok
impl Hashed<UserStore, User, string> as byEmail {
    P.lookup   = UserStore.findByEmail
    P.insert   = UserStore.addByEmail
    P.remove   = UserStore.removeByEmail
    P.contains = UserStore.hasEmail
    P.entries  = UserStore.emailEntries
    C.key      = User.email
}

impl Hashed<UserStore, User, string> as byUsername {
    P.lookup   = UserStore.findByUsername
    P.insert   = UserStore.addByUsername
    P.remove   = UserStore.removeByUsername
    P.contains = UserStore.hasUsername
    P.entries  = UserStore.usernameEntries
    C.key      = User.username
}
```

At call sites, qualify with the label:

```grok
byEmail.insert(store, user)
byUsername.insert(store, user)

let found = byEmail.lookup(store, "bob@example.com")
let also  = byUsername.lookup(store, "bob")
```

**Labels are only required when ambiguous** — two impls of the same interface
with the same type parameters. Most impls won't need them.

## Uniform Function Call Syntax (Future)

`C.foo(args)` and `foo(self: C, args)` could be treated as equivalent — the
compiler wouldn't distinguish method calls from free function calls where the
first argument matches. Not required for initial implementation but the design
is compatible. Don't write code that cares about the distinction.

## Implementation Plan

### Phase 1: Generators
- Add `gen T` type to AST, parser, checker
- Add `yield` statement to AST, parser, checker
- Transpiler: `gen T` → Go iterator pattern (or channel-based)
- LIR: generator state machine lowering
- C backend: struct-based state machine

### Phase 2: Multi-Class Interface Declarations
- Extend `InterfaceDecl` AST to support `func T.method(self)` syntax
- Parser: `func` + identifier + `.` + identifier = typed method
- Checker: validate type params match interface type params
- Store per-type method maps on interface declaration

### Phase 3: impl Blocks
- New `ImplBlock` AST node: `impl Interface<T1, T2, ...> { mappings }`
- Parser: `=` (alias), `<->` (field binding), `{ body }` (inline)
- Checker: validate mapping signatures match interface declarations
- Registry: store impl blocks for constraint resolution

### Phase 4: Bare Relational Constraints
- Extend `where` clause parser for bare `Interface<T1, T2, T3>` (no `:`)
- Constraint solver: when `where Graph<G, N, E>`, grant all `func N.*`
  methods to type param N, etc.
- Lookup: find matching `impl` block to resolve concrete types at call sites

### Phase 5: Default Implementations
- Allow function bodies in interface declarations
- Transpiler: emit as standalone generic functions parameterized by interface types
- At call sites, resolve to the default unless overridden in impl block

### Phase 6: Transpiler & Backend Support
- Go backend: multi-class interfaces → multiple Go interfaces + glue
- C backend: vtable structs per interface, dispatch functions
- LIR: interface dispatch lowering

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Method binding syntax | `func T.method(self)` | Reading order matches reasoning — receiver type first |
| Constraint syntax | `where Graph<G, N, E>` | Bare relational, no redundant `G:` prefix |
| Field binding | `<->` operator | Generates getter+setter, cuts boilerplate in half |
| Method alias | `=` operator | Simple, distinct from field binding |
| Generators | `gen T` + `yield` | Lazy iteration, no allocation, natural for linked structures |
| Free functions in interfaces | `func name(args)` (no type prefix) | For operations with no natural receiver |
| UFCS | Deferred | Design is compatible; don't write code that distinguishes |
