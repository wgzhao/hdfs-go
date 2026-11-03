package main

import (
	"fmt"
	"os"

	"github.com/colinmarc/hdfs/v2"
)

func chgrp(args []string, recursive bool) {
	if len(args) < 2 {
		fatalWithUsage()
	}

	group := args[0]
	expanded, client, err := getClientAndExpandedPaths(args[1:])
	if err != nil {
		fatal(err)
	}

	visit := func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
			return err
		}

		// Changing a group to its current value is a no-op, and the
		// namenode rejects it if the user isn't a member of that group.
		if hfi, ok := fi.(*hdfs.FileInfo); ok && hfi.OwnerGroup() == group {
			return nil
		}

		err = client.Chown(p, "", group)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
			return err
		}

		return nil
	}

	for _, p := range expanded {
		if recursive {
			err = client.Walk(p, visit)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		} else {
			info, err := client.Stat(p)
			if err != nil {
				fatal(err)
			}

			visit(p, info, nil)
		}
	}
}
