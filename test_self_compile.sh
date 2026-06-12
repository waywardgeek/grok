#!/bin/bash
# test_self_compile.sh — Two-stage src self-compilation test
# Stage 1: Build forge from checked-in forge.c, compile src → forge_stage2.c
# Stage 2: Build from forge_stage2.c, compile src → forge_stage3.c
# Fixed point: forge_stage2.c == forge_stage3.c
set -e

cd "$(dirname "$0")"

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

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "=== Stage 0: Build forge from checked-in forge.c ==="
make -s forge
echo "OK"
echo ""

echo "=== Stage 1: forge compiles itself → forge_stage2.c ==="
time ./forge compile "${BOOTSTRAP_FILES[@]}" -o "$TMPDIR/forge_stage2.c" 2>&1
echo ""
echo "Compiling forge_stage2.c with GCC..."
time gcc -std=gnu11 -O2 -w -o "$TMPDIR/forge_stage2" "$TMPDIR/forge_stage2.c" -I runtime/
echo "forge_stage2: $(wc -l < "$TMPDIR/forge_stage2.c") lines C"
echo ""

echo "=== Stage 2: forge_stage2 compiles itself → forge_stage3.c ==="
time "$TMPDIR/forge_stage2" compile "${BOOTSTRAP_FILES[@]}" -o "$TMPDIR/forge_stage3.c" 2>&1
echo ""
echo "forge_stage3: $(wc -l < "$TMPDIR/forge_stage3.c") lines C"
echo ""

echo "=== Comparing Stage 1 and Stage 2 outputs ==="
if diff -q "$TMPDIR/forge_stage2.c" "$TMPDIR/forge_stage3.c" > /dev/null 2>&1; then
  echo "✅ FIXED POINT REACHED — forge_stage2.c == forge_stage3.c"
else
  echo "❌ forge_stage2.c != forge_stage3.c"
  diff "$TMPDIR/forge_stage2.c" "$TMPDIR/forge_stage3.c" | head -40
  exit 1
fi

# Also verify forge.c matches (it should be identical to what forge produces)
echo ""
echo "=== Verifying checked-in forge.c is current ==="
if diff -q forge.c "$TMPDIR/forge_stage2.c" > /dev/null 2>&1; then
  echo "✅ forge.c matches compiler output"
else
  echo "⚠️  forge.c differs from compiler output — run 'make update' to refresh"
  diff forge.c "$TMPDIR/forge_stage2.c" | head -20
fi
