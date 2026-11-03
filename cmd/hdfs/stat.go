package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/colinmarc/hdfs/v2"
)

func stat(args []string) {
	if len(args) == 0 {
		fatalWithUsage()
	}

	format := "%y"
	if strings.Contains(args[0], "%") {
		format = args[0]
		args = args[1:]
	}

	if len(args) == 0 {
		fatalWithUsage()
	}

	expanded, client, err := getClientAndExpandedPaths(args)
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

		fmt.Println(formatStat(format, p, info))
	}
}

func formatStat(format string, name string, info os.FileInfo) string {
	hfi, ok := info.(*hdfs.FileInfo)
	if !ok {
		hfi = nil
	}

	var buf strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 == len(format) {
			buf.WriteByte(format[i])
			continue
		}

		i++
		switch format[i] {
		case '%':
			buf.WriteByte('%')
		case 'a':
			if hfi != nil {
				status := hfi.Sys().(*hdfs.FileStatus)
				fmt.Fprintf(&buf, "%o", status.GetPermission().GetPerm())
			}
		case 'A':
			mode := info.Mode().String()
			if len(mode) > 1 {
				buf.WriteString(mode[1:])
			}
		case 'b':
			fmt.Fprintf(&buf, "%d", info.Size())
		case 'F':
			buf.WriteString(fileTypeName(hfi, info))
		case 'g':
			if hfi != nil {
				buf.WriteString(hfi.OwnerGroup())
			}
		case 'n':
			buf.WriteString(info.Name())
		case 'o':
			if hfi != nil {
				fmt.Fprintf(&buf, "%d", hfi.BlockSize())
			}
		case 'r':
			if hfi != nil {
				fmt.Fprintf(&buf, "%d", hfi.Replication())
			}
		case 'u':
			if hfi != nil {
				buf.WriteString(hfi.Owner())
			}
		case 'x':
			if hfi != nil {
				buf.WriteString(hfi.AccessTime().UTC().Format("2006-01-02 15:04:05"))
			}
		case 'X':
			if hfi != nil {
				status := hfi.Sys().(*hdfs.FileStatus)
				fmt.Fprintf(&buf, "%d", status.GetAccessTime())
			}
		case 'y':
			buf.WriteString(info.ModTime().UTC().Format("2006-01-02 15:04:05"))
		case 'Y':
			if hfi != nil {
				status := hfi.Sys().(*hdfs.FileStatus)
				fmt.Fprintf(&buf, "%d", status.GetModificationTime())
			}
		default:
			buf.WriteByte('%')
			buf.WriteByte(format[i])
		}
	}

	return buf.String()
}

func fileTypeName(info *hdfs.FileInfo, osInfo os.FileInfo) string {
	if info != nil && info.IsSymlink() {
		return "symbolic link"
	} else if osInfo.IsDir() {
		return "directory"
	}

	return "regular file"
}
