package ast

import (
	"os"
	"path/filepath"
)

// MergeStdlib merges interface declarations from a parsed stdlib file
// into every block of the target file. This must be called before the
// desugar passes so that relation declarations can reference stdlib
// interfaces like ArrayList.
func MergeStdlib(file *File, stdFile *File) {
	// Collect all interface declarations from the stdlib
	var stdIfaces []InterfaceDecl
	for _, block := range stdFile.Blocks {
		stdIfaces = append(stdIfaces, block.Interfaces...)
	}

	if len(stdIfaces) == 0 {
		return
	}

	// Merge into every block of the target file
	for i := range file.Blocks {
		file.Blocks[i].Interfaces = append(stdIfaces, file.Blocks[i].Interfaces...)
	}
}

// FindStdlibDir locates the stdlib directory.
// Search order:
//  1. FORGE_STDLIB env var
//  2. ../stdlib/ relative to the executable
//  3. ./stdlib/ relative to current directory
func FindStdlibDir() string {
	if dir := os.Getenv("FORGE_STDLIB"); dir != "" {
		return dir
	}

	// Relative to executable
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "..", "stdlib")
		if info, err := os.Stat(filepath.Join(dir, "std.fg")); err == nil && !info.IsDir() {
			return dir
		}
		dir = filepath.Join(filepath.Dir(exe), "stdlib")
		if info, err := os.Stat(filepath.Join(dir, "std.fg")); err == nil && !info.IsDir() {
			return dir
		}
	}

	// Relative to working directory
	if info, err := os.Stat("stdlib/std.fg"); err == nil && !info.IsDir() {
		return "stdlib"
	}

	return ""
}
