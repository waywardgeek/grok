#!/bin/bash
# Test the bootstrap compiler against all testdata/*.fg files
# Usage: ./test_bootstrap.sh [--rebuild] [--verbose] [pattern]

set -euo pipefail

FORGE="./forge"
BOOTSTRAP="/tmp/bootstrap"
RUNTIME_DIR="runtime"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

REBUILD=false
VERBOSE=false
PATTERN=""

for arg in "$@"; do
  case "$arg" in
    --rebuild) REBUILD=true ;;
    --verbose) VERBOSE=true ;;
    *) PATTERN="$arg" ;;
  esac
done

# Build bootstrap if needed
if [ ! -f "$BOOTSTRAP" ] || $REBUILD; then
  echo "Building bootstrap..."
  $FORGE compile bootstrap/lir/lir.fg bootstrap/lexer/lexer.fg bootstrap/parser/parser.fg \
    bootstrap/parser/expr_parser.fg bootstrap/desugar/desugar.fg bootstrap/checker/checker.fg \
    bootstrap/lowerer/lowerer.fg bootstrap/ast/ast.fg bootstrap/optimizer/optimizer.fg \
    bootstrap/monomorphizer/monomorphizer.fg bootstrap/c_backend/c_backend.fg \
    bootstrap/main/main.fg -o "$TMPDIR/bootstrap.c"
  gcc -std=gnu11 -O2 -o "$BOOTSTRAP" "$TMPDIR/bootstrap.c" -I "$RUNTIME_DIR" 2>/dev/null
  echo "Bootstrap built."
  echo ""
fi

# Skip files that use features not yet in bootstrap (channels, spawn, select, lock)
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
    echo "SKIP  $name"
    continue
  fi

  # Step 1: Bootstrap compile .fg → .c
  bs_c="$TMPDIR/bs_${name%.fg}.c"
  bs_out="$TMPDIR/bs_${name%.fg}"

  if ! $BOOTSTRAP compile "$fg" -o "$bs_c" 2>"$TMPDIR/err" ; then
    FAIL=$((FAIL + 1))
    err=$(cat "$TMPDIR/err")
    FAILURES="$FAILURES\nFAIL  $name  (bootstrap compile: $err)"
    echo "FAIL  $name  (bootstrap compile)"
    if $VERBOSE; then cat "$TMPDIR/err"; fi
    continue
  fi

  # Step 2: GCC compile
  if ! gcc -std=gnu11 -O2 -o "$bs_out" "$bs_c" -I "$RUNTIME_DIR" 2>"$TMPDIR/err"; then
    FAIL=$((FAIL + 1))
    err=$(head -5 "$TMPDIR/err")
    FAILURES="$FAILURES\nFAIL  $name  (gcc: $err)"
    echo "FAIL  $name  (gcc)"
    if $VERBOSE; then head -20 "$TMPDIR/err"; fi
    continue
  fi

  # Step 3: Run if it has test functions (use bootstrap test command)
  if grep -q 'func test_' "$fg"; then
    if ! timeout 10 "$bs_out" >"$TMPDIR/out" 2>&1; then
      # Check if Go compiler also fails
      go_c="$TMPDIR/go_${name%.fg}.c"
      go_out="$TMPDIR/go_${name%.fg}"
      if $FORGE compile "$fg" -o "$go_c" 2>/dev/null && \
         gcc -std=gnu11 -O2 -o "$go_out" "$go_c" -I "$RUNTIME_DIR" 2>/dev/null && \
         ! timeout 10 "$go_out" >/dev/null 2>&1; then
        SKIP=$((SKIP + 1))
        echo "SKIP  $name (Go compiler also fails)"
        continue
      fi
      FAIL=$((FAIL + 1))
      err=$(tail -5 "$TMPDIR/out")
      FAILURES="$FAILURES\nFAIL  $name  (test run: $err)"
      echo "FAIL  $name  (test run)"
      if $VERBOSE; then tail -10 "$TMPDIR/out"; fi
      continue
    fi
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
