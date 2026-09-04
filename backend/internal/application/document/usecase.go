package document

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// DocType 文档类型（动态，用户可自定义）
type DocType string

// 领域错误
var (
	ErrInvalidDocType = errors.New("无效的文档类型")
	ErrInvalidPath    = errors.New("路径非法")
	ErrFileExists     = errors.New("同名文件已存在")
	ErrFileNotFound   = errors.New("文件不存在")
	ErrFileTooLarge   = errors.New("文件过大")
	ErrFileTypeDenied = errors.New("文件类型不允许")
	ErrNotTextFile    = errors.New("非文本文件，无法预览")
	ErrTypeExists     = errors.New("文档类型已存在")
)

// FileNode 文件树节点
type FileNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"is_dir"`
	Size     int64       `json:"size"`
	ModTime  time.Time   `json:"mod_time"`
	Children []*FileNode `json:"children,omitempty"`
}

// FileContent 文件内容
type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

// 允许的文件扩展名
var allowedExtensions = map[string]bool{
	".md": true, ".txt": true, ".yaml": true, ".yml": true,
	".json": true, ".xml": true, ".html": true, ".css": true,
	".js": true, ".ts": true, ".tsx": true, ".jsx": true,
	".go": true, ".py": true, ".sh": true, ".sql": true,
	".toml": true, ".ini": true, ".conf": true, ".cfg": true,
	".env": true, ".log": true, ".csv": true,
	".doc": true, ".docx": true, ".pdf": true,
	".xls": true, ".xlsx": true, ".pptx": true,
}

// 可预览的文本扩展名
var previewableExtensions = map[string]bool{
	".md": true, ".txt": true, ".yaml": true, ".yml": true,
	".json": true, ".xml": true, ".html": true, ".css": true,
	".js": true, ".ts": true, ".tsx": true, ".jsx": true,
	".go": true, ".py": true, ".sh": true, ".sql": true,
	".toml": true, ".ini": true, ".conf": true, ".cfg": true,
	".env": true, ".log": true, ".csv": true,
}

type UseCase struct {
	baseDir     string
	maxFileSize int64
}

func NewUseCase(baseDir string, maxFileSize int64) *UseCase {
	return &UseCase{baseDir: baseDir, maxFileSize: maxFileSize}
}

// DocCategory 文档类型信息
type DocCategory struct {
	Name string `json:"name"` // 类型标识（目录名）
	Path string `json:"path"` // 相对路径
}

// ListTypes 列出所有文档类型（即 baseDir 下的一级子目录）
func (uc *UseCase) ListTypes(_ context.Context) ([]DocCategory, error) {
	_ = os.MkdirAll(uc.baseDir, 0755)
	entries, err := os.ReadDir(uc.baseDir)
	if err != nil {
		return nil, fmt.Errorf("读取文档根目录失败: %w", err)
	}
	var types []DocCategory
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			types = append(types, DocCategory{
				Name: entry.Name(),
				Path: entry.Name(),
			})
		}
	}
	return types, nil
}

// CreateType 创建新的文档类型（即创建一级目录）
func (uc *UseCase) CreateType(_ context.Context, name string) (*DocCategory, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return nil, ErrInvalidDocType
	}
	// 验证名称合法性：允许中文、字母、数字、-_
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != ' ' {
			return nil, ErrInvalidDocType
		}
	}

	dir := filepath.Join(uc.baseDir, name)
	if _, err := os.Stat(dir); err == nil {
		return nil, ErrTypeExists
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建文档类型目录失败: %w", err)
	}
	return &DocCategory{Name: name, Path: name}, nil
}

// DeleteType 删除文档类型（删除一级目录及所有内容）
func (uc *UseCase) DeleteType(_ context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return ErrInvalidDocType
	}
	dir := filepath.Join(uc.baseDir, name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return ErrFileNotFound
	}
	return os.RemoveAll(dir)
}

// GetTree 获取指定类型的文档树
func (uc *UseCase) GetTree(_ context.Context, docType DocType) ([]*FileNode, error) {
	root, err := uc.typeDir(docType)
	if err != nil {
		return nil, err
	}
	_ = os.MkdirAll(root, 0755)
	return uc.buildTree(root, "")
}

// Upload 上传文件
func (uc *UseCase) Upload(_ context.Context, docType DocType, dir string, filename string, reader io.Reader, overwrite bool) (*FileNode, error) {
	// 校验文件扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExtensions[ext] {
		return nil, ErrFileTypeDenied
	}

	// 清理文件名
	filename = sanitizeFilename(filename)
	if filename == "" {
		return nil, ErrInvalidPath
	}

	// 安全路径
	baseDir, err := uc.typeDir(docType)
	if err != nil {
		return nil, err
	}
	targetDir, err := uc.safePath(baseDir, dir)
	if err != nil {
		return nil, err
	}

	// 创建目录
	_ = os.MkdirAll(targetDir, 0755)

	fullPath := filepath.Join(targetDir, filename)
	// 确保最终路径仍在 baseDir 内
	if !strings.HasPrefix(fullPath, baseDir) {
		return nil, ErrInvalidPath
	}

	// 检查是否已存在
	if _, statErr := os.Stat(fullPath); statErr == nil {
		if !overwrite {
			return nil, ErrFileExists
		}
	}

	// 写入文件（限制大小）
	limitedReader := io.LimitReader(reader, uc.maxFileSize+1)
	f, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, limitedReader)
	if err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}
	if written > uc.maxFileSize {
		os.Remove(fullPath)
		return nil, ErrFileTooLarge
	}

	info, _ := os.Stat(fullPath)
	relPath := filepath.Join(dir, filename)
	return &FileNode{
		Name:    filename,
		Path:    filepath.ToSlash(relPath),
		IsDir:   false,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// Mkdir 创建文件夹
func (uc *UseCase) Mkdir(_ context.Context, docType DocType, path string) (*FileNode, error) {
	baseDir, err := uc.typeDir(docType)
	if err != nil {
		return nil, err
	}
	fullPath, err := uc.safePath(baseDir, path)
	if err != nil {
		return nil, err
	}

	if _, statErr := os.Stat(fullPath); statErr == nil {
		return nil, ErrFileExists
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	info, _ := os.Stat(fullPath)
	name := filepath.Base(path)
	return &FileNode{
		Name:    name,
		Path:    filepath.ToSlash(path),
		IsDir:   true,
		Size:    0,
		ModTime: info.ModTime(),
	}, nil
}

// Delete 删除文件或文件夹
func (uc *UseCase) Delete(_ context.Context, docType DocType, path string) error {
	if path == "" || path == "." || path == "/" {
		return ErrInvalidPath
	}
	baseDir, err := uc.typeDir(docType)
	if err != nil {
		return err
	}
	fullPath, err := uc.safePath(baseDir, path)
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
		return ErrFileNotFound
	}

	return os.RemoveAll(fullPath)
}

// GetContent 获取文件内容（仅文本文件）
func (uc *UseCase) GetContent(_ context.Context, docType DocType, path string) (*FileContent, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if !previewableExtensions[ext] {
		return nil, ErrNotTextFile
	}

	baseDir, err := uc.typeDir(docType)
	if err != nil {
		return nil, err
	}
	fullPath, err := uc.safePath(baseDir, path)
	if err != nil {
		return nil, err
	}

	info, statErr := os.Stat(fullPath)
	if os.IsNotExist(statErr) {
		return nil, ErrFileNotFound
	}
	if info.IsDir() {
		return nil, ErrInvalidPath
	}
	if info.Size() > 1*1024*1024 { // 预览限制 1MB
		return nil, ErrFileTooLarge
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return &FileContent{
		Path:    filepath.ToSlash(path),
		Content: string(data),
		Size:    info.Size(),
	}, nil
}

// GetFilePath 获取文件绝对路径（供下载使用）
func (uc *UseCase) GetFilePath(_ context.Context, docType DocType, path string) (string, error) {
	baseDir, err := uc.typeDir(docType)
	if err != nil {
		return "", err
	}
	fullPath, err := uc.safePath(baseDir, path)
	if err != nil {
		return "", err
	}

	info, statErr := os.Stat(fullPath)
	if os.IsNotExist(statErr) {
		return "", ErrFileNotFound
	}
	if info.IsDir() {
		return "", ErrInvalidPath
	}

	return fullPath, nil
}

// --- 内部方法 ---

func (uc *UseCase) typeDir(docType DocType) (string, error) {
	name := string(docType)
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", ErrInvalidDocType
	}
	dir := filepath.Join(uc.baseDir, name)
	// 确保最终路径在 baseDir 内
	absBase, _ := filepath.Abs(uc.baseDir)
	absDir, _ := filepath.Abs(dir)
	if !strings.HasPrefix(absDir, absBase) {
		return "", ErrInvalidDocType
	}
	return dir, nil
}

func (uc *UseCase) safePath(baseDir, relPath string) (string, error) {
	if relPath == "" {
		return baseDir, nil
	}
	cleaned := filepath.Clean(relPath)
	if strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", ErrInvalidPath
	}
	fullPath := filepath.Join(baseDir, cleaned)
	// 规范化后再次确认前缀
	absBase, _ := filepath.Abs(baseDir)
	absFull, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFull, absBase) {
		return "", ErrInvalidPath
	}
	return fullPath, nil
}

func (uc *UseCase) buildTree(rootDir, relPrefix string) ([]*FileNode, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	var nodes []*FileNode
	for _, entry := range entries {
		name := entry.Name()
		// 跳过隐藏文件
		if strings.HasPrefix(name, ".") {
			continue
		}

		relPath := filepath.Join(relPrefix, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		node := &FileNode{
			Name:    name,
			Path:    filepath.ToSlash(relPath),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		if entry.IsDir() {
			node.Size = 0
			children, err := uc.buildTree(filepath.Join(rootDir, name), relPath)
			if err == nil {
				node.Children = children
			}
		}

		nodes = append(nodes, node)
	}
	return nodes, nil
}

// sanitizeFilename 清理文件名，保留中文、字母、数字和安全字符
func sanitizeFilename(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' || r == ' ' {
			sb.WriteRune(r)
		}
	}
	result := strings.TrimSpace(sb.String())
	if strings.HasPrefix(result, ".") {
		return ""
	}
	if len(result) > 255 {
		result = result[:255]
	}
	return result
}
