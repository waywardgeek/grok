package lir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDumpC(t *testing.T) {
	files := []string{"interfaces", "lambdas", "demo", "advanced", "methods", "stdlib", "collections", "unions", "typealias", "channels", "http_router", "inference"}
	for _, name := range files {
		path := filepath.Join("..", "..", "testdata", name+".gk")
		data, err := os.ReadFile(path)
		if err != nil { continue }
		c := cPipeline(t, string(data), name)
		os.WriteFile("/tmp/c_"+name+".c", []byte(c), 0644)
		t.Logf("wrote /tmp/c_%s.c", name)
	}
}
