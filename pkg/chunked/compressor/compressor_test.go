package compressor

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHole(t *testing.T) {
	data := []byte("\x00\x00\x00\x00\x00")

	hf := &holesFinder{
		threshold: 1,
		reader:    bufio.NewReader(bytes.NewReader(data)),
	}

	hole, _, err := hf.readByte()
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	if hole != 5 {
		t.Error("expected hole not found")
	}

	if _, _, err := hf.readByte(); err != io.EOF {
		t.Errorf("EOF not found")
	}

	hf = &holesFinder{
		threshold: 1000,
		reader:    bufio.NewReader(bytes.NewReader(data)),
	}
	for range 5 {
		hole, b, err := hf.readByte()
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		if hole != 0 {
			t.Error("hole found")
		}
		if b != 0 {
			t.Error("wrong read")
		}
	}
	if _, _, err := hf.readByte(); err != io.EOF {
		t.Error("didn't receive EOF")
	}
}

func TestTwoHoles(t *testing.T) {
	data := []byte("\x00\x00\x00\x00\x00FOO\x00\x00\x00\x00\x00")

	hf := &holesFinder{
		threshold: 2,
		reader:    bufio.NewReader(bytes.NewReader(data)),
	}

	hole, _, err := hf.readByte()
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	if hole != 5 {
		t.Error("hole not found")
	}

	for _, e := range []byte("FOO") {
		hole, c, err := hf.readByte()
		if err != nil {
			t.Errorf("got error: %v", err)
		}
		if hole != 0 {
			t.Error("hole found")
		}
		if c != e {
			t.Errorf("wrong byte read %v instead of %v", c, e)
		}
	}
	hole, _, err = hf.readByte()
	if err != nil {
		t.Errorf("got error: %v", err)
	}
	if hole != 5 {
		t.Error("expected hole not found")
	}

	if _, _, err := hf.readByte(); err != io.EOF {
		t.Error("didn't receive EOF")
	}
}

func TestNoCompressionWrite(t *testing.T) {
	var buf bytes.Buffer
	nc := &noCompression{dest: &buf}

	data := []byte("hello world")
	n, err := nc.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buf.Bytes())

	data2 := []byte(" again")
	n, err = nc.Write(data2)
	assert.NoError(t, err)
	assert.Equal(t, len(data2), n)
	assert.Equal(t, append(data, data2...), buf.Bytes())
}

func TestNoCompressionClose(t *testing.T) {
	var buf bytes.Buffer
	nc := &noCompression{dest: &buf}
	err := nc.Close()
	assert.NoError(t, err)
}

func TestNoCompressionFlush(t *testing.T) {
	var buf bytes.Buffer
	nc := &noCompression{dest: &buf}
	err := nc.Flush()
	assert.NoError(t, err)
}

func TestNoCompressionReset(t *testing.T) {
	var buf1 bytes.Buffer
	nc := &noCompression{dest: &buf1}

	data1 := []byte("initial data")
	_, err := nc.Write(data1)
	assert.NoError(t, err)
	assert.Equal(t, data1, buf1.Bytes())

	err = nc.Close()
	assert.NoError(t, err)

	var buf2 bytes.Buffer
	nc.Reset(&buf2)

	data2 := []byte("new data")
	_, err = nc.Write(data2)
	assert.NoError(t, err)

	assert.Equal(t, data1, buf1.Bytes(), "Buffer 1 should remain unchanged")
	assert.Equal(t, data2, buf2.Bytes(), "Buffer 2 should contain the new data")

	err = nc.Close()
	assert.NoError(t, err)

	// Test Reset with nil, though Write would panic, Reset itself should work
	nc.Reset(nil)
	assert.Nil(t, nc.dest)
}

// Mock writer that returns an error on Write
type errorWriter struct{}

func (ew *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("mock write error")
}

func TestNoCompressionWriteError(t *testing.T) {
	ew := &errorWriter{}
	nc := &noCompression{dest: ew}

	data := []byte("hello world")
	n, err := nc.Write(data)
	assert.Error(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, "mock write error", err.Error())
}
