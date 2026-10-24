package safeio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveFilesKeepsUnknownFilesAndDoesNotFollowLinks(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	target := filepath.Join(outside, "keep.json")
	for _, path := range []string{target, filepath.Join(dir, "config.json"), filepath.Join(dir, "keep.txt")} {
		if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(target, filepath.Join(dir, "web.json")); err != nil {
		t.Fatal(err)
	}
	if err := RemoveFiles(dir, "config.json", "web.json", "missing.json"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{target, filepath.Join(dir, "keep.txt")} {
		if data, err := os.ReadFile(path); err != nil || string(data) != "keep" {
			t.Fatal("changed unrelated data", path, err)
		}
	}
	for _, name := range []string{"config.json", "web.json"} {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatal("left known file", name, err)
		}
	}
	if err := RemoveFiles(filepath.Join(dir, "absent"), "config.json"); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveFilesRejectsDirectoriesTraversalAndLinkedRoots(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "config.json"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"config.json", "..", ".", "../keep", "nested/file"} {
		if err := RemoveFiles(dir, name); err == nil {
			t.Fatal("accepted unsafe cleanup", name)
		}
	}
	link := filepath.Join(t.TempDir(), "configuration")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if err := RemoveFiles(link, "config.json"); err == nil {
		t.Fatal("followed linked cleanup root")
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Fatal(err)
	}
}
