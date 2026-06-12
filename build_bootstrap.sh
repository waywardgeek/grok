#!/bin/bash
# build_bootstrap.sh — Build the Forge compiler from bootstrap .fg source
# Uses the checked-in forge.c binary to compile bootstrap source.
# Usage: ./build_bootstrap.sh [output_binary]
# Default output: ./forge
set -e

cd "$(dirname "$0")"

OUTPUT="${1:-./forge}"

BOOTSTRAP_FILES=(
  bootstrap/ast/ast.fg bootstrap/ast/modules.fg
  bootstrap/lexer/lexer.fg
  bootstrap/parser/parser.fg
  bootstrap/parser/expr_parser.fg
  bootstrap/desugar/desugar.fg
  bootstrap/checker/checker.fg
  bootstrap/lir/lir.fg
  bootstrap/lowerer/lowerer.fg
  bootstrap/optimizer/optimizer.fg
  bootstrap/monomorphizer/monomorphizer.fg
  bootstrap/c_backend/c_backend.fg
  bootstrap/main/main.fg
)

TMPDIR_BUILD=$(mktemp -d -t forge_build_XXXXXX)
trap "rm -rf $TMPDIR_BUILD" EXIT

# Build from checked-in C if no forge binary exists
if [ ! -f ./forge ]; then
  echo "=== Building forge from forge.c ==="
  gcc -std=gnu11 -O2 -w -I runtime -o ./forge forge.c -lm
fi

echo "=== Compiling bootstrap → C ==="
./forge compile "${BOOTSTRAP_FILES[@]}" -o "$TMPDIR_BUILD/forge_new.c" 2>&1

echo "=== GCC compile ==="
gcc -std=gnu11 -O2 -w -I runtime -o "$OUTPUT" "$TMPDIR_BUILD/forge_new.c" -lm
echo "=== Built: $OUTPUT ==="
