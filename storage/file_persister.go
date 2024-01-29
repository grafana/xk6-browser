package storage

import (
	"context"
	"io"
)

// FilePersister will persist files. It abstracts away the where and how of
// writing files to the source destination.
type FilePersister interface {
	Persist(ctx context.Context, path string, data io.Reader) error
}
