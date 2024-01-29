package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FilePersister will persist files. It abstracts away the where and how of
// writing files to the source destination.
type FilePersister interface {
	Persist(ctx context.Context, path string, data io.Reader) error
}

// LocalFilePersister will persist files to the local disk.
type LocalFilePersister struct{}

// Persist will write the contents of data to the local disk on the specified path.
// TODO: we should not write to disk here but put it on some queue for async disk writes.
func (l *LocalFilePersister) Persist(_ context.Context, path string, data io.Reader) (err error) {
	cp := filepath.Clean(path)

	dir := filepath.Dir(cp)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating a local directory %q: %w", dir, err)
	}

	f, err := os.OpenFile(cp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating a local file %q: %w", cp, err)
	}
	defer func() {
		tempErr := f.Close()
		// Only return the close error if there isn't already an existing error.
		if tempErr != nil && err == nil {
			err = fmt.Errorf("closing the local file %q: %w", cp, tempErr)
		}
	}()

	_, err = io.Copy(f, data)

	return
}
