package main

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"io"
	"os"
)

func text(paths []string) {
	if len(paths) == 0 {
		fatalWithUsage()
	}

	expanded, client, err := getClientAndExpandedPaths(paths)
	if err != nil {
		fatal(err)
	}

	for _, p := range expanded {
		file, err := client.Open(p)
		if err != nil {
			fatal(err)
		}

		if file.Stat().IsDir() {
			file.Close()
			fatal(&os.PathError{"text", p, errors.New("file is a directory")})
		}

		reader := bufio.NewReader(file)
		magic, _ := reader.Peek(3)

		var source io.Reader = reader
		var gzipReader *gzip.Reader
		switch {
		case len(magic) >= 2 && magic[0] == 0x1f && magic[1] == 0x8b:
			gzipReader, err = gzip.NewReader(reader)
			if err != nil {
				file.Close()
				fatal(err)
			}
			source = gzipReader
		case len(magic) >= 3 && magic[0] == 'B' && magic[1] == 'Z' && magic[2] == 'h':
			source = bzip2.NewReader(reader)
		}

		if _, err := io.Copy(os.Stdout, source); err != nil {
			if gzipReader != nil {
				gzipReader.Close()
			}
			file.Close()
			fatal(err)
		}

		if gzipReader != nil {
			if err := gzipReader.Close(); err != nil {
				file.Close()
				fatal(err)
			}
		}

		if err := file.Close(); err != nil {
			fatal(err)
		}
	}
}
