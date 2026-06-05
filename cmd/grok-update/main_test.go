package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunIndexesNestedSourcePaths(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	goSrc := `package example

func Exported() {}
`
	if err := os.WriteFile(filepath.Join(srcDir, "example.go"), []byte(goSrc), 0644); err != nil {
		t.Fatal(err)
	}

	grokSrc := `grok Example {
  source: ["src/example.go"]
}
`
	grokPath := filepath.Join(dir, "example.grok")
	if err := os.WriteFile(grokPath, []byte(grokSrc), 0644); err != nil {
		t.Fatal(err)
	}

	if err := run(grokPath); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	updated, err := os.ReadFile(grokPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	if !strings.Contains(text, "// file: src/example.go") {
		t.Fatalf("expected nested source path in index, got:\n%s", text)
	}
	if !strings.Contains(text, "// func Exported()") {
		t.Fatalf("expected function signature in nested source index, got:\n%s", text)
	}
}
