#!/bin/bash
# test_self_compile.sh — Three-stage bootstrap self-compilation test
# Stage 1: Go compiler builds bootstrap → bootstrap1 binary
# Stage 2: bootstrap1 compiles itself → bootstrap2 binary  
# Stage 3: bootstrap2 compiles itself → bootstrap3.c (compare with stage 2 output)
set -e

cd "$(dirname "$0")"

BOOTSTRAP_FILES=(
  bootstrap/ast/ast.fg
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

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "=== Stage 0: Build Go compiler ==="
go build -o "$TMPDIR/forge_go" ./cmd/forge/
echo "OK"

echo ""
echo "=== Stage 1: Go compiler compiles bootstrap → bootstrap1.c ==="
time "$TMPDIR/forge_go" compile "${BOOTSTRAP_FILES[@]}" -o "$TMPDIR/bootstrap1.c" 2>&1
echo ""
echo "Compiling bootstrap1.c with GCC..."
time gcc -std=gnu11 -O2 -w -o "$TMPDIR/bootstrap1" "$TMPDIR/bootstrap1.c" -I runtime/
echo "bootstrap1: $(wc -l < "$TMPDIR/bootstrap1.c") lines C"
echo ""

echo "=== Stage 2: bootstrap1 compiles itself → bootstrap2.c ==="
time "$TMPDIR/bootstrap1" compile "${BOOTSTRAP_FILES[@]}" -o "$TMPDIR/bootstrap2.c" 2>&1
echo ""
echo "Compiling bootstrap2.c with GCC..."
time gcc -std=gnu11 -O2 -w -o "$TMPDIR/bootstrap2" "$TMPDIR/bootstrap2.c" -I runtime/
echo "bootstrap2: $(wc -l < "$TMPDIR/bootstrap2.c") lines C"
echo ""

echo "=== Stage 3: bootstrap2 compiles itself → bootstrap3.c ==="
time "$TMPDIR/bootstrap2" compile "${BOOTSTRAP_FILES[@]}" -o "$TMPDIR/bootstrap3.c" 2>&1
echo ""
echo "bootstrap3: $(wc -l < "$TMPDIR/bootstrap3.c") lines C"
echo ""

echo "=== Comparing Stage 2 and Stage 3 outputs ==="
if diff -q "$TMPDIR/bootstrap2.c" "$TMPDIR/bootstrap3.c" > /dev/null 2>&1; then
  echo "✅ FIXED POINT REACHED — bootstrap2.c == bootstrap3.c"
else
  echo "❌ bootstrap2.c != bootstrap3.c"
  diff "$TMPDIR/bootstrap2.c" "$TMPDIR/bootstrap3.c" | head -40
  exit 1
fi
