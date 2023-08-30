//go:build linux
// +build linux

package internal

import (
	"testing"
)

func TestIsZstdChunkedFrameMagic(t *testing.T) {
	b := append(ZstdChunkedFrameMagic[:], make([]byte, 200)...)
	if !IsZstdChunkedFrameMagic(b) {
		t.Fatal("Chunked frame magic not found")
	}
	// change a byte
	b[0] = -b[0]
	if IsZstdChunkedFrameMagic(b) {
		t.Fatal("Invalid chunked frame magic found")
	}
}
