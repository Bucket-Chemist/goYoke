package regression

import (
	"io/fs"
	"testing"

	"github.com/Bucket-Chemist/goYoke/defaults"
)

func TestEmbeddedFSSize_Under3MB(t *testing.T) {
	var totalSize int64
	err := fs.WalkDir(defaults.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		totalSize += info.Size()
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk embedded FS: %v", err)
	}

	const maxBytes = 3 * 1024 * 1024 // 3MB
	t.Logf("Embedded FS total size: %d bytes (%.1f MB)", totalSize, float64(totalSize)/1024/1024)
	if totalSize > maxBytes {
		t.Errorf("Embedded FS size %d bytes exceeds 3MB limit", totalSize)
	}
}
