#!/bin/bash
# slab_dev.sh — build forge from source using forge.sav, then compile+run a test file
set -e
cd /Users/bill/projects/forge
cp forge.sav forge
make update 2>&1 | tail -1
gcc -std=gnu11 -O2 -w -I runtime -o forge forge.c -lm
FILE="${1:-testdata/slab_test.fg}"
echo "--- Testing: $FILE ---"
./forge compile "$FILE" -o /tmp/slab_dev.c 2>/dev/null
gcc -std=gnu11 -O2 -w -I runtime -o /tmp/slab_dev /tmp/slab_dev.c -lm
/tmp/slab_dev
