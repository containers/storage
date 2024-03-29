package ioutils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteCloserWrapperClose(t *testing.T) {
	called := false
	writer := bytes.NewBuffer([]byte{})
	wrapper := NewWriteCloserWrapper(writer, func() error {
		called = true
		return nil
	})
	if err := wrapper.Close(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatalf("writeCloserWrapper should have call the anonymous function.")
	}
}

func TestNopWriteCloser(t *testing.T) {
	writer := bytes.NewBuffer([]byte{})
	wrapper := NopWriteCloser(writer)
	if err := wrapper.Close(); err != nil {
		t.Fatal("NopWriteCloser always return nil on Close.")
	}
}

func TestNopWriter(t *testing.T) {
	nw := &NopWriter{}
	l, err := nw.Write([]byte{'c'})
	if err != nil {
		t.Fatal(err)
	}
	if l != 1 {
		t.Fatalf("Expected 1 got %d", l)
	}
}

func TestWriteCounter(t *testing.T) {
	dummy1 := "This is a dummy string."
	dummy2 := "This is another dummy string."
	totalLength := int64(len(dummy1) + len(dummy2))

	reader1 := strings.NewReader(dummy1)
	reader2 := strings.NewReader(dummy2)

	var buffer bytes.Buffer
	wc := NewWriteCounter(&buffer)

	_, err := reader1.WriteTo(wc)
	require.NoError(t, err)
	_, err = reader2.WriteTo(wc)
	require.NoError(t, err)

	if wc.Count != totalLength {
		t.Errorf("Wrong count: %d vs. %d", wc.Count, totalLength)
	}

	if buffer.String() != dummy1+dummy2 {
		t.Error("Wrong message written")
	}
}
