//go:build linux

package minimal

import (
	"bytes"
	"encoding/binary"
	"errors"
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

	footer2, err := readFooterDataFromBlob(b)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, footer, footer2)
}

// readFooterDataFromBlob reads the zstd:chunked footer from the binary buffer.
func readFooterDataFromBlob(footer []byte) (ZstdChunkedFooterData, error) {
	var footerData ZstdChunkedFooterData

	if len(footer) < FooterSizeSupported {
		return footerData, errors.New("blob too small")
	}
	footerData.Offset = binary.LittleEndian.Uint64(footer[0:8])
	footerData.LengthCompressed = binary.LittleEndian.Uint64(footer[8:16])
	footerData.LengthUncompressed = binary.LittleEndian.Uint64(footer[16:24])
	footerData.ManifestType = binary.LittleEndian.Uint64(footer[24:32])
	footerData.OffsetTarSplit = binary.LittleEndian.Uint64(footer[32:40])
	footerData.LengthCompressedTarSplit = binary.LittleEndian.Uint64(footer[40:48])
	footerData.LengthUncompressedTarSplit = binary.LittleEndian.Uint64(footer[48:56])

	// the magic number is stored in the last 8 bytes
	if !bytes.Equal(ZstdChunkedFrameMagic, footer[len(footer)-len(ZstdChunkedFrameMagic):]) {
		return footerData, errors.New("invalid magic number")
	}
	return footerData, nil
}
