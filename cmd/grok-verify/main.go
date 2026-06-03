// grok-verify is a CLI tool that checks .grok understanding files against Go source.
package main

import (
	"fmt"
	"os"

	"github.com/waywardgeek/grok/pkg/verifier"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: grok-verify <file.grok>\n")
		os.Exit(1)
	}

	grokPath := os.Args[1]

	result, err := verifier.Verify(grokPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	errors := 0
	warnings := 0
	for _, f := range result.Findings {
		fmt.Println(f)
		switch f.Severity {
		case verifier.Error:
			errors++
		case verifier.Warning:
			warnings++
		}
	}

	fmt.Printf("\n%d errors, %d warnings\n", errors, warnings)
	if errors > 0 {
		os.Exit(1)
	}
}
