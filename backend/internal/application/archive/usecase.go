package archive

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	domainArchive "ops-hub/internal/domain/archive"
)

const defaultMaxArchiveSize int64 = 200 * 1024 * 1024

type UseCase struct {
	parser         domainArchive.Parser
	maxArchiveSize int64
}

func NewUseCase(parser domainArchive.Parser) *UseCase {
	return &UseCase{parser: parser, maxArchiveSize: defaultMaxArchiveSize}
}

func (uc *UseCase) Analyze(ctx context.Context, filename string, reader io.Reader, size int64) (*domainArchive.ArchiveAnalysis, error) {
	if size > uc.maxArchiveSize {
		return nil, domainArchive.ErrFileTooLarge
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".deb" {
		return nil, domainArchive.ErrUnsupportedFile
	}
	analysis, err := uc.parser.Parse(ctx, filename, reader, size)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}
	return analysis, nil
}
