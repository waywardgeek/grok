# Forge Compiler — Makefile
#
# The Forge compiler is self-hosting. forge.c is the checked-in canonical
# compiler output (88K+ lines of C). Building requires only GCC and libc.
#
# Usage:
#   make            — build the forge binary
#   make test       — run test suite
#   make self-test  — verify fixed-point (forge compiles itself to identical C)
#   make update     — regenerate forge.c from bootstrap source
#   make clean      — remove build artifacts

CC      ?= gcc
CFLAGS  ?= -std=gnu11 -O2 -w
RUNTIME  = runtime

BOOTSTRAP_FILES = \
  bootstrap/ast/ast.fg bootstrap/ast/modules.fg \
  bootstrap/lexer/lexer.fg \
  bootstrap/parser/parser.fg bootstrap/parser/expr_parser.fg \
  bootstrap/desugar/desugar.fg \
  bootstrap/checker/checker.fg \
  bootstrap/lir/lir.fg \
  bootstrap/lowerer/lowerer.fg \
  bootstrap/optimizer/optimizer.fg \
  bootstrap/monomorphizer/monomorphizer.fg \
  bootstrap/c_backend/c_backend.fg \
  bootstrap/main/main.fg

.PHONY: all test self-test update clean

all: forge

forge: forge.c runtime/forge_runtime.h
	$(CC) $(CFLAGS) -I $(RUNTIME) -o $@ forge.c -lm

test: forge
	@bash test_forge.sh

self-test: forge
	@bash test_self_compile.sh

# Regenerate forge.c from bootstrap Forge source using the current forge binary
update: forge
	./forge compile $(BOOTSTRAP_FILES) -o forge.c
	@echo "forge.c updated ($$(wc -l < forge.c) lines)"

clean:
	rm -f forge
