package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRegeneratesFunctionIndex(t *testing.T) {
	dir := t.TempDir()
	grokFile := filepath.Join(dir, "example.grok")

	writeUpdateTestFile(t, filepath.Join(dir, "example.go"), `package example

func ParseConfig() error {
	return nil
}
`)
	writeUpdateTestFile(t, grokFile, `grok Example {
  why: "Preserve this human-curated section."

  source: ["example.go"]
}
// --- index ---
// stale index text
`)

	if err := run(grokFile); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	got := readUpdateTestFile(t, grokFile)
	assertUpdateContains(t, got, `why: "Preserve this human-curated section."`)
	assertUpdateContains(t, got, `source: ["example.go"]`)
	assertUpdateContains(t, got, "// --- index ---")
	assertUpdateContains(t, got, "// file: example.go")
	assertUpdateContains(t, got, "func ParseConfig() error")
	if strings.Contains(got, "stale index text") {
		t.Fatalf("expected stale index text to be replaced, got:\n%s", got)
	}
}

func writeUpdateTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func readUpdateTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func assertUpdateContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q, got:\n%s", want, got)
	}
}
