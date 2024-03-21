package chunked

import (
	"bytes"
	"io"
	"testing"

	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

var (
	presentDigestInCache          string
	notPresentDigestInCache       string
	presentDigestInCacheBinary    []byte
	notPresentDigestInCacheBinary []byte
	preloadedCache                *cacheFile
	preloadedbloomFilter          *bloomFilter
	benchmarkN                    int = 100000
)

// Using 3 hashes functions and n/m = 10 gives a false positive rate of ~1.7%:
// https://pages.cs.wisc.edu/~cao/papers/summary-cache/node8.html
var (
	factorNM     int    = 10
	numberHashes uint32 = 3
)

func initCache(sizeCache int) (*cacheFile, string, string, *bloomFilter) {
	var tagsBuffer bytes.Buffer
	var vdata bytes.Buffer
	tags := [][]byte{}
	tagLen := 0
	digestLen := 64
	var presentDigest, notPresentDigest string

	bloomFilter := newBloomFilter(sizeCache*factorNM, numberHashes)

	digester := digest.Canonical.Digester()
	hash := digester.Hash()
	for i := 0; i < sizeCache; i++ {
		hash.Write([]byte("1"))
		d := digester.Digest().String()
		digestLen = len(d)
		presentDigest = d
		tag, err := appendTag([]byte(d), 0, 0)
		if err != nil {
			panic(err)
		}
		tagLen = len(tag)
		tags = append(tags, tag)
		bd, err := makeBinaryDigest(d)
		if err != nil {
			panic(err)
		}
		bloomFilter.add(bd)
	}

	hash.Write([]byte("1"))
	notPresentDigest = digester.Digest().String()

	writeCacheFileToWriter(io.Discard, bloomFilter, tags, tagLen, digestLen, vdata, &tagsBuffer)

	cache := &cacheFile{
		digestLen: digestLen,
		tagLen:    tagLen,
		tags:      tagsBuffer.Bytes(),
		vdata:     vdata.Bytes(),
	}
	return cache, presentDigest, notPresentDigest, bloomFilter
}

func init() {
	var err error
	preloadedCache, presentDigestInCache, notPresentDigestInCache, preloadedbloomFilter = initCache(10000)
	presentDigestInCacheBinary, err = makeBinaryDigest(presentDigestInCache)
	if err != nil {
		panic(err)
	}
	notPresentDigestInCacheBinary, err = makeBinaryDigest(notPresentDigestInCache)
	if err != nil {
		panic(err)
	}
}

func BenchmarkLookupBloomFilter(b *testing.B) {
	for i := 0; i < benchmarkN; i++ {
		if preloadedbloomFilter.maybeContains(notPresentDigestInCacheBinary) {
			findTag(notPresentDigestInCache, preloadedCache)
		}
		if preloadedbloomFilter.maybeContains(presentDigestInCacheBinary) {
			findTag(presentDigestInCache, preloadedCache)
		}
	}
}

func BenchmarkLookupBloomRaw(b *testing.B) {
	for i := 0; i < benchmarkN; i++ {
		findTag(notPresentDigestInCache, preloadedCache)
		findTag(presentDigestInCache, preloadedCache)
	}
}

func TestBloomFilter(t *testing.T) {
	bloomFilter := newBloomFilter(1000, 1)
	digester := digest.Canonical.Digester()
	hash := digester.Hash()
	for i := 0; i < 1000; i++ {
		hash.Write([]byte("1"))
		d := digester.Digest().String()
		bd, err := makeBinaryDigest(d)
		assert.NoError(t, err)
		contains := bloomFilter.maybeContains(bd)
		assert.False(t, contains)
	}
	for i := 0; i < 1000; i++ {
		hash.Write([]byte("1"))
		d := digester.Digest().String()
		bd, err := makeBinaryDigest(d)
		assert.NoError(t, err)
		bloomFilter.add(bd)

		contains := bloomFilter.maybeContains(bd)
		assert.True(t, contains)
	}
}
