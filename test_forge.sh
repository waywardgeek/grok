#!/bin/bash
# test_forge.sh — Test the Forge compiler against all testdata/*.fg files
# Usage: ./test_forge.sh [--verbose] [pattern]
#
# Builds forge from forge.c, then compiles+runs each test file.
# Test files with test_ functions use `forge test`, others use `forge compile`.

set -euo pipefail

cd "$(dirname "$0")"

RUNTIME_DIR="runtime"
TMPDIR=$(mktemp -d -t forge_test_XXXXXX)
trap "rm -rf $TMPDIR" EXIT

VERBOSE=false
PATTERN=""

for arg in "$@"; do
  case "$arg" in
    --verbose) VERBOSE=true ;;
    *) PATTERN="$arg" ;;
  esac
done

# Build forge from checked-in C
echo "Building forge..."
make -s forge
FORGE="./forge"
echo ""

# Skip files that use features not yet implemented (channels, spawn, select, lock)
SKIP_FILES="channels.fg spawn.fg select.fg lock.fg guarded_by.fg"

PASS=0
FAIL=0
SKIP=0
FAILURES=""

for fg in testdata/*.fg; do
  name=$(basename "$fg")

  # Filter by pattern if given
  if [ -n "$PATTERN" ] && [[ "$name" != *"$PATTERN"* ]]; then
    continue
  fi

  # Check skip list
  skip=false
  for s in $SKIP_FILES; do
    if [ "$name" = "$s" ]; then
      skip=true
      break
    fi
  done
  if $skip; then
    SKIP=$((SKIP + 1))
    if $VERBOSE; then echo "SKIP  $name"; fi
    continue
  fi

  # Detect test-only files (have test_ functions but no main)
  CMD="compile"
  if grep -q 'func test_' "$fg" && ! grep -q 'func main()' "$fg" && ! grep -q 'func Main()' "$fg"; then
    CMD="test"
  fi

  # Determine dependencies for unit tests
  DEPS=""
  case "$name" in
    test_lexer.fg) DEPS="bootstrap/lexer/lexer.fg bootstrap/ast/ast.fg bootstrap/parser/parser.fg bootstrap/parser/expr_parser.fg" ;;
    test_parser.fg) DEPS="bootstrap/parser/parser.fg bootstrap/parser/expr_parser.fg bootstrap/lexer/lexer.fg bootstrap/ast/ast.fg" ;;
    test_desugar.fg) DEPS="bootstrap/desugar/desugar.fg bootstrap/parser/parser.fg bootstrap/parser/expr_parser.fg bootstrap/lexer/lexer.fg bootstrap/ast/ast.fg" ;;
    test_min.fg) DEPS="bootstrap/parser/parser.fg bootstrap/parser/expr_parser.fg bootstrap/lexer/lexer.fg bootstrap/ast/ast.fg" ;;
  esac

  out_c="$TMPDIR/${name%.fg}.c"
  out_bin="$TMPDIR/${name%.fg}"

  # Step 1: Compile .fg → .c
  if ! $FORGE $CMD "$fg" $DEPS -o "$out_c" 2>"$TMPDIR/err"; then
    FAIL=$((FAIL + 1))
    err=$(cat "$TMPDIR/err")
    FAILURES="$FAILURES\nFAIL  $name  (compile: $err)"
    echo "FAIL  $name  (compile)"
    if $VERBOSE; then cat "$TMPDIR/err"; fi
    continue
  fi

  # For `forge test`, the test already ran — check exit code (already checked above)
  if [ "$CMD" = "test" ]; then
    PASS=$((PASS + 1))
    echo "PASS  $name"
    continue
  fi

  # Step 2: GCC compile
  if ! gcc -std=gnu11 -O2 -o "$out_bin" "$out_c" -I "$RUNTIME_DIR" 2>"$TMPDIR/err"; then
    FAIL=$((FAIL + 1))
    err=$(head -5 "$TMPDIR/err")
    FAILURES="$FAILURES\nFAIL  $name  (gcc: $err)"
    echo "FAIL  $name  (gcc)"
    if $VERBOSE; then head -20 "$TMPDIR/err"; fi
    continue
  fi

  # Step 3: Run
  if ! "$out_bin" >"$TMPDIR/stdout" 2>"$TMPDIR/stderr"; then
    FAIL=$((FAIL + 1))
    err=$(tail -5 "$TMPDIR/stderr")
    FAILURES="$FAILURES\nFAIL  $name  (runtime: $err)"
    echo "FAIL  $name  (runtime)"
    if $VERBOSE; then tail -10 "$TMPDIR/stderr"; fi
    continue
  fi

  PASS=$((PASS + 1))
  echo "PASS  $name"
done

echo ""
echo "=== Results ==="
echo "PASS: $PASS  FAIL: $FAIL  SKIP: $SKIP  TOTAL: $((PASS + FAIL + SKIP))"
if [ -n "$FAILURES" ]; then
  echo ""
  echo "=== Failures ==="
  echo -e "$FAILURES"
fi

exit $( [ $FAIL -eq 0 ] && echo 0 || echo 1 )
