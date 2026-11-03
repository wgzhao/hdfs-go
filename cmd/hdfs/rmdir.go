package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func rmdir(args []string, ignoreNonEmpty bool) {
	if len(args) == 0 {
		fatalWithUsage()
	}

	expanded, client, err := getClientAndExpandedPaths(args)
	if err != nil {
		fatal(err)
	}

	for _, p := range expanded {
		err = client.Remove(p)
		if err != nil {
			if ignoreNonEmpty && errors.Is(err, syscall.ENOTEMPTY) {
				continue
			}

			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}
}
