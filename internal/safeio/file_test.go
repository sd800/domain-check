package safeio

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestUntrustedLocalFilesCannotBlockOrEscapeLimits(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "data.json")
	if err := os.WriteFile(regular, []byte(`{"language":"en"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if data, info, err := ReadFile(regular, 64); err != nil || string(data) != `{"language":"en"}` || info == nil {
		t.Fatalf("ordinary data rejected: %q, %v", data, err)
	}
	link, fifo := filepath.Join(dir, "link"), filepath.Join(dir, "fifo")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{link, fifo, dir, "/dev/zero"} {
		if _, _, err := ReadFile(name, 64); err == nil {
			t.Errorf("accepted special file: %s", name)
		}
	}
	if _, _, err := ReadFile(regular, 4); err == nil {
		t.Fatal("accepted oversized configuration")
	}
	if err := os.Chmod(regular, 0666); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadFile(regular, 64); err == nil {
		t.Fatal("accepted data writable by other users")
	}
}
