package ioutils

import (
	"crypto/sha1"
	"encoding/hex"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBytesPipeRead(t *testing.T) {
	buf := NewBytesPipe()
	_, err := buf.Write([]byte("12"))
	require.NoError(t, err)
	_, err = buf.Write([]byte("34"))
	require.NoError(t, err)
	_, err = buf.Write([]byte("56"))
	require.NoError(t, err)
	_, err = buf.Write([]byte("78"))
	require.NoError(t, err)
	_, err = buf.Write([]byte("90"))
	require.NoError(t, err)
	rd := make([]byte, 4)
	n, err := buf.Read(rd)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("Wrong number of bytes read: %d, should be %d", n, 4)
	}
	if string(rd) != "1234" {
		t.Fatalf("Read %s, but must be %s", rd, "1234")
	}
	n, err = buf.Read(rd)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("Wrong number of bytes read: %d, should be %d", n, 4)
	}
	if string(rd) != "5678" {
		t.Fatalf("Read %s, but must be %s", rd, "5679")
	}
	n, err = buf.Read(rd)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("Wrong number of bytes read: %d, should be %d", n, 2)
	}
	if string(rd[:n]) != "90" {
		t.Fatalf("Read %s, but must be %s", rd, "90")
	}
}

func TestBytesPipeWrite(t *testing.T) {
	buf := NewBytesPipe()
	_, err := buf.Write([]byte("12"))
	require.NoError(t, err)
	_, err = buf.Write([]byte("34"))
	require.NoError(t, err)
	_, err = buf.Write([]byte("56"))
	require.NoError(t, err)
	_, err = buf.Write([]byte("78"))
	require.NoError(t, err)
	_, err = buf.Write([]byte("90"))
	require.NoError(t, err)
	if buf.buf[0].String() != "1234567890" {
		t.Fatalf("Buffer %q, must be %q", buf.buf[0].String(), "1234567890")
	}
}

// Write and read in different speeds/chunk sizes and check valid data is read.
func TestBytesPipeWriteRandomChunks(t *testing.T) {
	cases := []struct{ iterations, writesPerLoop, readsPerLoop int }{
		{100, 10, 1},
		{1000, 10, 5},
		{1000, 100, 0},
		{1000, 5, 6},
		{10000, 50, 25},
	}

	testMessage := []byte("this is a random string for testing")
	// random slice sizes to read and write
	writeChunks := []int{25, 35, 15, 20}
	readChunks := []int{5, 45, 20, 25}

	for _, c := range cases {
		// first pass: write directly to hash
		hash := sha1.New()
		for i := 0; i < c.iterations*c.writesPerLoop; i++ {
			if _, err := hash.Write(testMessage[:writeChunks[i%len(writeChunks)]]); err != nil {
				t.Fatal(err)
			}
		}
		expected := hex.EncodeToString(hash.Sum(nil))

		// write/read through buffer
		buf := NewBytesPipe()
		hash.Reset()

		done := make(chan struct{})

		go func() {
			// random delay before read starts
			<-time.After(rand.N(10 * time.Millisecond))
			for i := 0; ; i++ {
				p := make([]byte, readChunks[(c.iterations*c.readsPerLoop+i)%len(readChunks)])
				n, _ := buf.Read(p)
				if n == 0 {
					break
				}
				hash.Write(p[:n])
			}

			close(done)
		}()

		for i := 0; i < c.iterations; i++ {
			for w := 0; w < c.writesPerLoop; w++ {
				_, err := buf.Write(testMessage[:writeChunks[(i*c.writesPerLoop+w)%len(writeChunks)]])
				require.NoError(t, err)
			}
		}
		buf.Close()
		<-done

		actual := hex.EncodeToString(hash.Sum(nil))

		if expected != actual {
			t.Fatalf("BytesPipe returned invalid data. Expected checksum %v, got %v", expected, actual)
		}

	}
}

func BenchmarkBytesPipeWrite(b *testing.B) {
	testData := []byte("pretty short line, because why not?")
	for range b.N {
		readBuf := make([]byte, 1024)
		buf := NewBytesPipe()
		go func() {
			var err error
			for err == nil {
				_, err = buf.Read(readBuf)
			}
		}()
		for range 1000 {
			_, err := buf.Write(testData)
			require.NoError(b, err)
		}
		buf.Close()
	}
}

func BenchmarkBytesPipeRead(b *testing.B) {
	rd := make([]byte, 512)
	for range b.N {
		b.StopTimer()
		buf := NewBytesPipe()
		for range 500 {
			_, err := buf.Write(make([]byte, 1024))
			require.NoError(b, err)
		}
		b.StartTimer()
		for range 1000 {
			if n, _ := buf.Read(rd); n != 512 {
				b.Fatalf("Wrong number of bytes: %d", n)
			}
		}
	}
}
