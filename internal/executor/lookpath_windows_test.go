package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLookPathInHonoursPATHEXT(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "docker.exe")
	if err := os.WriteFile(p, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := lookPathIn("docker", dir, "windows", ".COM;.EXE")
	if err != nil || !strings.EqualFold(got, p) {
		t.Fatalf("lookPathIn = %q, %v; want %q", got, err, p)
	}
}
