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

// SeekingHTTP uses a series of HTTP GETs to implement
// io.ReadSeeker `nd io.ReaderAt.
type SeekingHTTP struct {
	URL    string
	url    *url.URL
	Client *http.Client
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

func (s *SeekingHTTP) fmtRange(l int64) string {
	from := s.offset
	var to int64
	if l == 0 {
		to = from
	} else {
		to = from + (l - 1)
	}
	return fmt.Sprintf("bytes=%v-%v", from, to)
}

// ReadAt reads len(buf) bytes into buf starting at offset off.
// It is not safe to call ReadAt and Seek concurrently.
func (s *SeekingHTTP) ReadAt(buf []byte, off int64) (int, error) {
	cur, _ := s.Seek(0, os.SEEK_CUR)
	s.Seek(off, os.SEEK_SET)
	defer s.Seek(cur, os.SEEK_SET)
	return s.Read(buf)
}

func (s *SeekingHTTP) init() error {
	if s.Client == nil {
		s.Client = http.DefaultClient
	}

	return nil
}

func (s *SeekingHTTP) Read(buf []byte) (int, error) {
	//log.Printf("got read for %v bytes @%v", len(buf), s.offset)
	if err := s.init(); err != nil {
		return 0, err
	}

	req, err := s.newreq()
	if err != nil {
		return 0, err
	}
	req.Header.Del("Range")
	req.Header.Add("Range", s.fmtRange(int64(len(buf))))

	resp, err := s.Client.Do(req)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
		n, err := resp.Body.Read(buf)
		resp.Body.Close()
		s.offset += int64(n)
		// HTTP is trying to tell us, "that's all". Which is fine, but we don't
		// want callers to think it is EOF, it's not.
		if err == io.EOF && n == len(buf) {
			err = nil
		}
		return n, err
	}
	log.Printf("Status code: %v", resp.StatusCode)
	return 0, io.EOF
}

// Seek sets the offset for the next Read.
func (s *SeekingHTTP) Seek(offset int64, whence int) (int64, error) {
	//log.Printf("got seek %v %v", offset, whence)
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
	return resp.ContentLength, nil
}
