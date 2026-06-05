### USER (gemini)

# Task: Expert Module Analysis

Create or improve `runtime/design.md` — the expert-level design documentation for the **runtime** module.

## Module Context

- Name: runtime
- Path: runtime
- Output: `runtime/design.md`
- Files: 1 source files

## Files to Analyze

- runtime/forge_runtime.h

## Your Strategy

1. **Read Existing Design**: Read `runtime/design.md` if it exists. This is your starting point — it may contain valuable architectural insights from prior runs that you MUST preserve. Also read `project-design.md` at the project root for system-wide context.
2. **Analyze Code**: Read ALL source files in this module. Identify internal dependencies and read their `design.md` files to understand the interfaces you consume.
3. **Cross-Link**: In your documentation, include **clickable markdown links** to the `design.md` files of any modules this package depends on (e.g., `[pkg/database](../database/design.md)`).
4. **Map Interfaces**: Explicitly identify which interfaces this module implements (e.g., check pkg/database/interfaces.go).
5. **Verify and Update**: Check that all information in the existing doc is still accurate against the current source files. Fix any bit-rot — but **do not drop design insights that are still valid**. Enhance, don't replace.
6. **Document Deeply**: Create a technical narrative that passes the Reimplementation Test. Ensure EVERY non-test source file is accounted for in your File Inventory, and **link to each file** (e.g., `[main.go](main.go)`).
7. **Write Output**: Use `write_file` to create `runtime/design.md`. Use `edit_file` to append additional sections if needed to stay under output token limits.

When finished, confirm: 'Documentation complete.'


---

### ASSISTANT (gemini)

**Tool Call: `list_directory`**
```json
{
  "directory": "."
}
```

**Tool Call: `list_directory`**
```json
{
  "directory": "runtime"
}
```

---

### USER (gemini)

**Tool Result:**
```
📁 **Directory Listing: /Users/bill/projects/forge**

📁 cmd
📁 cr
📁 dg
📁 pkg
📁 runtime
📁 testdata
📄 LICENSE (10.5 KB)
📄 README.md (8.0 KB)
📄 forge (4.5 MB)
📄 forge.forge (8.3 KB)
📄 go.mod (47 B)
📄 hello.go (582 B)
📄 project-design.md (11.9 KB)

**Summary:** 13 items (7 files, 6 directories)
**Total Size:** 4.50 MB
**Hidden Items (excluded):** 2
```

**Tool Result:**
```
📁 **Directory Listing: /Users/bill/projects/forge/runtime**

📄 design.md (5.5 KB)
📄 forge_runtime.h (8.4 KB)

**Summary:** 2 items (2 files, 0 directories)
**Total Size:** 0.01 MB
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "project-design.md"
}
```

**Tool Call: `read_file`**
```json
{
  "file_path": "runtime/design.md"
}
```

