package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdCompileStopsOnCheckerErrors(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "bad.gk")
	output := filepath.Join(dir, "bad.go")

	src := `grok bad {
  func main() {
    let value: string = 42
  }
}
`
	if err := os.WriteFile(input, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}

	err := cmdCompile([]string{input, "-o", output})
	if err == nil {
		t.Fatal("expected type-check failure")
	}
	if !strings.Contains(err.Error(), "type check failed") {
		t.Fatalf("expected type-check failure, got %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("expected no output file, stat error: %v", statErr)
	}
}
