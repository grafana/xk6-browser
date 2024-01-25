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
func (l *LocalFilePersister) Persist(_ context.Context, path string, data io.Reader) error {
	cp := filepath.Clean(path)

	dir := filepath.Dir(cp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating a local directory %q: %w", dir, err)
	}

	f, err := os.OpenFile(cp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating a local file %q: %w", cp, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logger.Errorf("LocalFilePersister:Persist", "closing the local file %q: %v", cp, err)
		}
	}()

	return write(f, data)
}

func write(w io.Writer, r io.Reader) error {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("writing to the local writer: %w", writeErr)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading from the reader: %w", err)
		}
	}

	return nil
}
