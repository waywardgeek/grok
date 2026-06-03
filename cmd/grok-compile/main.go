// Command grok-compile parses .gk files and transpiles them to Go.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/waywardgeek/grok/pkg/checker"
	"github.com/waywardgeek/grok/pkg/parser"
	"github.com/waywardgeek/grok/pkg/transpiler"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: grok-compile <file.gk> [-o output.go] [-pkg name]\n")
		os.Exit(1)
	}

	input := os.Args[1]
	output := ""
	pkg := "main"

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-o":
			i++
			if i < len(os.Args) {
				output = os.Args[i]
			}
		case "-pkg":
			i++
			if i < len(os.Args) {
				pkg = os.Args[i]
			}
		}
	}

	if output == "" {
		output = strings.TrimSuffix(filepath.Base(input), filepath.Ext(input)) + ".go"
	}

	src, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", input, err)
		os.Exit(1)
	}

	// Parse
	file, err := parser.ParseFile(string(src), input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Type check
	ch := checker.New()
	ch.CheckFile(file)
	if errs := ch.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		// Continue — warnings are informational
	}

	// Transpile
	tr := transpiler.New(pkg)
	goSrc := tr.Transpile(file)

	if err := os.WriteFile(output, []byte(goSrc), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", output, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", output)
}
