package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/colinmarc/hdfs/v2"
)

func cp(args []string, force, preserve bool) {
	if len(args) < 2 {
		fatalWithUsage()
	}

	sources, client, err := getClientAndExpandedPaths(args[:len(args)-1])
	if err != nil {
		fatal(err)
	}

	dests, _, err := normalizePaths(args[len(args)-1:])
	if err != nil {
		fatal(err)
	}

	dest := dests[0]
	destInfo, statErr := client.Stat(dest)
	destExists := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		fatal(statErr)
	}

	if len(sources) > 1 && (!destExists || !destInfo.IsDir()) {
		fatal(&os.PathError{"cp", dest, errors.New("destination must be a directory")})
	}

	for _, source := range sources {
		target := dest
		if destExists && destInfo.IsDir() {
			target = path.Join(dest, path.Base(source))
		}

		if err := copyPath(client, source, target, force, preserve); err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}
}

func copyPath(client *hdfs.Client, source, dest string, force, preserve bool) error {
	if source == dest {
		return &os.PathError{"cp", source, errors.New("source and destination are the same")}
	}

	info, err := client.Stat(source)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(client, source, dest, force, preserve)
	}

	return copyFile(client, source, dest, force, preserve)
}

func copyFile(client *hdfs.Client, source, dest string, force, preserve bool) error {
	sourceInfo, err := client.Stat(source)
	if err != nil {
		return err
	}

	if destInfo, err := client.Stat(dest); err == nil {
		if destInfo.IsDir() {
			return &os.PathError{"cp", dest, errors.New("is a directory")}
		} else if !force {
			return &os.PathError{"cp", dest, os.ErrExist}
		}

		if err := client.Remove(dest); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	var writer *hdfs.FileWriter
	if preserve {
		if hfi, ok := sourceInfo.(*hdfs.FileInfo); ok {
			writer, err = client.CreateFile(dest, hfi.Replication(), hfi.BlockSize(), hdfsPermissions(sourceInfo.Mode()))
		} else {
			writer, err = client.Create(dest)
		}
	} else {
		writer, err = client.Create(dest)
	}
	if err != nil {
		return err
	}

	reader, err := client.Open(source)
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

	if preserve {
		if hfi, ok := sourceInfo.(*hdfs.FileInfo); ok {
			err = client.Chtimes(dest, hfi.AccessTime(), hfi.ModTime())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func copyDir(client *hdfs.Client, source, dest string, force, preserve bool) error {
	if dest == source || strings.HasPrefix(dest, source+"/") {
		return &os.PathError{"cp", dest, errors.New("cannot copy a directory into itself")}
	}

	sourceInfo, err := client.Stat(source)
	if err != nil {
		return err
	}

	if destInfo, err := client.Stat(dest); err == nil {
		if !destInfo.IsDir() {
			return &os.PathError{"cp", dest, errors.New("not a directory")}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	err = client.MkdirAll(dest, hdfsPermissions(sourceInfo.Mode()))
	if err != nil {
		return err
	}

	err = client.Walk(source, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if p == source {
			return nil
		}

		relative := strings.TrimPrefix(p, source)
		relative = strings.TrimPrefix(relative, "/")
		target := path.Join(dest, relative)

		if info.IsDir() {
			return client.MkdirAll(target, hdfsPermissions(info.Mode()))
		}

		return copyFile(client, p, target, force, preserve)
	})
	if err != nil {
		return err
	}

	if preserve {
		if hfi, ok := sourceInfo.(*hdfs.FileInfo); ok {
			return client.Chtimes(dest, hfi.AccessTime(), hfi.ModTime())
		}
	}

	return nil
}

// hdfsPermissions converts an os.FileMode to the low permission bits used by
// HDFS, including the setuid, setgid, and sticky bits.
func hdfsPermissions(mode os.FileMode) os.FileMode {
	permissions := mode.Perm()
	if mode&os.ModeSetuid != 0 {
		permissions |= 04000
	}
	if mode&os.ModeSetgid != 0 {
		permissions |= 02000
	}
	if mode&os.ModeSticky != 0 {
		permissions |= 01000
	}

	return permissions
}
