// Command grok-compile parses .gk files and transpiles them to Go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/waywardgeek/grok/pkg/ast"
	"github.com/waywardgeek/grok/pkg/checker"
	"github.com/waywardgeek/grok/pkg/parser"
	"github.com/waywardgeek/grok/pkg/transpiler"
)

func main() {
	if err := runCompile(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCompile(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: grok-compile <file.gk>... [-o output.go] [-pkg name]")
	}

	var inputs []string
	output := ""
	pkg := "main"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o":
			i++
			if i < len(args) {
				output = args[i]
			}
		case "-pkg":
			i++
			if i < len(args) {
				pkg = args[i]
			}
		default:
			inputs = append(inputs, args[i])
		}
	}

	if len(inputs) == 0 {
		return fmt.Errorf("no input files")
	}

	// Parse all files
	type parsedFile struct {
		file   *ast.File
		input  string
		output string
	}
	var files []parsedFile
	for _, input := range inputs {
		src, err := os.ReadFile(input)
		if err != nil {
			return fmt.Errorf("reading %s: %w", input, err)
		}
		file, err := parser.ParseFile(string(src), input)
		if err != nil {
			return err
		}
		out := output
		if out == "" {
			out = strings.TrimSuffix(filepath.Base(input), filepath.Ext(input)) + ".go"
		}
		files = append(files, parsedFile{file: file, input: input, output: out})
	}

	// Shared type checker — register types from all files first
	ch := checker.New()
	for _, pf := range files {
		ch.CheckFile(pf.file)
	}
	if errs := ch.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		return fmt.Errorf("type check failed with %d error(s)", len(errs))
	}

	// Transpile each file
	for _, pf := range files {
		tr := transpiler.New(pkg)
		goSrc := tr.Transpile(pf.file)
		if err := os.WriteFile(pf.output, []byte(goSrc), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", pf.output, err)
		}
		fmt.Printf("wrote %s\n", pf.output)
	}
	return nil
}
