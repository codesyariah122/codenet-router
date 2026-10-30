package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

func readLimitedFile(ctx context.Context, path string, maxBytes int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("membuka %s: %w", path, err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("membaca %s: %w", path, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("data %s melebihi batas", path)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

type limitedBuffer struct {
	bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	remaining := b.max - b.Len()
	if remaining <= 0 {
		return 0, errors.New("output command melebihi batas")
	}
	if len(data) > remaining {
		_, _ = b.Buffer.Write(data[:remaining])
		return remaining, errors.New("output command melebihi batas")
	}
	return b.Buffer.Write(data)
}
