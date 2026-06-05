package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCmdCompileWritesGoOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "hello.gk")
	output := filepath.Join(dir, "hello.go")

	writeTestFile(t, input, `grok hello {
  func main() {
    println("hello")
  }
}
`)

	if err := cmdCompile([]string{input, "-o", output}); err != nil {
		t.Fatalf("cmdCompile failed: %v", err)
	}

	got := readTestFile(t, output)
	assertContains(t, got, "package main")
	assertContains(t, got, `"fmt"`)
	assertContains(t, got, "func main()")
	assertContains(t, got, `fmt.Println("hello")`)
}

func TestCmdVerifyValidGrokFile(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "example.go")
	grokFile := filepath.Join(dir, "example.grok")

	writeTestFile(t, goFile, `package example

type Widget struct {
	Name string
}

func NewWidget(name string) *Widget {
	return &Widget{Name: name}
}
`)
	writeTestFile(t, grokFile, `grok Example {
  struct Widget {
    name: string
  }

  func new_widget(name: string) -> Widget?

  source: ["example.go"]
}
`)

	if err := cmdVerify([]string{grokFile}); err != nil {
		t.Fatalf("cmdVerify failed: %v", err)
	}
}

func TestCmdGenScaffoldsGrokFile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "widget.go"), `package sample

type Widget struct {
	Name string
}

func NewWidget(name string) *Widget {
	return &Widget{Name: name}
}
`)

	if err := cmdGen([]string{dir}); err != nil {
		t.Fatalf("cmdGen failed: %v", err)
	}

	got := readTestFile(t, filepath.Join(dir, "sample.grok"))
	assertContains(t, got, "grok Sample")
	assertContains(t, got, "struct Widget")
	assertContains(t, got, "Name: string")
	assertContains(t, got, `source: ["widget.go"]`)
	assertContains(t, got, "func NewWidget(name string) *Widget")
}

func TestCmdFmtFormatsZoneOneAndPreservesGeneratedZones(t *testing.T) {
	dir := t.TempDir()
	grokFile := filepath.Join(dir, "example.grok")

	writeTestFile(t, grokFile, `grok Example {
struct Point { x: i32 }
source: ["point.go"]
}
// --- index ---
// keep this index text
`)

	if err := cmdFmt([]string{grokFile}); err != nil {
		t.Fatalf("cmdFmt failed: %v", err)
	}

	got := readTestFile(t, grokFile)
	assertContains(t, got, "grok Example {")
	assertContains(t, got, "  struct Point {")
	assertContains(t, got, "    x: i32")
	assertContains(t, got, `  source: ["point.go"]`)
	assertContains(t, got, "// --- index ---\n// keep this index text\n")
}

func TestCmdUpdateAddsMissingSymbolsAndRegeneratesIndex(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "example.go")
	grokFile := filepath.Join(dir, "example.grok")

	writeTestFile(t, goFile, `package example

type Config struct {
	Debug bool
}

func ParseConfig() *Config {
	return nil
}
`)
	writeTestFile(t, grokFile, `grok Example {
  why: "Integration test fixture."

  source: ["example.go"]
}
`)

	if err := cmdUpdate([]string{grokFile}); err != nil {
		t.Fatalf("cmdUpdate failed: %v", err)
	}

	got := readTestFile(t, grokFile)
	assertContains(t, got, "struct Config")
	assertContains(t, got, "Debug: bool")
	assertContains(t, got, "func ParseConfig() -> Config?")
	assertContains(t, got, "// --- index ---")
	assertContains(t, got, "func ParseConfig() *Config")
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q, got:\n%s", want, got)
	}
}
