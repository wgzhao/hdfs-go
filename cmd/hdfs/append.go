package main

import (
	"errors"
	"io"
	"os"

	"github.com/colinmarc/hdfs/v2"
)

func appendToFile(args []string) {
	if len(args) < 2 {
		fatalWithUsage()
	}

	dests, nn, err := normalizePaths(args[len(args)-1:])
	if err != nil {
		fatal(err)
	}

	dest := dests[0]
	client, err := getClient(nn)
	if err != nil {
		fatal(err)
	}

	var writer *hdfs.FileWriter
	info, err := client.Stat(dest)
	if err == nil {
		if info.IsDir() {
			fatal(&os.PathError{"appendToFile", dest, errors.New("is a directory")})
		}

		writer, err = client.Append(dest)
	} else if os.IsNotExist(err) {
		writer, err = client.Create(dest)
	} else {
		fatal(err)
	}

	if err != nil {
		fatal(err)
	}

	for _, source := range args[:len(args)-1] {
		var reader io.Reader
		if source == "-" {
			reader = os.Stdin
		} else {
			file, err := os.Open(source)
			if err != nil {
				writer.Close()
				fatal(err)
			}

			defer file.Close()
			reader = file
		}

		if _, err := io.Copy(writer, reader); err != nil {
			writer.Close()
			fatal(err)
		}
	}

	if err := closeWriter(writer); err != nil {
		fatal(err)
	}
}
