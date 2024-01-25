package storage

import (
	"context"
	"io"

	"github.com/grafana/xk6-browser/log"
)

// FilePersister will persist files. It abstracts away the where and how of
// writing files to the source destination.
type FilePersister interface {
	Persist(ctx context.Context, logger *log.Logger, path string, data io.Reader) error
}
