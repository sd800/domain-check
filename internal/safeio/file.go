// Package safeio bounds reads of local application data.
package safeio

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

// ReadFile refuses symlinks and special files without blocking on a FIFO.
// Checking the opened descriptor also avoids a check-then-open replacement race.
func ReadFile(name string, limit int64) ([]byte, os.FileInfo, error) {
	f, err := os.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, nil, fmt.Errorf("application data must be a regular file of at most %d bytes", limit)
	}
	if info.Mode().Perm()&0022 != 0 {
		return nil, nil, fmt.Errorf("application data must not be writable by other users")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = fmt.Errorf("application data exceeds %d bytes", limit)
	}
	return data, info, err
}
