package archive

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	domainArchive "ops-hub/internal/domain/archive"
)

const (
	arMagic               = "!<arch>\n"
	arHeaderSize          = 60
	maxTextPreviewBytes   = 128 * 1024
	maxPackageFieldLength = 4096
)

type DebParser struct{}

func NewDebParser() *DebParser {
	return &DebParser{}
}

func (p *DebParser) Parse(ctx context.Context, filename string, reader io.Reader, size int64) (*domainArchive.ArchiveAnalysis, error) {
	buf := bufio.NewReader(reader)
	magic := make([]byte, len(arMagic))
	if _, err := io.ReadFull(buf, magic); err != nil {
		return nil, domainArchive.ErrInvalidArchive
	}
	if string(magic) != arMagic {
		return nil, domainArchive.ErrInvalidArchive
	}

	analysis := &domainArchive.ArchiveAnalysis{
		Filename:    filename,
		Type:        "deb",
		Size:        size,
		PackageInfo: map[string]string{},
		Files:       []*domainArchive.ArchiveNode{},
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		member, err := readArMember(buf)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		root := &domainArchive.ArchiveNode{
			Name:    member.Name,
			Path:    member.Name,
			Kind:    domainArchive.FileKindFile,
			Section: "ar",
			Size:    int64(len(member.Data)),
		}
		analysis.Files = append(analysis.Files, root)

		switch {
		case member.Name == "debian-binary":
			analysis.DebianBinary = strings.TrimSpace(string(member.Data))
			if isText(member.Data) {
				root.Content = string(member.Data)
			}
		case strings.HasPrefix(member.Name, "control.tar"):
			children, controlInfo, warnings := parseTarMember(member.Name, member.Data, "control")
			root.Kind = domainArchive.FileKindDir
			root.Children = children
			mergePackageInfo(analysis.PackageInfo, controlInfo)
			analysis.Warnings = append(analysis.Warnings, warnings...)
		case strings.HasPrefix(member.Name, "data.tar"):
			children, _, warnings := parseTarMember(member.Name, member.Data, "data")
			root.Kind = domainArchive.FileKindDir
			root.Children = children
			analysis.Warnings = append(analysis.Warnings, warnings...)
		}
	}

	if len(analysis.PackageInfo) == 0 {
		analysis.PackageInfo = nil
	}
	return analysis, nil
}

type arMember struct {
	Name string
	Data []byte
}

func readArMember(r *bufio.Reader) (*arMember, error) {
	header := make([]byte, arHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, domainArchive.ErrInvalidArchive
		}
		return nil, err
	}
	if string(header[58:60]) != "`\n" {
		return nil, domainArchive.ErrInvalidArchive
	}
	name := normalizeArName(string(header[0:16]))
	sizeText := strings.TrimSpace(string(header[48:58]))
	size, err := strconv.Atoi(sizeText)
	if err != nil || size < 0 {
		return nil, domainArchive.ErrInvalidArchive
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, domainArchive.ErrInvalidArchive
	}
	if size%2 == 1 {
		if _, err := r.ReadByte(); err != nil {
			return nil, domainArchive.ErrInvalidArchive
		}
	}
	return &arMember{Name: name, Data: data}, nil
}

func normalizeArName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, "/")
	return name
}

func parseTarMember(filename string, data []byte, section string) ([]*domainArchive.ArchiveNode, map[string]string, []string) {
	reader, err := compressedTarReader(filename, data)
	if err != nil {
		return nil, nil, []string{fmt.Sprintf("%s 解压失败: %v", filename, err)}
	}
	tr := tar.NewReader(reader)
	root := map[string]*domainArchive.ArchiveNode{}
	var nodes []*domainArchive.ArchiveNode
	info := map[string]string{}
	var warnings []string

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s 读取 tar 条目失败: %v", filename, err))
			break
		}
		cleanPath := cleanArchivePath(hdr.Name)
		if cleanPath == "" {
			continue
		}
		ensureParents(root, &nodes, cleanPath, section)
		node := &domainArchive.ArchiveNode{
			Name:    path.Base(cleanPath),
			Path:    cleanPath,
			Kind:    domainArchive.FileKindFile,
			Section: section,
			Size:    hdr.Size,
			Mode:    fmt.Sprintf("%o", hdr.FileInfo().Mode().Perm()),
		}
		if hdr.FileInfo().IsDir() {
			node.Kind = domainArchive.FileKindDir
		} else if shouldPreviewText(cleanPath, hdr.Size) {
			content, err := readSmallText(tr, hdr.Size)
			if err == nil {
				node.Content = content
				if section == "control" && cleanPath == "control" {
					info = parseDebControl(content)
				}
			}
		}
		addNode(root, &nodes, node)
	}
	return nodes, info, warnings
}

