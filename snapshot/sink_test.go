package snapshot

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_NewSinkOpenOK(t *testing.T) {
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "work")
	mustCreateDir(workDir)
	currGenDir := filepath.Join(tmpDir, "curr")
	nextGenDir := filepath.Join(tmpDir, "next")
	s := NewSink(workDir, currGenDir, nextGenDir, &Meta{})
	if err := s.Open(); err != nil {
		t.Fatal(err)
	}
}

func mustCreateDir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		panic(err)
	}
}
