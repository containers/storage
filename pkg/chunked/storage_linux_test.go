package chunked

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock for ImageSourceSeekable
type mockImageSource struct {
	streams chan io.ReadCloser
	errors  chan error
}

func (m *mockImageSource) GetBlobAt(chunks []ImageSourceChunk) (chan io.ReadCloser, chan error, error) {
	return m.streams, m.errors, nil
}

type mockReadCloser struct {
	io.Reader
	closed bool
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

func mockReadCloserFromContent(content string) *mockReadCloser {
	return &mockReadCloser{Reader: bytes.NewBufferString(content), closed: false}
}

func TestGetBlobAtNormalOperation(t *testing.T) {
	errors := make(chan error, 1)
	expectedStreams := []string{"stream1", "stream2"}
	streamsObjs := []*mockReadCloser{
		mockReadCloserFromContent(expectedStreams[0]),
		mockReadCloserFromContent(expectedStreams[1]),
	}
	streams := make(chan io.ReadCloser, len(streamsObjs))

	for _, s := range streamsObjs {
		streams <- s
	}
	close(streams)
	close(errors)

	is := &mockImageSource{streams: streams, errors: errors}

	chunks := []ImageSourceChunk{
		{Offset: 0, Length: 1},
		{Offset: 1, Length: 1},
	}

	resultChan, err := getBlobAt(is, chunks...)
	require.NoError(t, err)

	i := 0
	for result := range resultChan {
		assert.NoError(t, result.err)
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(result.stream)
		result.stream.Close()
		assert.Equal(t, expectedStreams[i], buf.String())
		i++
	}
	assert.Len(t, expectedStreams, i)
	for _, s := range streamsObjs {
		assert.True(t, s.closed)
	}
}

func TestGetBlobAtMaxStreams(t *testing.T) {
	streams := make(chan io.ReadCloser, 5)
	errors := make(chan error)

	streamsObjs := []*mockReadCloser{}

	for i := 1; i <= 5; i++ {
		s := mockReadCloserFromContent(fmt.Sprintf("stream%d", i))
		streamsObjs = append(streamsObjs, s)
		streams <- s
	}
	close(streams)
	close(errors)

	is := &mockImageSource{streams: streams, errors: errors}

	chunks := []ImageSourceChunk{
		{Offset: 0, Length: 1},
		{Offset: 1, Length: 1},
		{Offset: 2, Length: 1},
	}

	resultChan, err := getBlobAt(is, chunks...)
	require.NoError(t, err)

	count := 0
	receivedErr := false
	for result := range resultChan {
		if result.err != nil {
			receivedErr = true
		} else {
			result.stream.Close()
			count++
		}
	}
	assert.True(t, receivedErr)
	assert.Equal(t, 3, count)
	for _, s := range streamsObjs {
		assert.True(t, s.closed)
	}
}

func TestGetBlobAtWithErrors(t *testing.T) {
	streams := make(chan io.ReadCloser)
	errorsC := make(chan error, 2)

	errorsC <- errors.New("error1")
	errorsC <- errors.New("error2")
	close(streams)
	close(errorsC)

	is := &mockImageSource{streams: streams, errors: errorsC}

	resultChan, err := getBlobAt(is)
	require.NoError(t, err)

	expectedErrors := []string{"error1", "error2"}
	i := 0
	for result := range resultChan {
		assert.Nil(t, result.stream)
		assert.NotNil(t, result.err)
		if result.err != nil {
			assert.Equal(t, expectedErrors[i], result.err.Error())
		}
		i++
	}
	assert.Equal(t, len(expectedErrors), i)
}

func TestGetBlobAtMixedStreamsAndErrors(t *testing.T) {
	streams := make(chan io.ReadCloser, 2)
	errorsC := make(chan error, 1)

	streams <- mockReadCloserFromContent("stream1")
	errorsC <- errors.New("error1")
	close(streams)
	close(errorsC)

	is := &mockImageSource{streams: streams, errors: errorsC}

	resultChan, err := getBlobAt(is)
	require.NoError(t, err)

	var receivedStreams int
	var receivedErrors int
	for result := range resultChan {
		if result.err != nil {
			receivedErrors++
		} else {
			receivedStreams++
		}
	}
	assert.Equal(t, 0, receivedStreams)
	assert.Equal(t, 2, receivedErrors)
}
