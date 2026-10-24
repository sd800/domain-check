package safeio

import (
	"fmt"
	"os"
	"path/filepath"
)

// RemoveFiles unlinks only named regular files or links, without traversing links
// inside the directory. Other entries and the containing directory are preserved.
func RemoveFiles(dir string, names ...string) error {
	if dir == "" {
		return nil
	}
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to clean a non-directory: %s", dir)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	opened, err := root.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(info, opened) {
		return fmt.Errorf("cleanup directory changed: %s", dir)
	}
	for _, name := range names {
		if name == "." || name == ".." || filepath.Base(name) != name {
			return fmt.Errorf("invalid cleanup filename")
		}
		info, err := root.Lstat(name)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to remove a non-file: %s", filepath.Join(dir, name))
		}
		if err := root.Remove(name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
