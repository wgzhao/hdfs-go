package main

import (
	"os"
)

func moveFromLocal(args []string) {
	if len(args) < 2 {
		fatalWithUsage()
	}

	put(args)

	for _, source := range args[:len(args)-1] {
		if source == "-" {
			continue
		}

		if err := os.RemoveAll(source); err != nil {
			fatal(err)
		}
	}
}

func moveToLocal(args []string) {
	fatal("moveToLocal: Option '-moveToLocal' is not implemented yet.")
}
