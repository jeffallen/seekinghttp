package seekinghttp

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockHTTPClient is a mock implementation of the http.Client interface for testing purposes.
type MockHTTPClient struct{}

func (c *MockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	// Create a mock response for testing purposes.
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte("Mock HTTP response body"))),
	}
	return resp, nil
}

func TestReadAt(t *testing.T) {
	// Create a new SeekingHTTP instance with a mock HTTP client.
	s := New("https://example.com")
	s.Client = &MockHTTPClient{}

	// Define test cases.
	testCases := []struct {
		offset    int64
		bufSize   int
		expectLen int
		expectErr error
	}{
		{0, 10, 10, nil},
		{10, 1, 1, nil},
		{30, 30, 23, nil},
	}

	for _, tc := range testCases {
		buf := make([]byte, tc.bufSize)
		n, err := s.ReadAt(buf, tc.offset)

		assert.ErrorIs(t, err, tc.expectErr, "ReadAt(offset=%d, bufSize=%d) error = %v, expected error = %v", tc.offset, tc.bufSize, err, tc.expectErr)
		assert.Equal(t, n, tc.expectLen, "ReadAt(offset=%d, bufSize=%d) len = %d, expected len = %d", tc.offset, tc.bufSize, n, tc.expectLen)
	}
}
