package chunked

import (
	"encoding/binary"
	"io"
	"unsafe"
)

type bloomFilter struct {
	bitArray []uint64
	k        uint32
}

func newBloomFilter(size uint, k uint32) *bloomFilter {
	numElements := (size + 63) / 64
	return &bloomFilter{
		bitArray: make([]uint64, numElements),
		k:        k,
	}
}

func newBloomFilterFromArray(bitArray []uint64, k uint32) *bloomFilter {
	return &bloomFilter{
		bitArray: bitArray,
		k:        k,
	}
}

func byteSliceAsUint64(b []byte) []uint64 {
	return *(*[]uint64)(unsafe.Pointer(&b))
}

func (bf *bloomFilter) hashFn(item []byte, seed uint32) (uint64, uint64) {
	var hash uint64 = 0

	// long a multiple of 8
	off := len(item) % 8
	for _, b := range item[:off] {
		hash = 17*hash + uint64(uint32(b)^seed)
	}
	// handle the remaining data as uint64
	itemsAsUint64 := byteSliceAsUint64([]byte(item[off:]))
	for _, b := range itemsAsUint64 {
		hash = 17*hash + uint64(uint32(b)^seed)
	}
	hash %= uint64(len(bf.bitArray) * 8)
	return hash / 64, uint64(1 << (hash % 64))
}

func (bf *bloomFilter) add(item []byte) {
	for i := uint32(0); i < bf.k; i++ {
		index, mask := bf.hashFn(item, i)
		bf.bitArray[index] |= mask
	}
}

func (bf *bloomFilter) maybeContains(item []byte) bool {
	for i := uint32(0); i < bf.k; i++ {
		index, mask := bf.hashFn(item, i)
		if bf.bitArray[index]&mask == 0 {
			return false
		}
	}
	return true
}

func (bf *bloomFilter) writeTo(writer io.Writer) error {
	if err := binary.Write(writer, binary.LittleEndian, uint64(len(bf.bitArray))); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint32(bf.k)); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, bf.bitArray); err != nil {
		return err
	}
	return nil
}

func readBloomFilter(reader io.Reader) (*bloomFilter, error) {
	var bloomFilterLen uint64
	var k uint32

	if err := binary.Read(reader, binary.LittleEndian, &bloomFilterLen); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &k); err != nil {
		return nil, err
	}
	bloomFilterArray := make([]uint64, bloomFilterLen)
	if err := binary.Read(reader, binary.LittleEndian, &bloomFilterArray); err != nil {
		return nil, err
	}
	return newBloomFilterFromArray(bloomFilterArray, k), nil
}
