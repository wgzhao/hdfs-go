package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/colinmarc/hdfs/v2"
)

func fatalSnapshotError(err error) {
	if remoteErr, ok := err.(hdfs.Error); ok {
		message := remoteErr.Message()
		if i := strings.IndexByte(message, '\n'); i >= 0 {
			message = message[:i]
		}

		if message != "" {
			fatal(fmt.Sprintf("%s: %s", remoteErr.Method(), message))
		}
	}

	fatal(err)
}

func createSnapshot(args []string) {
	if len(args) < 1 || len(args) > 2 {
		fatalWithUsage()
	}

	dirs, client, err := getClientAndExpandedPaths(args[0:1])
	if err != nil {
		fatal(err)
	}

	name := ""
	if len(args) == 2 {
		name = args[1]
	} else {
		name = fmt.Sprintf("s%d", time.Now().UnixNano()/int64(time.Millisecond))
	}

	snapshotPath, err := client.CreateSnapshot(dirs[0], name)
	if err != nil {
		fatalSnapshotError(err)
	}

	if snapshotPath != "" {
		fmt.Println(snapshotPath)
	}
}

func deleteSnapshot(args []string) {
	if len(args) != 2 {
		fatalWithUsage()
	}

	dirs, client, err := getClientAndExpandedPaths(args[0:1])
	if err != nil {
		fatal(err)
	}

	if err := client.DeleteSnapshot(dirs[0], args[1]); err != nil {
		fatalSnapshotError(err)
	}
}

func renameSnapshot(args []string) {
	if len(args) != 3 {
		fatalWithUsage()
	}

	dirs, client, err := getClientAndExpandedPaths(args[0:1])
	if err != nil {
		fatal(err)
	}

	if err := client.RenameSnapshot(dirs[0], args[1], args[2]); err != nil {
		fatalSnapshotError(err)
	}
}
