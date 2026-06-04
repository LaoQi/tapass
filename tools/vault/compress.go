package vault

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
)

func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, fmt.Errorf("create flate writer: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("write compressed data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close flate writer: %w", err)
	}
	return buf.Bytes(), nil
}

func Decompress(data []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()

	result, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read compressed data: %w", err)
	}
	return result, nil
}
