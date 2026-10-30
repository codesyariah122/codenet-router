package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadLimitedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proc-sample")
	if err := os.WriteFile(path, []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := readLimitedFile(context.Background(), path, 5)
	if err != nil || string(data) != "12345" {
		t.Fatalf("hasil baca salah: %q %v", data, err)
	}
	if _, err := readLimitedFile(context.Background(), path, 4); err == nil {
		t.Fatal("batas ukuran tidak diterapkan")
	}
}

func TestReadLimitedFileMenghormatiContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := readLimitedFile(ctx, filepath.Join(t.TempDir(), "tidak-dibaca"), 10)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("mengharapkan context canceled, mendapat %v", err)
	}
}

func TestLimitedBuffer(t *testing.T) {
	buffer := limitedBuffer{max: 4}
	if _, err := buffer.Write([]byte("12345")); err == nil || buffer.String() != "1234" {
		t.Fatalf("output command tidak dibatasi: %q %v", buffer.String(), err)
	}
}
