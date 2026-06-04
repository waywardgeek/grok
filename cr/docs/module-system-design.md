# Grok Module System — Design

*2026-06-04*

## Problem

Currently, `import X from "path"` only works for Go standard library packages. The checker registers the alias as `TyUnknown` and the transpiler passes the import through to Go. There's no support for importing types/functions from other `.gk` files.

## Design

### Syntax (unchanged)

```grok
import math from "mylib/math.gk"

let result = math.add(1, 2)
let p = math.Point { x: 10, y: 20 }
```

The existing `import alias from "path"` syntax works. The compiler detects `.gk` imports by file extension.

### Compilation Model

Each `.gk` file becomes a separate Go package. The compiler:

1. **Resolves** import paths relative to the importing file's directory
2. **Parses + checks** imported files recursively (with cycle detection)
3. **Exports** public symbols (types, functions) from the imported file into the importer's checker scope under the alias
4. **Transpiles** each file to its own Go package directory

### Key Decisions

- **File = package**: Each `.gk` file is one package. The Go package name = the grok block name (lowercased).
- **Only `pub` symbols exported**: Private symbols in imported files are invisible to importers.
- **Go import path**: The transpiler maps `.gk` import paths to Go module-relative paths. Requires a `-mod` flag for the Go module path prefix.
- **Cycle detection**: Track files being processed; error on cycles.
- **Shared checker state**: The checker needs a `ModuleRegistry` that caches checked modules so the same file isn't checked twice.

### CLI Changes

```
grok compile -mod github.com/user/project src/main.gk -o out/
```

- `-mod`: Go module path prefix (required for multi-file projects)
- `-o`: Output directory (each .gk becomes a subdirectory with package.go)

For single-file compilation (no .gk imports), behavior is unchanged.

### Checker Changes

1. New `ModuleRegistry` on Checker: maps file paths to exported `map[string]*Type`
2. When encountering an `ImportDecl` with `.gk` path:
   - Resolve absolute path relative to current file
   - Check cycle (error if file is in-progress)
   - If already checked, reuse cached exports
   - Otherwise: parse → check → extract pub symbols → cache
3. Register imported symbols as `alias.Name` in scope, or register alias as a "module type" with field access resolving to module exports

### Transpiler Changes

1. `.gk` imports → Go import with module-path-based package path
2. Qualified references (`math.Point`) already work for Go packages; same mechanism for Grok modules

### Directory Layout

```
src/
  main.gk          → out/main/main.go (package main)
  math/lib.gk      → out/math/lib/lib.go (package lib)
```

## Implementation Plan

1. Add `ModuleRegistry` to checker with cycle detection
2. Modify `checkBlock` import handling to parse/check .gk files
3. Add `-mod` flag to CLI
4. Modify transpiler to emit correct Go import paths for .gk imports
5. Write multi-file test case
6. Update .grok files and language spec
