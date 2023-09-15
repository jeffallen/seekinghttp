package main

import (
	"archive/tar"
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/jeffallen/seekinghttp"
)

const (
	LevelDebug int = -4
	LevelInfo  int = 0
)

// CustomLogger is a custom implementation of the Logger interface
type CustomLogger struct {
	Level int
}

// Infof logs an informational message
func (l CustomLogger) Infof(format string, args ...interface{}) {
	if l.Level <= LevelInfo {
		log.Printf(fmt.Sprintf("[INFO] %s", format), args...)
	}
}

// Debugf logs a debug message
func (l CustomLogger) Debugf(format string, args ...interface{}) {
	if l.Level <= LevelDebug {
		log.Printf(fmt.Sprintf("[DEBUG] %s", format), args...)
	}
}

func (l CustomLogger) Fatal(args ...interface{}) {
	log.Fatal(args...)
}

var debug = flag.Bool("debug", false, "enable verbose output")

func main() {
	flag.Parse()

	level := LevelInfo
	if *debug {
		level = LevelDebug
	}

	logger := &CustomLogger{Level: level}

	if flag.NArg() == 0 {
		logger.Fatal("Expected a URL as the first argument.")
	}

	r := seekinghttp.New(flag.Arg(0))
	r.SetLogger(logger)

	if strings.HasSuffix(flag.Arg(0), ".tar") {
		t := tar.NewReader(r)
		for {
			h, err := t.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				logger.Fatal(err)
			}
			logger.Infof("File: %s", h.Name)
		}
		return
	}

	if strings.HasSuffix(flag.Arg(0), ".zip") {
		sz, err := r.Size()
		if err != nil {
			logger.Fatal(err)
		}

		z, err := zip.NewReader(r, sz)
		if err != nil {
			logger.Fatal(err)
		}

		for _, f := range z.File {
			logger.Infof("File: %s", f.FileHeader.Name)
		}
		return
	}

	logger.Fatal("Unknown file type. URL does not end in .tar or .zip")
}
