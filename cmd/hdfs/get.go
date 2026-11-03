package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func get(args []string) {
	if len(args) == 0 {
		fatalWithUsage()
	}

	var sources []string
	var dest string
	if len(args) == 1 {
		cwd, err := os.Getwd()
		if err != nil {
			fatal(err)
		}

		sources = args
		dest = cwd
	} else {
		sources = args[:len(args)-1]
		dest = args[len(args)-1]
	}

	normalized, nn, err := normalizePaths(sources)
	if err != nil {
		fatal(err)
	}

	client, err := getClient(nn)
	if err != nil {
		fatal(err)
	}

	destInfo, destErr := os.Stat(dest)
	destIsDir := destErr == nil && destInfo.IsDir()
	if len(normalized) > 1 && !destIsDir {
		fatal(&os.PathError{"get", dest, errors.New("destination must be a directory")})
	}

	for _, source := range normalized {
		target := dest
		if destIsDir {
			target = filepath.Join(dest, path.Base(source))
		}

		err = client.Walk(source, func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relative := strings.TrimPrefix(p, source)
			relative = strings.TrimPrefix(relative, "/")
			fullDest := filepath.Join(target, relative)

			if fi.IsDir() {
				if info, err := os.Stat(fullDest); err == nil {
					if info.IsDir() {
						return nil
					}

					return &os.PathError{"mkdir", fullDest, os.ErrExist}
				}

				return os.Mkdir(fullDest, 0755)
			}

			return client.CopyToLocal(p, fullDest)
		})

		if err != nil {
			fatal(err)
		}
	}
}

func getmerge(args []string, addNewlines bool) {
	if len(args) != 2 {
		fatalWithUsage()
	}

	dest := args[1]
	sources, nn, err := normalizePaths(args[0:1])
	if err != nil {
		fatal(err)
	}

	client, err := getClient(nn)
	if err != nil {
		fatal(err)
	}

	local, err := os.Create(dest)
	if err != nil {
		fatal(err)
	}

	source := sources[0]
	children, err := client.ReadDir(source)
	if err != nil {
		fatal(err)
	}

	readers := make([]io.Reader, 0, len(children))
	for _, child := range children {
		if child.IsDir() {
			continue
		}

		childPath := path.Join(source, child.Name())
		file, err := client.Open(childPath)
		if err != nil {
			fatal(err)
		}

		readers = append(readers, file)
		if addNewlines {
			readers = append(readers, bytes.NewBufferString("\n"))
		}
	}

	_, err = io.Copy(local, io.MultiReader(readers...))
	if err != nil {
		fatal(err)
	}
}
