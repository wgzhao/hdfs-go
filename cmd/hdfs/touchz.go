package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

func touchz(paths []string) {
	if len(paths) == 0 {
		fatalWithUsage()
	}

	paths, nn, err := normalizePaths(paths)
	if err != nil {
		fatal(err)
	}

	client, err := getClient(nn)
	if err != nil {
		fatal(err)
	}

	now := time.Now()
	for _, p := range paths {
		info, err := client.Stat(p)
		if err == nil {
			if info.IsDir() {
				fmt.Fprintln(os.Stderr, &os.PathError{"touchz", p, errors.New("is a directory")})
				status = 1
				continue
			} else if info.Size() != 0 {
				fmt.Fprintln(os.Stderr, &os.PathError{"touchz", p, errors.New("not a zero-length file")})
				status = 1
				continue
			}

			err = client.Chtimes(p, now, now)
		} else if os.IsNotExist(err) {
			err = client.CreateEmptyFile(p)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}
}