**Tool Call: `read_file`**
```json
{
  "file_path": "runtime/forge_runtime.h"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_36c0d04d1f794a4b]
File: /Users/bill/projects/forge/project-design.md (71 lines)

# Forge Project Architecture

## Executive Summary

The Forge project is a comprehensive compiler and toolchain for the Forge programming language. It is designed to provide a high-level, expressive language with advanced features such as relations, interfaces, generic type inference, and structured control flow, while maintaining the capability to compile down to multiple backends, currently supporting Go and C. The project exists to bridge the critical gap between high-level architectural intent and low-level implementation. By capturing design specifications in dedicated `.forge` files, the system ensures structural integrity and actively prevents architectural drift. Its primary goals are to deliver a robust, multi-stage compilation pipeline—spanning parsing, semantic analysis, low-level intermediate representation, and code generation—while maintaining a strong, verifiable connection between the documented design and the evolving codebase through its unique verification engine.

## System Architecture

The Forge compiler follows a classic pipeline architecture, augmented by a parallel verification track that acts as an architectural guardian. The architecture is fundamentally layered and data-centric, designed to isolate complexity at each distinct stage of transformation. The primary compilation flow is linear, progressing from raw source text to an Abstract Syntax Tree (AST), through semantic checking, into a Low-level Intermediate Representation (LIR), and finally to target code generation.

The philosophy of the system is to resolve specific classes of problems in dedicated modules. The parser handles all syntax and grammatical ambiguities, producing a structural representation. The AST module provides a clean, desugared representation of the program, expanding high-level syntactic sugar into fundamental constructs. The checker assumes responsibility for all semantic rules, performing type inference, scope management, and validation, ultimately annotating the AST with resolved types. The LIR module then takes this semantically rich tree and flattens it into a simplified, SSA-like sequence of operations with structured control flow. This crucial step removes the burden of semantic resolution from the backends. Finally, the backends act as thin emitters that translate the simplified LIR into idiomatic target source code.

In parallel to the compilation pipeline, the Verifier module operates as a standalone structural integrity engine. It parses `.forge` specification files and performs a deep structural comparison against the actual Go implementation. This ensures that the documented design and the codebase remain perfectly synchronized, enforcing the philosophy that documentation should be an active, verifiable contract rather than a passive artifact.


## Interface & Contract Map

The system's boundaries are primarily defined by the rich data structures passed between modules rather than traditional Go interfaces. These data structures act as the fundamental contracts that govern module interactions.

The most critical contract is the Abstract Syntax Tree. The `ast.File` and its constituent nodes representing declarations, expressions, and statements form the central nervous system of the compiler. The parser module acts as the initial producer of this contract, transforming source code into the raw AST. The AST module itself acts as both a consumer and producer, taking the raw AST and producing a desugared version. The checker module consumes this desugared AST for semantic analysis and mutates it by attaching resolved type information. The LIR module consumes the fully type-checked AST to generate the intermediate representation. Additionally, the verifier module consumes the AST of `.forge` files to compare against Go source code.

The Type Registry contract, defined by the `checker.Registry` and `checker.Type` structures, represents the semantic truth of the program. The checker module builds and populates this registry during its declaration and checking passes. The LIR module is the primary consumer of this contract, relying heavily on the type registry and resolved types to correctly lower high-level constructs, such as generics, classes, and enums, into low-level operations.

The LIR contract, represented by the `lir.LProgram`, defines the fully resolved and flattened program state. The LIR module's lowerer produces this representation. Internal optimization and monomorphization passes within the LIR module consume and mutate this structure. Ultimately, the Go and C backends consume the optimized `LProgram` to emit the final source code.

The Runtime contract is defined by the C runtime header, which establishes the execution environment for C-compiled Forge programs. The runtime module provides these foundational macros and functions. The C backend within the LIR module acts as the consumer, emitting code that relies entirely on this contract for memory layout and operational semantics.

## Module Map

### Frontend
- [pkg/parser](pkg/parser/design.md)
- [pkg/ast](pkg/ast/design.md)

### Semantic Analysis
- [pkg/checker](pkg/checker/design.md)

### Intermediate Representation & Backends
- [pkg/lir](pkg/lir/design.md)
- [runtime](runtime/design.md)

### Tooling
- [pkg/verifier](pkg/verifier/design.md)

The **Parser** module is the entry point for the compiler frontend, responsible for transforming raw Forge source code into an Abstract Syntax Tree. It manages internal state through a hand-written lexer that tokenizes the input, carefully handling significant newlines and collecting comments. The parser employs recursive descent for high-level structures and a Pratt parser for expressions, allowing it to efficiently handle complex operator precedence. It implements the primary data contract by producing the `ast.File` consumed by the rest of the system.

The **AST** module defines the core data structures that represent the structure of Forge code, acting as the central data hub. It manages state primarily as a tree of polymorphic nodes utilizing a "Kind + Data" pattern to avoid deep interface hierarchies. Crucially, it includes a desugaring phase that transforms high-level constructs, such as relations and interface fields, into fundamental AST nodes before semantic analysis occurs. It implements the primary data contract consumed by the checker and LIR modules.

The **Checker** module implements the semantic analysis phase of the compiler. It is responsible for type checking, generic type inference, name resolution, and enforcing all language constraints. It maintains complex internal state, including a stack of scopes for variable tracking and a global registry for user-defined types. It consumes the desugared AST, resolves all type expressions, and annotates the AST nodes with `ResolvedType` information, fulfilling the semantic contract required by the subsequent lowering phase.

The **LIR** module bridges the high-level AST and the backend code generators by resolving all semantic complexity into a simplified, flat representation with structured control flow. It manages state during the lowering process by mapping AST nodes to LIR constructs and assigning all sub-expressions to unique temporary variables. It also includes sophisticated optimization and monomorphization passes. It consumes the type-checked AST and produces the `LProgram` contract, which is then consumed by its internal Go and C backends to emit target code.

The **Verifier** module is a standalone structural integrity engine designed to detect architectural drift. It compares high-level `.forge` specifications against the actual Go implementation. It maintains state during a run to aggregate type information extracted from Go source files using the standard library's AST tools. It consumes the AST produced by the parser for `.forge` files and performs deep structural comparisons, reporting discrepancies to ensure the specification remains a complete and accurate model of the codebase.

The **Runtime** module provides the minimal C runtime library required by the Forge C backend. It is a passive, header-only library that defines the memory layout and operational semantics for Forge constructs like dynamic slices, optional types, and error results in a C environment. It does not maintain internal state but provides the foundational macros and functions consumed by the C code generated by the LIR module.

## Integration Patterns & Workflows

The system relies on clear, sequential workflows to transform data across module boundaries. Two primary workflows define the operation of the Forge toolchain: the compilation pipeline and the architectural verification process.

The compilation pipeline workflow traces the journey of a Forge source file from raw text to executable code. The process begins when a file is read from disk and passed to the parser module. The lexer tokenizes the text, and the parser constructs an initial `ast.File`. This AST is immediately passed to the AST module's desugaring functions, which expand complex features like relations into standard classes and implementation blocks. The desugared AST is then handed to the checker module. The checker performs a declaration pass to populate its type registry, followed by a deep traversal to resolve types, infer generics, and validate semantics, annotating the AST with `ResolvedType` fields. Once validated, the AST and the type registry are passed to the LIR module's lowerer. The lowerer flattens expressions into temporaries and converts high-level constructs into LIR statements. The resulting `LProgram` undergoes optimization passes to remove redundant operations. If the target is C, a monomorphization pass specializes generic types. Finally, the backend emitter traverses the LIR and generates the final source code string, which is written to disk.

The architectural verification workflow operates independently to ensure design fidelity. The process starts when the verifier module is invoked on a `.forge` specification file. It uses the parser module to generate an AST of the specification. The verifier then inspects the source annotations within the AST to locate the corresponding Go source files. It uses the Go standard library to extract all exported types, interfaces, and function signatures from these Go files, building a comprehensive internal registry. The verifier then recursively walks the Forge AST, translating Forge type expressions into Go string representations and comparing them against the extracted Go types. It normalizes naming conventions on the fly to facilitate this comparison. Finally, it performs a reverse completeness check, ensuring every exported Go symbol is documented in the Forge specification, and aggregates any mismatches into a detailed report.

## Dependency Overview

The project exhibits a strict, unidirectional dependency graph that prevents circular imports and enforces a clear separation of concerns. This layering is critical for maintaining the maintainability of the compiler.

At the base of the hierarchy is the AST module, which acts as a leaf node containing pure data structures. The parser module depends solely on the AST module to construct its output, ensuring that parsing logic is decoupled from semantic analysis. The checker module depends on both the AST module for input and annotation, and the parser module to handle imported files during its semantic analysis phase. The LIR module depends on the AST and checker modules to perform its lowering process, relying on the semantic truth established by the checker. 

The verifier module depends on the AST and parser modules to read specifications, but operates entirely independently of the compiler's semantic and backend phases. This isolation allows the verifier to act as an objective observer of the codebase. The runtime module is completely independent of all Go code, serving only as a target dependency for the generated C code. This layered architecture ensures that the frontend remains decoupled from backend concerns, and that data structures are defined independently of the logic that manipulates them.
```

