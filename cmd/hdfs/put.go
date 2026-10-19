package main

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/colinmarc/hdfs/v2"
)

func put(args []string) {
	if len(args) != 2 {
		fatalWithUsage()
	}

	dests, nn, err := normalizePaths(args[1:])
	if err != nil {
		fatal(err)
	}

	dest := dests[0]
	source, err := filepath.Abs(args[0])
	if err != nil {
		fatal(err)
	}

	client, err := getClient(nn)
	if err != nil {
		fatal(err)
	}

	if filepath.Base(source) == "-" {
		putFromStdin(client, dest)
	} else {
		putFromFile(client, source, dest)
	}
}

func putFromStdin(client *hdfs.Client, dest string) {
	// If the destination exists, regardless of what it is, bail out.
	_, err := client.Stat(dest)
	if err == nil {
		fatal(&os.PathError{"put", dest, os.ErrExist})
	} else if !os.IsNotExist(err) {
		fatal(err)
	}

	mode := 0755 | os.ModeDir
	parentDir := path.Dir(dest)
	if parentDir != "." && parentDir != "/" {
		if err := client.MkdirAll(parentDir, mode); err != nil {
			fatal(err)
		}
	}

	writer, err := client.Create(dest)
	if err != nil {
		fatal(err)
	}

	if _, err := io.Copy(writer, os.Stdin); err != nil {
		writer.Close()
		fatal(err)
	}

	if err := closeWriter(writer); err != nil {
		fatal(err)
	}
}

func putFromFile(client *hdfs.Client, source string, dest string) {
	// If the destination is an existing directory, place it inside. Otherwise,
	// the destination is really the parent directory, and we need to rename the
	// source directory as we copy.
	existing, err := client.Stat(dest)
	if err == nil {
		if existing.IsDir() {
			dest = path.Join(dest, filepath.Base(source))
		} else {
			fatal(&os.PathError{"mkdir", dest, os.ErrExist})
		}
	} else if !os.IsNotExist(err) {
		fatal(err)
	}

	mode := 0755 | os.ModeDir
	err = filepath.Walk(source, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(source, p)
		if err != nil {
			return err
		}

		// rel uses the local filesystem's separator; HDFS paths always use
		// forward slashes.
		fullDest := path.Join(dest, filepath.ToSlash(rel))
		if fi.IsDir() {
			client.Mkdir(fullDest, mode)
		} else {
			writer, err := client.Create(fullDest)
			if err != nil {
				return err
			}

			reader, err := os.Open(p)
			if err != nil {
				writer.Close()
				return err
			}

			defer reader.Close()
			if _, err := io.Copy(writer, reader); err != nil {
				writer.Close()
				return err
			}

			if err := closeWriter(writer); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		fatal(err)
	}
}

// closeWriter retries Close while the namenode reports that the file is still
// being replicated. HDFS only releases the file lease once all blocks have
// been acknowledged, which can take a moment after the data itself is written.
func closeWriter(w *hdfs.FileWriter) error {
	var err error
	for i := 0; i < 300; i++ {
		err = w.Close()
		if !hdfs.IsErrReplicating(err) {
			return err
		}

		time.Sleep(100 * time.Millisecond)
	}

	return err
}
