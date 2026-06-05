package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunScaffoldsGrokFile(t *testing.T) {
	dir := t.TempDir()
	writeGenTestFile(t, filepath.Join(dir, "widget.go"), `package sample

type Widget struct {
	Name string
}

func NewWidget(name string) *Widget {
	return &Widget{Name: name}
}
`)

	if err := run(dir); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	got := readGenTestFile(t, filepath.Join(dir, "sample.grok"))
	assertGenContains(t, got, "grok Sample")
	assertGenContains(t, got, "struct Widget")
	assertGenContains(t, got, "Name: string")
	assertGenContains(t, got, `source: ["widget.go"]`)
	assertGenContains(t, got, "func NewWidget(name string) *Widget")
}

func writeGenTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func readGenTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func assertGenContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q, got:\n%s", want, got)
	}
}
