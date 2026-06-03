# Grok

Parser, verifier, and compiler for the Grok language.

- **Parser**: PEG parser for `.grok` (declaration-only) and `.gk` (full code) files
- **Verifier**: Compares `.grok` declarations against Go source code, reports drift
- **Compiler**: Transpiles `.gk` files to Go

## Status

Early development. Building with GDD — using Grok to build Grok.

## Language Spec

See [grok-language.md](https://coderhapsody.ai/docs/grok-language) for the full specification.

## Build

```
go build -o grok ./cmd/
go test ./...
```
