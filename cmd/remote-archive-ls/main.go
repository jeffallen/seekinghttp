package main

import (
	"archive/tar"
	"archive/zip"
	"flag"
	"io"
	"log"
	"strings"

	"github.com/jeffallen/seekinghttp"
)

func main() {
	flag.Parse()
	r := seekinghttp.New(flag.Arg(0))

	if strings.HasSuffix(flag.Arg(0), ".tar") {
		t := tar.NewReader(r)
		for {
			h, err := t.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			log.Print("File: ", h.Name)
		}
		return
	}

	if strings.HasSuffix(flag.Arg(0), ".zip") {
		sz, err := r.Size()
		if err != nil {
			log.Fatal(err)
		}

		z, err := zip.NewReader(r, sz)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range z.File {
			log.Print("File: ", f.FileHeader.Name)
		}
		return
	}

	log.Fatal("Unknown file type.")
}
