package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewBackupTempFileCreatesIsolatedPaths(t *testing.T) {
	fileName := "postgres_all_20260708_020000.tar.gz"

	dirA, pathA, err := newBackupTempFile(fileName)
	if err != nil {
		t.Fatalf("newBackupTempFile A failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dirA)
	})

	dirB, pathB, err := newBackupTempFile(fileName)
	if err != nil {
		t.Fatalf("newBackupTempFile B failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dirB)
	})

	if dirA == dirB {
		t.Fatalf("expected unique temp dirs, got %q", dirA)
	}
	if pathA == pathB {
		t.Fatalf("expected unique temp file paths, got %q", pathA)
	}
	if filepath.Base(pathA) != fileName || filepath.Base(pathB) != fileName {
		t.Fatalf("expected remote-facing file name to stay unchanged")
	}
}
