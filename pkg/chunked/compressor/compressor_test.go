package compressor

import (
	"bufio"
	"bytes"
	"io"
	"testing"
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