**Tool Result:**
```
[ID: tool_6cfd65a981be2981]
File: /Users/bill/projects/forge/runtime/design.md (84 lines)

# Runtime Module Design

## Executive Summary

The `runtime` module provides the minimal C runtime library required by the Forge compiler's C backend. It is implemented as a header-only library ([forge_runtime.h](forge_runtime.h)) that provides the low-level primitives necessary to support Forge's high-level features in a C environment. This includes dynamic slices, optional types, error results, string manipulation, and tagged unions. By providing these abstractions, the runtime enables the C backend to emit code that is both readable and semantically equivalent to the original Forge source.

## File Inventory

- [forge_runtime.h](forge_runtime.h): The core runtime header. It contains all macro definitions and inline functions for the C-based Forge execution environment.

## Architecture and Data Flow

The `runtime` module is a passive component that provides a set of types and utilities used by the code generated by the [LIR module](../pkg/lir/design.md). It does not maintain internal state or have an active execution flow of its own. Instead, it defines the memory layout and operational semantics for Forge constructs when compiled to C.

The architecture is centered around C macros that allow for a form of "template-based" type generation. Since C lacks generics, the Forge C backend uses these macros to define concrete struct types for every unique instantiation of a generic Forge type (e.g., `Slice_int32_t`, `Opt_string`).

## Interface Implementations

As a C header library, this module does not implement any Go interfaces. It serves as a target for the C code generator in the [LIR module](../pkg/lir/design.md).

## Public API

The "Public API" of the runtime consists of the macros and functions intended to be called by Forge-generated C code.

### Dynamic Slices
Forge slices are implemented as dynamic arrays with automatic resizing.
- `FORGE_SLICE_DEF(ElemType, SliceName)`: Defines a struct for a slice of a specific type.
- `forge_slice_empty(SliceName)`: Initializes an empty slice.
- `forge_push(slice_ptr, elem, SliceName)`: Appends an element, growing the capacity by 2x if necessary.
- `forge_pop(slice_ptr)`: Removes and returns the last element.
- `forge_slice_lit(SliceName, ElemType, ...)`: Creates a slice from a C initializer list.
- `forge_contains(slice, elem)`: Performs a linear search for an element.

### Optional Types
Optionals (`T?`) are implemented as a struct containing a boolean flag and the value.
- `FORGE_OPT_DEF(ElemType, OptName)`: Defines an optional type.
- `forge_some(value, OptName)`: Creates an optional containing a value.
- `forge_none(OptName)`: Creates an empty optional.
- `forge_isnull(opt)`: Checks if the optional is empty.
- `forge_unwrap(opt)`: Returns the value (caller must ensure it is not null).

### Error Results
Forge's `(T, error)` return pattern is mapped to a result struct.
- `FORGE_RESULT_DEF(ElemType, ResultName)`: Defines a result type.
- `forge_ok(val, ResultName)`: Creates a successful result.
- `forge_err(msg, ResultName)`: Creates an error result with a message.
- `forge_is_err(r)`: Checks if the result represents an error.

### String Helpers
A collection of inline functions for high-level string operations.
- `forge_str_contains`, `forge_str_index_of`: Search operations.
- `forge_str_has_prefix`, `forge_str_has_suffix`: Prefix/suffix checks.
- `forge_str_replace`, `forge_str_repeat`, `forge_str_join`: String transformation and construction.
- `forge_sprintf`: A heap-allocating version of `sprintf`.

### Tagged Unions
Used for ad-hoc union types (e.g., `string | i32`).
- `ForgeUnion`: A generic tagged union struct.
- `forge_union_i32`, `forge_union_string`, etc.: Constructor functions for different types.

## Implementation Details

### Memory Management
The runtime currently relies on standard C `malloc` and `realloc`. A significant implementation detail is that the runtime does not yet include a garbage collector. Consequently, heap-allocated objects—including slices, strings created by helpers, and class instances—are currently leaked. This is a known limitation of the current C backend implementation.

### Slice Growth
The `forge_push` macro implements a simple geometric growth strategy. When the length reaches capacity, the capacity is doubled (starting from 4 if the slice was empty). This ensures amortized O(1) push operations.

### String Handling
Strings in the Forge C runtime are represented as standard null-terminated `const char*`. The helper functions are designed to mimic the behavior of Forge's built-in string methods, often allocating new memory for the result to ensure immutability-like behavior.

## Dependencies

- **Standard C Library**: The runtime depends on `<stdio.h>`, `<stdlib.h>`, `<stdint.h>`, `<stdbool.h>`, `<string.h>`, and `<stdarg.h>`.
- **LIR Module**: This module is a critical dependency for the C backend in [pkg/lir](../pkg/lir/design.md).

## Technical Debt and Future Work

- **Garbage Collection**: Implementing or integrating a garbage collector (e.g., Boehm-Demers-Weiser) is the highest priority for making the C backend viable for long-running programs.
- **Concurrency Support**: The C runtime currently lacks support for Forge's concurrency primitives (`spawn`, channels, `select`).
- **Map Implementation**: A hash map implementation is needed to support Forge's `map<K, V>` type in C.
- **Error Handling Refinement**: The current error result uses a `const char*` for the error message. This should be expanded to support more complex error types if Forge's error system evolves.
- **Thread Safety**: The current runtime primitives (especially slice operations) are not thread-safe.
```

