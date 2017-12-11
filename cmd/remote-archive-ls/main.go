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

var debug = flag.Bool("debug", false, "enable verbose output")

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatal("Expected a URL as the first argument.")
	}

	r := seekinghttp.New(flag.Arg(0))
	r.Debug = *debug

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
			fmt.Println("File:", h.Name)
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

	log.Fatal("Unknown file type. URL does not end in .tar or .zip")
}
