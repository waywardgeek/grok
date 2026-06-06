package ast

import (
	"os"
	"path/filepath"
)

// MergeStdlib merges interface declarations from a parsed stdlib file
// into every block of the target file. Only interfaces that are referenced
// by a relation declaration are merged, to avoid polluting programs that
// don't use them.
func MergeStdlib(file *File, stdFile *File) {
	// Collect all relation hints (interface names used in relations)
	usedIfaces := make(map[string]bool)
	for _, block := range file.Blocks {
		for _, rel := range block.Relations {
			usedIfaces[rel.Hint] = true
		}
	}

	if len(usedIfaces) == 0 {
		return
	}

	// Build a lookup of all stdlib interfaces by name
	stdIfaceMap := make(map[string]InterfaceDecl)
	for _, block := range stdFile.Blocks {
		for _, iface := range block.Interfaces {
			stdIfaceMap[iface.Name] = iface
		}
	}

	// Transitively pull in interfaces referenced by embeds
	queue := make([]string, 0, len(usedIfaces))
	for name := range usedIfaces {
		queue = append(queue, name)
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		iface, ok := stdIfaceMap[name]
		if !ok {
			continue
		}
		for _, emb := range iface.Embeds {
			embName := emb.Name
			if !usedIfaces[embName] {
				usedIfaces[embName] = true
				queue = append(queue, embName)
			}
		}
	}

	// Collect only referenced interface declarations from the stdlib
	var stdIfaces []InterfaceDecl
	for name := range usedIfaces {
		if iface, ok := stdIfaceMap[name]; ok {
			stdIfaces = append(stdIfaces, iface)
		}
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

	// Relative to working directory — walk up to find project root
	dir, _ := os.Getwd()
	for dir != "/" && dir != "." {
		candidate := filepath.Join(dir, "stdlib")
		if info, err := os.Stat(filepath.Join(candidate, "std.fg")); err == nil && !info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}

	return ""
}
