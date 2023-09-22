package seekinghttp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type logger struct {
	t *testing.T
}

func (l logger) Infof(format string, args ...interface{}) {
	l.t.Logf(fmt.Sprintf("[INFO] %s", format), args...)
}
func (l logger) Debugf(format string, args ...interface{}) {
	l.t.Logf(fmt.Sprintf("[DEBUG] %s", format), args...)
}

// MockHTTPClient is a mock implementation of the http.Client interface for testing purposes.
type MockHTTPClient struct {
	str    string
	numReq int
}

func (c *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	start := 0
	end := 0
	r := req.Header["Range"][0]
	switch r {
	case "bytes=0-99":
		start = 0
		end = 99
	case "bytes=30-329":
		start = 30
		end = 329
	case "bytes=10-109":
		start = 10
		end = 109
	case "bytes=20-119":
		start = 20
		end = 119
	default:
		panic(fmt.Sprintf("unknown range: %s", r))
	}

	if end > len(c.str) {
		end = len(c.str)
	}
	if start > end {
		start = end
	}

	// Create a mock response for testing purposes.
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(c.str[start:end]))),
	}
	c.numReq++
	return resp, nil
}

func TestReadAt(t *testing.T) {
	// Create a new SeekingHTTP instance with a mock HTTP client.
	s := New("https://example.com")
	m := &MockHTTPClient{str: "Mock HTTP response body"}
	s.Client = m
	s.Logger = &logger{t: t}

	// Define test cases.
	testCases := []struct {
		offset    int64
		bufSize   int
		expectLen int
		expectErr error
	}{
		{0, 10, 10, nil},
		{10, 1, 1, nil},
		{30, 30, 0, nil},
		{-1, 0, 0, io.EOF},
	}

	for _, tc := range testCases {
		buf := make([]byte, tc.bufSize)
		n, err := s.ReadAt(buf, tc.offset)

		assert.ErrorIs(t, tc.expectErr, err, "ReadAt(offset=%d, bufSize=%d) error = %v, expected error = %v", tc.offset, tc.bufSize, err, tc.expectErr)
		assert.Equal(t, tc.expectLen, n, "ReadAt(offset=%d, bufSize=%d) len = %d, expected len = %d", tc.offset, tc.bufSize, n, tc.expectLen)
	}
	// expect 2 reads: one to load the cache, and one to look for bytes past the end for the seek to 30.
	assert.Equal(t, 2, m.numReq)
}

func TestReadNothing(t *testing.T) {
	// Create a new SeekingHTTP instance with a mock HTTP client.
	s := New("https://example.com")
	s.Client = &MockHTTPClient{str: ""}
	s.Logger = &logger{t: t}

	buf := make([]byte, 10)
	n, err := s.Read(buf)
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, 0, n)
}

func TestReadOffEnd(t *testing.T) {
	// Create a new SeekingHTTP instance with a mock HTTP client.
	s := New("https://example.com")
	s.Client = &MockHTTPClient{str: "0123456789abcdefghij"}
	s.Logger = &logger{t: t}

	buf := make([]byte, 10)
	n, err := s.Read(buf)
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, n, len(buf))
	assert.Equal(t, "0123456789", string(buf))
	assert.Equal(t, int64(10), s.offset)

	n, err = s.Read(buf)
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, n, len(buf))
	assert.Equal(t, "abcdefghij", string(buf))
	assert.Equal(t, int64(20), s.offset)

	n, err = s.Read(buf)
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, 0, n)
	assert.Equal(t, int64(20), s.offset)

}