**Tool Result:**
```
[ID: tool_bf51fe18b31d75c3]
File: /Users/bill/projects/forge/runtime/forge_runtime.h (240 lines)

/* forge_runtime.h — Minimal C runtime for Forge-compiled programs.
 *
 * Provides macros for:
 *   - Dynamic slices (FORGE_SLICE_DEF, forge_push, forge_pop, forge_slice_lit)
 *   - Optionals (FORGE_OPT_DEF, forge_some, forge_none, forge_isnull)
 *   - Error results (FORGE_RESULT_DEF, forge_ok, forge_err)
 *   - String helpers (forge_contains, forge_index_of, forge_replace, forge_join, etc.)
 *   - Formatting (forge_sprintf)
 */

#ifndef FORGE_RUNTIME_H
#define FORGE_RUNTIME_H

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <string.h>

/* -------------------------------------------------------------------------
 * Dynamic Slices
 * -------------------------------------------------------------------------
 * Usage:
 *   FORGE_SLICE_DEF(int32_t, ForgeSlice_int32_t)
 *   ForgeSlice_int32_t xs = forge_slice_empty(ForgeSlice_int32_t);
 *   forge_push(&xs, 42, ForgeSlice_int32_t);
 *   int32_t val = xs.data[0];
 */

#define FORGE_SLICE_DEF(ElemType, SliceName) \
    typedef struct { ElemType* data; int32_t len; int32_t cap; } SliceName;

/* Create an empty slice */
#define forge_slice_empty(SliceName) ((SliceName){.data = NULL, .len = 0, .cap = 0})

/* Push an element (grows by 2x when full) */
#define forge_push(slice_ptr, elem, SliceName) do { \
    if ((slice_ptr)->len >= (slice_ptr)->cap) { \
        int32_t _newcap = (slice_ptr)->cap == 0 ? 4 : (slice_ptr)->cap * 2; \
        (slice_ptr)->data = realloc((slice_ptr)->data, sizeof(*(slice_ptr)->data) * _newcap); \
        (slice_ptr)->cap = _newcap; \
    } \
    (slice_ptr)->data[(slice_ptr)->len++] = (elem); \
} while(0)

/* Pop the last element (returns it). Caller must check len > 0. */
#define forge_pop(slice_ptr) ((slice_ptr)->data[--(slice_ptr)->len])

/* Create a slice from an initializer list */
#define forge_slice_lit(SliceName, ElemType, ...) ({ \
    ElemType _init[] = {__VA_ARGS__}; \
    int32_t _n = sizeof(_init) / sizeof(_init[0]); \
    SliceName _s; \
    _s.data = malloc(sizeof(ElemType) * _n); \
    memcpy(_s.data, _init, sizeof(ElemType) * _n); \
    _s.len = _n; \
    _s.cap = _n; \
    _s; \
})

/* Slice contains an element (linear scan) */
#define forge_contains(slice, elem) ({ \
    bool _found = false; \
    for (int32_t _i = 0; _i < (slice).len; _i++) { \
        if ((slice).data[_i] == (elem)) { _found = true; break; } \
    } \
    _found; \
})

/* -------------------------------------------------------------------------
 * Optionals  —  {bool has; T val}
 * -------------------------------------------------------------------------
 * Usage:
 *   FORGE_OPT_DEF(int32_t, ForgeOpt_int32_t)
 *   ForgeOpt_int32_t x = forge_some(42, ForgeOpt_int32_t);
 *   ForgeOpt_int32_t y = forge_none(ForgeOpt_int32_t);
 *   if (!forge_isnull(x)) { use(x.val); }
 */

#define FORGE_OPT_DEF(ElemType, OptName) \
    typedef struct { bool has; ElemType val; } OptName;

#define forge_some(value, OptName) ((OptName){.has = true, .val = (value)})
#define forge_none(OptName) ((OptName){.has = false})
#define forge_isnull(opt) (!(opt).has)
#define forge_unwrap(opt) ((opt).val)

/* -------------------------------------------------------------------------
 * Error Results  —  {bool is_err; T value; const char* error}
 * -------------------------------------------------------------------------
 * Usage:
 *   FORGE_RESULT_DEF(int32_t, ForgeResult_int32_t)
 *   ForgeResult_int32_t r = forge_ok(42, ForgeResult_int32_t);
 *   ForgeResult_int32_t e = forge_err("failed", ForgeResult_int32_t);
 */

#define FORGE_RESULT_DEF(ElemType, ResultName) \
    typedef struct { bool is_err; ElemType value; const char* error; } ResultName;

#define forge_ok(val, ResultName) ((ResultName){.is_err = false, .value = (val), .error = NULL})
#define forge_err(msg, ResultName) ((ResultName){.is_err = true, .error = (msg)})
#define forge_is_err(r) ((r).is_err)

/* -------------------------------------------------------------------------
 * String helpers
 * -------------------------------------------------------------------------
 * These return heap-allocated strings where needed. The Forge C runtime
 * does not yet have a GC — these leak. That's fine for now.
 */

static inline bool forge_str_contains(const char* s, const char* sub) {
    return strstr(s, sub) != NULL;
}

static inline int32_t forge_str_index_of(const char* s, const char* sub) {
    const char* p = strstr(s, sub);
    if (p == NULL) return -1;
    return (int32_t)(p - s);
}

static inline bool forge_str_has_prefix(const char* s, const char* prefix) {
    return strncmp(s, prefix, strlen(prefix)) == 0;
}

static inline bool forge_str_has_suffix(const char* s, const char* suffix) {
    size_t slen = strlen(s), suflen = strlen(suffix);
    if (slen < suflen) return false;
    return strcmp(s + slen - suflen, suffix) == 0;
}

static inline const char* forge_str_replace(const char* s, const char* old, const char* new_s) {
    const char* pos = strstr(s, old);
    if (!pos) {
        char* dup = malloc(strlen(s) + 1);
        strcpy(dup, s);
        return dup;
    }
    size_t oldlen = strlen(old), newlen = strlen(new_s), slen = strlen(s);
    /* Count occurrences */
    int count = 0;
    const char* p = s;
    while ((p = strstr(p, old)) != NULL) { count++; p += oldlen; }
    char* result = malloc(slen + count * (newlen - oldlen) + 1);
    char* dst = result;
    p = s;
    while ((pos = strstr(p, old)) != NULL) {
        memcpy(dst, p, pos - p);
        dst += pos - p;
        memcpy(dst, new_s, newlen);
        dst += newlen;
        p = pos + oldlen;
    }
    strcpy(dst, p);
    return result;
}

static inline const char* forge_str_repeat(const char* s, int32_t n) {
    size_t slen = strlen(s);
    char* result = malloc(slen * n + 1);
    result[0] = '\0';
    for (int32_t i = 0; i < n; i++) {
        strcat(result, s);
    }
    return result;
}

static inline const char* forge_str_join(const char* sep, const char** parts, int32_t count) {
    if (count == 0) {
        char* r = malloc(1);
        r[0] = '\0';
        return r;
    }
    size_t total = 0, seplen = strlen(sep);
    for (int32_t i = 0; i < count; i++) {
        total += strlen(parts[i]);
        if (i > 0) total += seplen;
    }
    char* result = malloc(total + 1);
    result[0] = '\0';
    for (int32_t i = 0; i < count; i++) {
        if (i > 0) strcat(result, sep);
        strcat(result, parts[i]);
    }
    return result;
}

/* forge_sprintf — heap-allocated formatted string */
static inline const char* forge_sprintf(const char* fmt, ...) {
    va_list args, args2;
    va_start(args, fmt);
    va_copy(args2, args);
    int n = vsnprintf(NULL, 0, fmt, args);
    va_end(args);
    char* buf = malloc(n + 1);
    vsnprintf(buf, n + 1, fmt, args2);
    va_end(args2);
    return buf;
}

/* Bool to string for printf */
static inline const char* forge_bool_str(bool b) {
    return b ? "true" : "false";
}

/* -------------------------------------------------------------------------
 * Tagged Unions (for ad-hoc union types like string | i32 | bool)
 * -------------------------------------------------------------------------
 * Tag constants identify which member is active.
 */
#define FORGE_UNION_TAG_I32    0
#define FORGE_UNION_TAG_I64    1
#define FORGE_UNION_TAG_F32    2
#define FORGE_UNION_TAG_F64    3
#define FORGE_UNION_TAG_BOOL   4
#define FORGE_UNION_TAG_STRING 5
#define FORGE_UNION_TAG_PTR    6

typedef struct {
    int tag;
    union {
        int32_t  as_i32;
        int64_t  as_i64;
        float    as_f32;
        double   as_f64;
        bool     as_bool;
        const char* as_string;
        void*    as_ptr;
    } data;
} ForgeUnion;

static inline ForgeUnion forge_union_i32(int32_t v)       { return (ForgeUnion){FORGE_UNION_TAG_I32, {.as_i32 = v}}; }
static inline ForgeUnion forge_union_i64(int64_t v)       { return (ForgeUnion){FORGE_UNION_TAG_I64, {.as_i64 = v}}; }
static inline ForgeUnion forge_union_f32(float v)         { return (ForgeUnion){FORGE_UNION_TAG_F32, {.as_f32 = v}}; }
static inline ForgeUnion forge_union_f64(double v)        { return (ForgeUnion){FORGE_UNION_TAG_F64, {.as_f64 = v}}; }
static inline ForgeUnion forge_union_bool(bool v)         { return (ForgeUnion){FORGE_UNION_TAG_BOOL, {.as_bool = v}}; }
static inline ForgeUnion forge_union_string(const char* v){ return (ForgeUnion){FORGE_UNION_TAG_STRING, {.as_string = v}}; }
static inline ForgeUnion forge_union_ptr(void* v)         { return (ForgeUnion){FORGE_UNION_TAG_PTR, {.as_ptr = v}}; }

#endif /* FORGE_RUNTIME_H */
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "pkg/lir/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_6ccae6f7cc68e140]
File: /Users/bill/projects/forge/pkg/lir/design.md (96 lines)

# LIR Module Design

## Executive Summary

The `lir` (Low-level Intermediate Representation) module serves as the critical bridge between the high-level, semantically rich Abstract Syntax Tree (AST) and the backend-specific code generators in the Forge compiler. Its primary mission is to resolve all semantic complexity of the Forge language into a simplified, flat representation that makes backend implementation a straightforward exercise in syntax mapping. 

The LIR is built on several foundational design principles that distinguish it from traditional compiler IRs. First, it preserves **structured control flow** (if, while, for, switch) rather than decomposing logic into basic blocks and jumps. This preservation is essential for generating idiomatic high-level target code, particularly for the Go backend. Second, it enforces **flat expressions**, where every sub-expression is assigned to a unique, named temporary variable. This SSA-like flattening ensures that backends never have to deal with nested expression trees. Third, it performs **semantic resolution**, lowering high-level Forge features like optional types, tagged unions (enums), method dispatch, and closures into explicit LIR constructs. Finally, the LIR is **backend agnostic**, providing a common foundation for multiple targets while offering optional passes like monomorphization for backends that require it.

## File Inventory

*   [lir.go](lir.go): Defines the core data structures for the LIR, including the type system (`LType`), operands (`LValue`), expressions (`LExpr`), and structured statements (`LStmt`).
*   [lower.go](lower.go): Implements the `Lowerer`, the primary engine that transforms a type-checked AST into an LIR program. This file contains the bulk of the semantic lowering logic.
*   [optimize.go](optimize.go): Performs post-lowering LIR-to-LIR optimizations, such as side-effect temporary elimination, multi-return destructuring, and unused variable removal.
*   [monomorphize.go](monomorphize.go): Implements a monomorphization pass that specializes generic functions, classes, and structs into concrete versions for backends like C that do not support generics natively.
*   [go_backend.go](go_backend.go): The Go code generator, which transforms LIR into idiomatic Go code, leveraging Go's native support for generics, interfaces, and concurrency.
*   [c_backend.go](c_backend.go): The C code generator, which transforms monomorphized LIR into C code, utilizing a dedicated runtime library for features like slices and strings.
*   [lir.forge](lir.forge): A Forge specification file that formally defines the LIR module's structure and interfaces, used by the verifier to ensure implementation integrity.
*   [lower_test.go](lower_test.go): Comprehensive unit tests for the lowering pass, verifying the transformation of various AST constructs.
*   [mono_test.go](mono_test.go): Unit tests for the monomorphization pass, ensuring correct specialization and name mangling.
*   [c_backend_test.go](c_backend_test.go): Tests for the C code generator, verifying the emission of valid C code.
*   [e2e_test.go](e2e_test.go): End-to-end tests that exercise the entire pipeline from lowering through optimization and code generation.
*   [dump_c_test.go](dump_c_test.go): A utility test for dumping generated C code for manual inspection.

## Architecture and Data Flow

The `lir` module operates as a sophisticated transformation pipeline that converts high-level architectural intent into executable source code. The flow of data through the module follows a strictly defined sequence of stages.

The pipeline begins with the **Lowering** stage. The `Lowerer` consumes a type-checked `ast.File` from the [pkg/ast](../ast/design.md) module and a `checker.Registry` from the [pkg/checker](../checker/design.md) module. It performs a recursive walk of the AST, flattening every nested expression into a sequence of single-operation assignments to named temporaries. During this process, high-level constructs are desugared into their LIR equivalents; for instance, a `match` statement on an enum is lowered into a `switch` on an integer variant tag, with explicit field extractions for any bound variables.

Once the initial LIR is generated, it enters the **Optimization** stage. The `Optimize` function runs several passes over the LIR to clean up the output of the lowerer. These passes are essential because the lowerer often produces redundant temporaries or patterns that are difficult for backends to emit efficiently. Key optimizations include fusing void-context calls into bare statements (side-effect elimination) and destructuring multi-return values from single temporaries into separate variables.

For backends that require it, such as C, the pipeline includes an optional **Monomorphization** stage. The `Monomorphize` function scans the LIR program for all instantiations of generic types and functions. It then creates specialized, non-generic copies for each unique set of type arguments and rewrites all call sites and type references to use these specialized versions. This ensures that the final LIR contains no type variables before it reaches the backend.

The final stage is **Code Generation**. A backend-specific emitter, either the Go backend or the C backend, walks the optimized (and potentially monomorphized) LIR tree and produces the final source code string. Because the LIR has already resolved all semantic complexity, these backends are primarily responsible for syntax mapping and utilizing their respective runtime libraries.

## Interface Implementations

The `lir` module does not explicitly implement interfaces defined in other packages, as its primary role is data transformation. However, it is a heavy consumer of the data structures and semantic information provided by:
*   [pkg/ast](../ast/design.md): The module provides the input `ast.File` and all its constituent nodes.
*   [pkg/checker](../checker/design.md): The module provides the `checker.Registry` and the `checker.Type` information used to resolve names and types during the lowering process.

## Public API

The `lir` module exposes a clean, functional API for interacting with the LIR pipeline. The primary entry points are:

*   `NewLowerer()`: This function creates a new instance of the `Lowerer`, initializing the internal state required for AST-to-LIR transformation.
*   `Lowerer.Lower(file *ast.File) *LProgram`: This is the "front door" to the module. It takes a type-checked AST file and returns a complete `LProgram` structure representing the lowered code.
*   `Optimize(prog *LProgram)`: This function runs the full suite of LIR-to-LIR optimization passes on a program, mutating it in place to improve code quality and backend compatibility.
*   `Monomorphize(prog *LProgram)`: This function specializes all generic declarations in the program, removing all type variables. It is a prerequisite for the C backend.
*   `EmitGo(prog *LProgram) string`: This function takes an LIR program and generates a complete Go source code string, handling all necessary imports and formatting.
*   `EmitC(prog *LProgram) string`: This function takes a monomorphized LIR program and generates a complete C source code string, relying on the `forge_runtime.h` contract.

## Implementation Details

### The Lowering Pass

The `Lowerer` is the most complex component of the module, responsible for the heavy lifting of semantic resolution. It maintains a stateful mapping of AST nodes to LIR constructs and tracks visibility, imports, and type information. A central feature of the lowerer is **expression flattening**. Every sub-expression in the AST is assigned to a unique temporary variable (`LValTemp`). This ensures that all `LExpr` instances in the LIR reference only simple operands, never nested expressions. For example, `a + b * c` is lowered into `%0 = b * c` followed by `%1 = a + %0`.

Control flow lowering in Forge LIR is **structured**. Instead of decomposing logic into basic blocks and jumps, the LIR uses high-level statements like `LStmtIf` and `LStmtWhile`. The `LStmtWhile` is particularly noteworthy; it contains a `CondBlock`—a list of statements executed before each iteration to compute the loop condition. This elegantly handles complex conditions that might involve side effects or multiple steps, ensuring they are re-evaluated correctly on every iteration without duplicating code.

The `Lowerer` also handles the desugaring of Forge's high-level types:
*   **Enums**: These are lowered to `LTyTaggedUnion`. Match statements are converted to switches on the variant tag, with the lowerer generating the necessary logic to extract variant fields into local bindings.
*   **Classes**: These are lowered to `LTyClassHandle`, representing a pointer to a heap-allocated struct. Field access is converted to explicit `LExprClassGet` and `LStmtClassSet` operations.
*   **Generics**: The lowerer preserves type parameters as `LTyTypeVar`. This allows the Go backend to emit native Go generics.
*   **Impl Blocks**: The lowerer generates forwarding wrapper methods for `impl` blocks. This enables classes to satisfy interfaces even when method names or signatures do not match exactly, by creating a bridge between the interface's expected signature and the class's actual implementation.
*   **Try Operator**: The `?` operator is lowered into a sequence of operations: extracting the error from a result, checking if it is non-null, and performing an early return if an error is present, followed by extracting the success value.

### The Optimization Pass

The optimizer runs after lowering to refine the LIR. One of its most critical tasks is **side-effect temporary elimination**. The lowerer often assigns the result of every call to a temporary, even if the result is immediately discarded (e.g., in a void-context call). The optimizer identifies these patterns and fuses them into `LStmtSideEffect` statements, which backends can emit as bare expressions.

Another vital optimization is **multi-return destructuring**. Forge allows functions to return multiple values, such as `(T, error)`. The lowerer initially represents this as a single temporary of type `LTyErrorResult`. The optimizer identifies patterns where this temporary is immediately destructured into separate variables and replaces them with a single `LStmtMultiAssign`. This maps directly to Go's multi-assignment syntax and simplifies C's handling of result structures.

### Monomorphization

For backends like C that lack native generic support, monomorphization is mandatory. The pass operates in four distinct phases. First, it indexes all generic declarations in the program. Second, it performs a deep walk of all function bodies to collect every unique instantiation of generic functions, classes, and structs based on the concrete type arguments used at call sites. Third, it generates specialized copies of these declarations, mangling their names (e.g., `Stack_i32`) to ensure uniqueness. Finally, it rewrites all call sites and type references throughout the program to use these specialized, non-generic versions.

### Backends

The **Go Backend** is designed as a "thin" emitter. It leverages Go's native support for generics, interfaces, and concurrency. It uses a `strings.Builder` to accumulate code and maintains a map of required imports, which are automatically added to the file header. It handles Forge-specific features like generators by emitting a goroutine and channel pattern that matches Forge's lazy iteration semantics.

The **C Backend** is significantly more involved due to C's lack of high-level features. It relies on a small runtime header (`forge_runtime.h`) that defines macros and structures for slices, optional types, and error results. It hoists lambdas to top-level static functions and uses `malloc` for class allocations. It also implements a sophisticated type inference mechanism to resolve `LTyAny` types into concrete C types where possible, ensuring type safety in the generated code.

## Dependencies

*   [pkg/ast](../ast/design.md): Provides the source AST and node definitions.
*   [pkg/checker](../checker/design.md): Provides the type registry and resolved type information.
*   [runtime](../../runtime/design.md): Provides the C runtime contract for the C backend.

## Technical Debt and Future Work

*   **Map Support in C**: The C backend currently lacks implementation for map types.
*   **Concurrency in C**: Goroutines and channels are not yet supported in the C backend.
*   **Advanced Optimizations**: Future improvements could include constant folding, dead code elimination beyond unused temporaries, and function inlining.
*   **Relational Constraints**: While basic support exists, more complex scenarios involving multi-class interfaces may require further refinement in the Go backend to ensure perfect compatibility with Go's type system.
```

