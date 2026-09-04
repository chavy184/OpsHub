package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"testing"
	"time"

	domainArchive "ops-hub/internal/domain/archive"
)

func TestDebParserParseGzipDeb(t *testing.T) {
	deb := buildTestDeb(t)
	parser := NewDebParser()

	result, err := parser.Parse(context.Background(), "demo.deb", bytes.NewReader(deb), int64(len(deb)))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.DebianBinary != "2.0" {
		t.Fatalf("DebianBinary = %q", result.DebianBinary)
	}
	if result.PackageInfo["Package"] != "demo" {
		t.Fatalf("Package = %q", result.PackageInfo["Package"])
	}
	if len(result.Files) != 3 {
		t.Fatalf("Files length = %d", len(result.Files))
	}

	var foundReadme bool
	for _, root := range result.Files {
		if root.Name != "data.tar.gz" {
			continue
		}
		foundReadme = hasNodeContent(root.Children, "usr/share/doc/demo/readme.txt", "hello\n")
	}
	if !foundReadme {
		t.Fatal("readme text content not found")
	}
}

func hasNodeContent(nodes []*domainArchive.ArchiveNode, targetPath, content string) bool {
	for _, node := range nodes {
		if node.Path == targetPath && node.Content == content {
			return true
		}
		if hasNodeContent(node.Children, targetPath, content) {
			return true
		}
	}
	return false
}

func buildTestDeb(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString(arMagic)
	writeArMember(t, &buf, "debian-binary", []byte("2.0\n"))
	writeArMember(t, &buf, "control.tar.gz", buildTarGz(t, map[string]string{
		"control": "Package: demo\nVersion: 1.0.0\nArchitecture: amd64\n",
	}))
	writeArMember(t, &buf, "data.tar.gz", buildTarGz(t, map[string]string{
		"usr/share/doc/demo/readme.txt": "hello\n",
	}))
	return buf.Bytes()
}

func writeArMember(t *testing.T, buf *bytes.Buffer, name string, data []byte) {
	t.Helper()
	header := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`\n", name+"/", time.Now().Unix(), 0, 0, 0644, len(data))
	if len(header) != arHeaderSize {
		t.Fatalf("ar header length = %d", len(header))
	}
	buf.WriteString(header)
	buf.Write(data)
	if len(data)%2 == 1 {
		buf.WriteByte('\n')
	}
}

func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		data := []byte(content)
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(data)),
		}); err != nil {
			t.Fatalf("WriteHeader() error = %v", err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close() error = %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip Close() error = %v", err)
	}
	return buf.Bytes()
}
