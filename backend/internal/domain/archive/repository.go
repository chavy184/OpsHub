package archive

import (
	"context"
	"io"
)

type Parser interface {
	Parse(ctx context.Context, filename string, reader io.Reader, size int64) (*ArchiveAnalysis, error)
}
