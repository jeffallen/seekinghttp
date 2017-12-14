package seekinghttp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

// SeekingHTTP uses a series of HTTP GETs with Ranger headers
// to implement io.ReadSeeker and io.ReaderAt.
type SeekingHTTP struct {
	URL    string
	Client *http.Client
	url    *url.URL
	offset int64
}

// Compile-time check of interface implementations.
var _ io.ReadSeeker = (*SeekingHTTP)(nil)
var _ io.ReaderAt = (*SeekingHTTP)(nil)

// New initializes a SeekingHTTP for the given URL.
// The SeekingHTTP.Client field may be set before the first call
// to Read or Seek.
func New(url string) *SeekingHTTP {
	return &SeekingHTTP{
		URL:    url,
		offset: 0,
	}
}

func (s *SeekingHTTP) newreq() (*http.Request, error) {
	var err error
	if s.url == nil {
		s.url, err = url.Parse(s.URL)
		if err != nil {
			return nil, err
		}
	}
	return &http.Request{
		Method:     "GET",
		URL:        s.url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       nil,
		Host:       s.url.Host,
	}, nil
}

func fmtRange(from, l int64) string {
	var to int64
	if l == 0 {
		to = from
	} else {
		to = from + (l - 1)
	}
	return fmt.Sprintf("bytes=%v-%v", from, to)
}

// ReadAt reads len(buf) bytes into buf starting at offset off.
func (s *SeekingHTTP) ReadAt(buf []byte, off int64) (int, error) {
	log.Printf("got readat len %v off %v", len(buf), off)
	req, err := s.newreq()
	if err != nil {
		return 0, err
	}
	req.Header.Add("Range", fmtRange(off, int64(len(buf))))

	resp, err := s.Client.Do(req)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
		n, err := resp.Body.Read(buf)
		resp.Body.Close()
		// HTTP is trying to tell us, "that's all". Which is fine, but we don't
		// want callers to think it is EOF, it's not.
		if err == io.EOF && n == len(buf) {
			err = nil
		}
		//		log.Printf("%#v", buf)
		return n, err
	}
	return 0, io.EOF
}

// If they did not give us an HTTP Client, use the default one.
func (s *SeekingHTTP) init() error {
	if s.Client == nil {
		s.Client = http.DefaultClient
	}

	return nil
}

func (s *SeekingHTTP) Read(buf []byte) (int, error) {
	log.Printf("got read len %v", len(buf))
	if err := s.init(); err != nil {
		return 0, err
	}

	n, err := s.ReadAt(buf, s.offset)
	if err == nil {
		s.offset += int64(n)
	}

	return n, err
}

// Seek sets the offset for the next Read.
func (s *SeekingHTTP) Seek(offset int64, whence int) (int64, error) {
	log.Printf("got seek %v %v", offset, whence)
	switch whence {
	case os.SEEK_SET:
		s.offset = offset
	case os.SEEK_CUR:
		s.offset += offset
	case os.SEEK_END:
		return 0, errors.New("whence relative to end not impl yet")
	default:
		return 0, os.ErrInvalid
	}
	return s.offset, nil
}

// Size uses an HTTP HEAD to find out how many bytes are available in total.
func (s *SeekingHTTP) Size() (int64, error) {
	if err := s.init(); err != nil {
		return 0, err
	}

	req, err := s.newreq()
	if err != nil {
		return 0, err
	}
	req.Method = "HEAD"

	resp, err := s.Client.Do(req)
	if err != nil {
		return 0, err
	}

	if resp.ContentLength < 0 {
		return 0, errors.New("no content length for Size()")
	}
	log.Printf("size %v", resp.ContentLength)
	return resp.ContentLength, nil
}
