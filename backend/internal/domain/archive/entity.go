package archive

import "errors"

var (
	ErrUnsupportedFile = errors.New("不支持的文件类型")
	ErrInvalidArchive  = errors.New("压缩包格式无效")
	ErrFileTooLarge    = errors.New("文件过大")
)

type FileKind string

const (
	FileKindFile FileKind = "file"
	FileKindDir  FileKind = "dir"
)

type ArchiveNode struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Kind     FileKind       `json:"kind"`
	Section  string         `json:"section"`
	Size     int64          `json:"size"`
	Mode     string         `json:"mode,omitempty"`
	Content  string         `json:"content,omitempty"`
	Children []*ArchiveNode `json:"children,omitempty"`
}

type ArchiveAnalysis struct {
	Filename     string            `json:"filename"`
	Type         string            `json:"type"`
	Size         int64             `json:"size"`
	DebianBinary string            `json:"debian_binary,omitempty"`
	PackageInfo  map[string]string `json:"package_info,omitempty"`
	Files        []*ArchiveNode    `json:"files"`
	Warnings     []string          `json:"warnings,omitempty"`
}
