//go:build linux
// +build linux

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAndReadFooter(t *testing.T) {
	footer := ZstdChunkedFooterData{
		ManifestType:               1,
		Offset:                     2,
		LengthCompressed:           3,
		LengthUncompressed:         4,
		OffsetTarSplit:             5,
		LengthCompressedTarSplit:   6,
		LengthUncompressedTarSplit: 7,
		ChecksumAnnotationTarSplit: "", // unused
	}
	b := footerDataToBlob(footer)
	assert.Len(t, b, FooterSizeSupported)

	footer2, err := ReadFooterDataFromBlob(b)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, footer, footer2)
}
