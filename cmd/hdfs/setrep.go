package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/colinmarc/hdfs/v2"
)

func setrep(args []string, recursive, wait bool) {
	if len(args) < 2 {
		fatalWithUsage()
	}

	replication, err := strconv.ParseUint(args[0], 10, 16)
	if err != nil {
		fatal("invalid replication factor:", args[0])
	}

	expanded, client, err := getClientAndExpandedPaths(args[1:])
	if err != nil {
		fatal(err)
	}

	for _, p := range expanded {
		info, err := client.Stat(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
			continue
		}

		if info.IsDir() {
			err = client.Walk(p, func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				} else if fi.IsDir() {
					return nil
				}

				return setrepFile(client, path, uint16(replication), wait)
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		} else {
			if err := setrepFile(client, p, uint16(replication), wait); err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		}
	}
}

func setrepFile(client *hdfs.Client, path string, replication uint16, wait bool) error {
	info, err := client.Stat(path)
	if err != nil {
		return err
	}

	current := 0
	if hfi, ok := info.(*hdfs.FileInfo); ok {
		current = hfi.Replication()
	}

	if err := client.SetReplication(path, replication); err != nil {
		return err
	}

	fmt.Printf("Replication %d set: %s\n", replication, path)

	if !wait {
		return nil
	}

	fmt.Printf("Waiting for %s ...\n", path)
	if int(replication) < current {
		fmt.Println("WARNING: the waiting time may be long for DECREASING the number of replications.")
	}

	for {
		counts, err := client.BlockReplicaCounts(path)
		if err != nil {
			return err
		}

		done := true
		for _, count := range counts {
			if int(replication) >= current && count < int(replication) {
				done = false
				break
			}
			if int(replication) < current && count > int(replication) {
				done = false
				break
			}
		}

		if done {
			fmt.Println(" done")
			return nil
		}

		fmt.Print(".")
		time.Sleep(time.Second)
	}
}