func compressedTarReader(filename string, data []byte) (io.Reader, error) {
	r := bytes.NewReader(data)
	switch {
	case strings.HasSuffix(filename, ".gz"):
		return gzip.NewReader(r)
	case strings.HasSuffix(filename, ".xz"):
		out, err := decompressXZ(data)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(out), nil
	case strings.HasSuffix(filename, ".tar"):
		return r, nil
	default:
		return nil, fmt.Errorf("暂不支持该 tar 压缩格式")
	}
}

func decompressXZ(data []byte) ([]byte, error) {
	cmd := exec.Command("xz", "-dc")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xz 解压失败，请确认服务器已安装 xz: %w", err)
	}
	return out, nil
}

func cleanArchivePath(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	p = strings.TrimPrefix(p, "./")
	p = path.Clean("/" + p)
	p = strings.TrimPrefix(p, "/")
	if p == "." || p == "" || strings.HasPrefix(p, "../") {
		return ""
	}
	return p
}

func ensureParents(index map[string]*domainArchive.ArchiveNode, roots *[]*domainArchive.ArchiveNode, p, section string) {
	dir := path.Dir(p)
	if dir == "." || dir == "" {
		return
	}
	parts := strings.Split(dir, "/")
	for i := range parts {
		parentPath := strings.Join(parts[:i+1], "/")
		if _, ok := index[parentPath]; ok {
			continue
		}
		addNode(index, roots, &domainArchive.ArchiveNode{
			Name:    path.Base(parentPath),
			Path:    parentPath,
			Kind:    domainArchive.FileKindDir,
			Section: section,
		})
	}
}

func addNode(index map[string]*domainArchive.ArchiveNode, roots *[]*domainArchive.ArchiveNode, node *domainArchive.ArchiveNode) {
	if existing, ok := index[node.Path]; ok {
		existing.Kind = node.Kind
		existing.Size = node.Size
		existing.Mode = node.Mode
		existing.Content = node.Content
		return
	}
	index[node.Path] = node
	parentPath := path.Dir(node.Path)
	if parentPath == "." || parentPath == "" {
		*roots = append(*roots, node)
		return
	}
	parent, ok := index[parentPath]
	if !ok {
		*roots = append(*roots, node)
		return
	}
	parent.Children = append(parent.Children, node)
}

func shouldPreviewText(p string, size int64) bool {
	if size < 0 || size > maxTextPreviewBytes {
		return false
	}
	name := path.Base(p)
	ext := strings.ToLower(path.Ext(name))
	if ext == ".txt" || ext == ".md" || ext == ".json" || ext == ".xml" || ext == ".conf" || ext == ".cfg" || ext == ".sh" || ext == ".service" {
		return true
	}
	switch name {
	case "control", "md5sums", "conffiles", "preinst", "postinst", "prerm", "postrm", "config", "templates", "shlibs", "triggers":
		return true
	default:
		return false
	}
}

func readSmallText(r io.Reader, size int64) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, size))
	if err != nil {
		return "", err
	}
	if !isText(data) {
		return "", nil
	}
	return string(data), nil
}

func isText(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	return utf8.Valid(data)
}

func parseDebControl(content string) map[string]string {
	result := map[string]string{}
	var current string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if current != "" && len(result[current]) < maxPackageFieldLength {
				result[current] += "\n" + strings.TrimSpace(line)
			}
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key != "" {
			current = key
			result[key] = val
		}
	}
	return result
}

func mergePackageInfo(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}