---

### ASSISTANT (gemini)

**Tool Call: `write_file`**
```json
{
  "content": "# Runtime Module Design\n\n## Executive Summary\n\nThe `runtime` module provides the foundational C runtime library required by the Forge compiler's C backend. It is implemented as a passive, header-only library that establishes the memory layout and operational semantics for Forge's high-level constructs in a C environment. By providing a suite of macros and inline functions, the runtime enables the C backend to transform complex Forge features—such as dynamic slices, optional types, error results, and tagged unions—into efficient and readable C code. Its primary goal is to bridge the gap between Forge's expressive type system and C's low-level primitives while maintaining semantic equivalence.\n\n## File Inventory\n\n- [forge_runtime.h](forge_runtime.h): The core runtime header. It contains all macro definitions, type structures, and inline helper functions that constitute the Forge execution environment for C targets.\n\n## Architecture and Data Flow\n\nThe `runtime` module is a passive component that does not maintain internal state or have an independent execution flow. Instead, it acts as a contract and a toolkit for the code generated by the [LIR module](../pkg/lir/design.md). \n\nThe architecture leverages C macros to implement a form of \"template-based\" type generation. Since C lacks native support for generics, the Forge C backend uses these macros to define concrete, specialized struct types for every unique instantiation of a generic Forge type (e.g., `Slice_int32_t` or `Opt_string`). This approach ensures type safety at the C level and allows the compiler to generate code that is tailored to specific data types. Data flows through these structures as they are passed between functions, with the runtime providing the logic for common operations like resizing slices or wrapping values in optional containers.\n\n## Interface Implementations\n\nAs a C header library, this module does not implement any Go interfaces. It serves as the target execution environment for the C code generator within the [LIR module](../pkg/lir/design.md).\n\n## Public API\n\nThe Public API of the runtime consists of the macros and inline functions intended to be utilized by the Forge-generated C code. These are categorized by the Forge language features they support.\n\n### Dynamic Slices\nForge slices are implemented as dynamic arrays with automatic resizing capabilities.\n- `FORGE_SLICE_DEF(ElemType, SliceName)`: A macro that defines a struct containing a pointer to the data, the current length, and the total capacity.\n- `forge_slice_empty(SliceName)`: Initializes a slice structure to an empty state (null data, zero length and capacity).\n- `forge_push(slice_ptr, elem, SliceName)`: Appends an element to the slice. If the length reaches the capacity, it performs a geometric resize (doubling the capacity) using `realloc`.\n- `forge_pop(slice_ptr)`: Removes and returns the last element of the slice. The caller is responsible for ensuring the slice is not empty.\n- `forge_slice_lit(SliceName, ElemType, ...)`: A convenience macro that creates a new slice from a C initializer list, allocating heap memory for the contents.\n- `forge_contains(slice, elem)`: Performs a linear search through the slice to determine if a specific element is present.\n\n### Optional Types\nOptional types (`T?`) are represented as a struct containing a boolean flag (`has`) and the actual value.\n- `FORGE_OPT_DEF(ElemType, OptName)`: Defines a specialized optional struct for a given type.\n- `forge_some(value, OptName)`: Wraps a concrete value into an optional struct with the `has` flag set to true.\n- `forge_none(OptName)`: Returns an optional struct with the `has` flag set to false.\n- `forge_isnull(opt)`: A helper to check if an optional value is empty.\n- `forge_unwrap(opt)`: Extracts the value from an optional. This should only be called after verifying the value exists.\n\n### Error Results\nForge's idiomatic `(T, error)` return pattern is mapped to a result struct in C.\n- `FORGE_RESULT_DEF(ElemType, ResultName)`: Defines a result struct containing a success value, an error message (`const char*`), and a boolean `is_err` flag.\n- `forge_ok(val, ResultName)`: Constructs a successful result containing the provided value.\n- `forge_err(msg, ResultName)`: Constructs an error result containing the provided error message.\n- `forge_is_err(r)`: Checks if a result represents a failure.\n\n### String Helpers\nThe runtime provides a collection of inline functions for high-level string manipulation, mimicking Forge's built-in string methods.\n- `forge_str_contains` and `forge_str_index_of`: Provide basic searching within strings.\n- `forge_str_has_prefix` and `forge_str_has_suffix`: Perform prefix and suffix checks.\n- `forge_str_replace`, `forge_str_repeat`, and `forge_str_join`: Handle string transformations and concatenation, often returning new heap-allocated strings.\n- `forge_sprintf`: A wrapper around `vsnprintf` that automatically calculates the required size and returns a heap-allocated formatted string.\n\n### Tagged Unions\nFor ad-hoc union types (e.g., `string | i32`), the runtime provides a generic `ForgeUnion` struct.\n- `ForgeUnion`: A struct containing an integer tag and a union of supported Forge types (i32, i64, f32, f64, bool, string, and pointers).\n- `forge_union_i32`, `forge_union_string`, etc.: Specialized constructor functions that initialize a `ForgeUnion` with the correct tag and value.\n\n## Implementation Details\n\n### Memory Management\nThe runtime currently utilizes the standard C library's `malloc`, `realloc`, and `free` for memory management. A critical implementation detail is that the runtime does not yet include a garbage collector. Consequently, heap-allocated objects—including slices, strings created by helper functions, and class instances—are currently leaked during execution. This is a known limitation that is acceptable for short-lived compiler tasks but will require a GC (such as Boehm-Demers-Weiser) for broader use.\n\n### Slice Growth Strategy\nThe `forge_push` macro implements a geometric growth strategy to ensure amortized O(1) append operations. When a slice's length reaches its capacity, the capacity is doubled. If the slice was previously empty, it is initialized with a default capacity of 4. This minimizes the number of reallocations required as a slice grows.\n\n### String Handling\nStrings in the Forge C runtime are represented as standard null-terminated `const char*`. The helper functions are designed to be safe and convenient, often allocating new memory for results to maintain the appearance of string immutability, which aligns with Forge's language semantics.\n\n### Tagged Union Dispatch\nThe `ForgeUnion` type uses an integer tag to identify the active member of its internal union. This allows the generated C code to perform safe type dispatching (e.g., in `match` statements) by checking the tag before accessing the union's data.\n\n## Dependencies\n\n- **Standard C Library**: The runtime depends on standard headers including `\u003cstdio.h\u003e`, `\u003cstdlib.h\u003e`, `\u003cstdint.h\u003e`, `\u003cstdbool.h\u003e`, `\u003cstring.h\u003e`, and `\u003cstdarg.h\u003e`.\n- **LIR Module**: This module is a critical dependency for the C backend in [pkg/lir](../pkg/lir/design.md), which relies on the macros and types defined here to emit valid C code.\n\n## Technical Debt and Future Work\n\n- **Garbage Collection**: Integrating a garbage collector is the highest priority to prevent memory leaks in long-running Forge programs compiled to C.\n- **Concurrency Support**: The C runtime currently lacks implementation for Forge's concurrency primitives, such as `spawn`, channels, and `select`.\n- **Map Implementation**: A robust hash map implementation is required to support Forge's `map\u003cK, V\u003e` type in the C backend.\n- **Error Handling Refinement**: The current error result uses a simple string for error messages. This should be expanded to support more complex, structured error types as the language evolves.\n- **Thread Safety**: The current runtime primitives, particularly slice operations, are not thread-safe and will require synchronization mechanisms once concurrency is introduced.\n",
  "file_path": "runtime/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
✅ **File Created Successfully**

**File**: `/Users/bill/projects/forge/runtime/design.md`
**Size**: 8108 bytes
**Permissions**: -rw-r--r--
**Lines**: 87
**Duration**: 0.00s
```

