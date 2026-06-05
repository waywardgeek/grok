# Forge Module System — Design

*2026-06-04*

## Problem

Currently, `import X from "path"` only works for Go standard library packages. The checker registers the alias as `TyUnknown` and the transpiler passes the import through to Go. There's no support for importing types/functions from other `.fg` files.

## Design

### Syntax (unchanged)

```forge
import math from "mylib/math.fg"

let result = math.add(1, 2)
let p = math.Point { x: 10, y: 20 }
```

The existing `import alias from "path"` syntax works. The compiler detects `.fg` imports by file extension.

### Compilation Model

Each `.fg` file becomes a separate Go package. The compiler:

1. **Resolves** import paths relative to the importing file's directory
2. **Parses + checks** imported files recursively (with cycle detection)
3. **Exports** public symbols (types, functions) from the imported file into the importer's checker scope under the alias
4. **Transpiles** each file to its own Go package directory

### Key Decisions

- **File = package**: Each `.fg` file is one package. The Go package name = the forge block name (lowercased).
- **Only `pub` symbols exported**: Private symbols in imported files are invisible to importers.
- **Go import path**: The transpiler maps `.fg` import paths to Go module-relative paths. Requires a `-mod` flag for the Go module path prefix.
- **Cycle detection**: Track files being processed; error on cycles.
- **Shared checker state**: The checker needs a `ModuleRegistry` that caches checked modules so the same file isn't checked twice.

### CLI Changes

```
forge compile -mod github.com/user/project src/main.fg -o out/
```

- `-mod`: Go module path prefix (required for multi-file projects)
- `-o`: Output directory (each .fg becomes a subdirectory with package.go)

For single-file compilation (no .fg imports), behavior is unchanged.

### Checker Changes

1. New `ModuleRegistry` on Checker: maps file paths to exported `map[string]*Type`
2. When encountering an `ImportDecl` with `.fg` path:
   - Resolve absolute path relative to current file
   - Check cycle (error if file is in-progress)
   - If already checked, reuse cached exports
   - Otherwise: parse → check → extract pub symbols → cache
3. Register imported symbols as `alias.Name` in scope, or register alias as a "module type" with field access resolving to module exports

### Transpiler Changes

1. `.fg` imports → Go import with module-path-based package path
2. Qualified references (`math.Point`) already work for Go packages; same mechanism for Forge modules

### Directory Layout

```
src/
  main.fg          → out/main/main.go (package main)
  math/lib.fg      → out/math/lib/lib.go (package lib)
```

## Implementation Plan

1. Add `ModuleRegistry` to checker with cycle detection
2. Modify `checkBlock` import handling to parse/check .fg files
3. Add `-mod` flag to CLI
4. Modify transpiler to emit correct Go import paths for .fg imports
5. Write multi-file test case
6. Update .forge files and language spec
