#!/bin/bash
# build_bootstrap.sh — Build the Forge compiler from source
# Uses the checked-in forge.c binary to compile src source.
# Usage: ./build_bootstrap.sh [output_binary]
# Default output: ./forge
set -e

cd "$(dirname "$0")"

OUTPUT="${1:-./forge}"

BOOTSTRAP_FILES=(
  src/ast/ast.fg src/ast/modules.fg
  src/lexer/lexer.fg
  src/parser/parser.fg
  src/parser/expr_parser.fg
  src/desugar/desugar.fg
  src/checker/checker.fg
  src/lir/lir.fg
  src/lowerer/lowerer.fg
  src/optimizer/optimizer.fg
  src/monomorphizer/monomorphizer.fg
  src/c_backend/c_backend.fg
  src/main/main.fg
)

TMPDIR_BUILD=$(mktemp -d -t forge_build_XXXXXX)
trap "rm -rf $TMPDIR_BUILD" EXIT

# Build from checked-in C if no forge binary exists
if [ ! -f ./forge ]; then
  echo "=== Building forge from forge.c ==="
  gcc -std=gnu11 -O2 -w -I runtime -o ./forge forge.c -lm
fi

echo "=== Compiling src → C ==="
./forge compile "${BOOTSTRAP_FILES[@]}" -o "$TMPDIR_BUILD/forge_new.c" 2>&1

echo "=== GCC compile ==="
gcc -std=gnu11 -O2 -w -I runtime -o "$OUTPUT" "$TMPDIR_BUILD/forge_new.c" -lm
echo "=== Built: $OUTPUT ==="