---

### ASSISTANT (gemini)

**Tool Call: `read_file`**
```json
{
  "file_path": "runtime/design.md"
}
```

---

### USER (gemini)

**Tool Result:**
```
[ID: tool_8e6f2321b966629b]
File: /Users/bill/projects/forge/runtime/design.md (87 lines)

# Runtime Module Design

## Executive Summary

The `runtime` module provides the foundational C runtime library required by the Forge compiler's C backend. It is implemented as a passive, header-only library that establishes the memory layout and operational semantics for Forge's high-level constructs in a C environment. By providing a suite of macros and inline functions, the runtime enables the C backend to transform complex Forge features—such as dynamic slices, optional types, error results, and tagged unions—into efficient and readable C code. Its primary goal is to bridge the gap between Forge's expressive type system and C's low-level primitives while maintaining semantic equivalence.

## File Inventory

- [forge_runtime.h](forge_runtime.h): The core runtime header. It contains all macro definitions, type structures, and inline helper functions that constitute the Forge execution environment for C targets.

## Architecture and Data Flow

The `runtime` module is a passive component that does not maintain internal state or have an independent execution flow. Instead, it acts as a contract and a toolkit for the code generated by the [LIR module](../pkg/lir/design.md). 

The architecture leverages C macros to implement a form of "template-based" type generation. Since C lacks native support for generics, the Forge C backend uses these macros to define concrete, specialized struct types for every unique instantiation of a generic Forge type (e.g., `Slice_int32_t` or `Opt_string`). This approach ensures type safety at the C level and allows the compiler to generate code that is tailored to specific data types. Data flows through these structures as they are passed between functions, with the runtime providing the logic for common operations like resizing slices or wrapping values in optional containers.

## Interface Implementations

As a C header library, this module does not implement any Go interfaces. It serves as the target execution environment for the C code generator within the [LIR module](../pkg/lir/design.md).

## Public API

The Public API of the runtime consists of the macros and inline functions intended to be utilized by the Forge-generated C code. These are categorized by the Forge language features they support.

### Dynamic Slices
Forge slices are implemented as dynamic arrays with automatic resizing capabilities.
- `FORGE_SLICE_DEF(ElemType, SliceName)`: A macro that defines a struct containing a pointer to the data, the current length, and the total capacity.
- `forge_slice_empty(SliceName)`: Initializes a slice structure to an empty state (null data, zero length and capacity).
- `forge_push(slice_ptr, elem, SliceName)`: Appends an element to the slice. If the length reaches the capacity, it performs a geometric resize (doubling the capacity) using `realloc`.
- `forge_pop(slice_ptr)`: Removes and returns the last element of the slice. The caller is responsible for ensuring the slice is not empty.
- `forge_slice_lit(SliceName, ElemType, ...)`: A convenience macro that creates a new slice from a C initializer list, allocating heap memory for the contents.
- `forge_contains(slice, elem)`: Performs a linear search through the slice to determine if a specific element is present.

### Optional Types
Optional types (`T?`) are represented as a struct containing a boolean flag (`has`) and the actual value.
- `FORGE_OPT_DEF(ElemType, OptName)`: Defines a specialized optional struct for a given type.
- `forge_some(value, OptName)`: Wraps a concrete value into an optional struct with the `has` flag set to true.
- `forge_none(OptName)`: Returns an optional struct with the `has` flag set to false.
- `forge_isnull(opt)`: A helper to check if an optional value is empty.
- `forge_unwrap(opt)`: Extracts the value from an optional. This should only be called after verifying the value exists.

### Error Results
Forge's idiomatic `(T, error)` return pattern is mapped to a result struct in C.
- `FORGE_RESULT_DEF(ElemType, ResultName)`: Defines a result struct containing a success value, an error message (`const char*`), and a boolean `is_err` flag.
- `forge_ok(val, ResultName)`: Constructs a successful result containing the provided value.
- `forge_err(msg, ResultName)`: Constructs an error result containing the provided error message.
- `forge_is_err(r)`: Checks if a result represents a failure.

### String Helpers
The runtime provides a collection of inline functions for high-level string manipulation, mimicking Forge's built-in string methods.
- `forge_str_contains` and `forge_str_index_of`: Provide basic searching within strings.
- `forge_str_has_prefix` and `forge_str_has_suffix`: Perform prefix and suffix checks.
- `forge_str_replace`, `forge_str_repeat`, and `forge_str_join`: Handle string transformations and concatenation, often returning new heap-allocated strings.
- `forge_sprintf`: A wrapper around `vsnprintf` that automatically calculates the required size and returns a heap-allocated formatted string.

### Tagged Unions
For ad-hoc union types (e.g., `string | i32`), the runtime provides a generic `ForgeUnion` struct.
- `ForgeUnion`: A struct containing an integer tag and a union of supported Forge types (i32, i64, f32, f64, bool, string, and pointers).
- `forge_union_i32`, `forge_union_string`, etc.: Specialized constructor functions that initialize a `ForgeUnion` with the correct tag and value.

## Implementation Details

### Memory Management
The runtime currently utilizes the standard C library's `malloc`, `realloc`, and `free` for memory management. A critical implementation detail is that the runtime does not yet include a garbage collector. Consequently, heap-allocated objects—including slices, strings created by helper functions, and class instances—are currently leaked during execution. This is a known limitation that is acceptable for short-lived compiler tasks but will require a GC (such as Boehm-Demers-Weiser) for broader use.

### Slice Growth Strategy
The `forge_push` macro implements a geometric growth strategy to ensure amortized O(1) append operations. When a slice's length reaches its capacity, the capacity is doubled. If the slice was previously empty, it is initialized with a default capacity of 4. This minimizes the number of reallocations required as a slice grows.

### String Handling
Strings in the Forge C runtime are represented as standard null-terminated `const char*`. The helper functions are designed to be safe and convenient, often allocating new memory for results to maintain the appearance of string immutability, which aligns with Forge's language semantics.

### Tagged Union Dispatch
The `ForgeUnion` type uses an integer tag to identify the active member of its internal union. This allows the generated C code to perform safe type dispatching (e.g., in `match` statements) by checking the tag before accessing the union's data.

## Dependencies

- **Standard C Library**: The runtime depends on standard headers including `<stdio.h>`, `<stdlib.h>`, `<stdint.h>`, `<stdbool.h>`, `<string.h>`, and `<stdarg.h>`.
- **LIR Module**: This module is a critical dependency for the C backend in [pkg/lir](../pkg/lir/design.md), which relies on the macros and types defined here to emit valid C code.

## Technical Debt and Future Work

- **Garbage Collection**: Integrating a garbage collector is the highest priority to prevent memory leaks in long-running Forge programs compiled to C.
- **Concurrency Support**: The C runtime currently lacks implementation for Forge's concurrency primitives, such as `spawn`, channels, and `select`.
- **Map Implementation**: A robust hash map implementation is required to support Forge's `map<K, V>` type in the C backend.
- **Error Handling Refinement**: The current error result uses a simple string for error messages. This should be expanded to support more complex, structured error types as the language evolves.
- **Thread Safety**: The current runtime primitives, particularly slice operations, are not thread-safe and will require synchronization mechanisms once concurrency is introduced.
```

---

### ASSISTANT (gemini)

Documentation complete.

---

