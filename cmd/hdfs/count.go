package main

import (
	"fmt"
	"os"

	"github.com/colinmarc/hdfs/v2"
)

func count(paths []string, showQuota, humanReadable, verbose, showUsage, showEC bool) {
	if len(paths) == 0 {
		fatalWithUsage()
	}

	expanded, client, err := getClientAndExpandedPaths(paths)
	if err != nil {
		fatal(err)
	}

	if verbose {
		printCountHeader(showQuota, showUsage, showEC)
	}

	for _, p := range expanded {
		summary, err := client.GetContentSummary(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
			continue
		}

		nameQuota, remainingNameQuota := quotaValues(summary)
		spaceQuota, remainingSpaceQuota := spaceQuotaValues(summary)

		if humanReadable {
			if nameQuota != "none" {
				nameQuota = formatCountSize(parseCountSize(nameQuota))
			}
			if remainingNameQuota != "inf" {
				remainingNameQuota = formatCountSize(parseCountSize(remainingNameQuota))
			}
			if spaceQuota != "none" {
				spaceQuota = formatCountSize(parseCountSize(spaceQuota))
			}
			if remainingSpaceQuota != "inf" {
				remainingSpaceQuota = formatCountSize(parseCountSize(remainingSpaceQuota))
			}
		}

		contentSize := formatCountNumber(summary.Size(), humanReadable)

		switch {
		case showUsage:
			if showEC {
				fmt.Printf("%12s %15s %15s %15s %s %s\n",
					nameQuota, remainingNameQuota, spaceQuota, remainingSpaceQuota, "EC:", p)
			} else {
				fmt.Printf("%12s %15s %15s %15s %s\n",
					nameQuota, remainingNameQuota, spaceQuota, remainingSpaceQuota, p)
			}
		case showQuota:
			if showEC {
				fmt.Printf("%12s %15s %15s %15s %12d %12d %18s %s %s\n",
					nameQuota, remainingNameQuota, spaceQuota, remainingSpaceQuota,
					summary.DirectoryCount(), summary.FileCount(), contentSize, "EC:", p)
			} else {
				fmt.Printf("%12s %15s %15s %15s %12d %12d %18s %s\n",
					nameQuota, remainingNameQuota, spaceQuota, remainingSpaceQuota,
					summary.DirectoryCount(), summary.FileCount(), contentSize, p)
			}
		default:
			if showEC {
				fmt.Printf("%12d %12d %18s %s %s\n",
					summary.DirectoryCount(), summary.FileCount(), contentSize, "EC:", p)
			} else {
				fmt.Printf("%12d %12d %18s %s\n",
					summary.DirectoryCount(), summary.FileCount(), contentSize, p)
			}
		}
	}
}

func printCountHeader(showQuota, showUsage, showEC bool) {
	if showUsage {
		if showEC {
			fmt.Printf("%12s %15s %15s %15s %s %s\n",
				"QUOTA", "REM_QUOTA", "SPACE_QUOTA", "REM_SPACE_QUOTA", "ERASURECODING_POLICY", "PATHNAME")
		} else {
			fmt.Printf("%12s %15s %15s %15s %s\n",
				"QUOTA", "REM_QUOTA", "SPACE_QUOTA", "REM_SPACE_QUOTA", "PATHNAME")
		}
	} else if showQuota {
		if showEC {
			fmt.Printf("%12s %15s %15s %15s %12s %12s %18s %s %s\n",
				"QUOTA", "REM_QUOTA", "SPACE_QUOTA", "REM_SPACE_QUOTA",
				"DIR_COUNT", "FILE_COUNT", "CONTENT_SIZE", "ERASURECODING_POLICY", "PATHNAME")
		} else {
			fmt.Printf("%12s %15s %15s %15s %12s %12s %18s %s\n",
				"QUOTA", "REM_QUOTA", "SPACE_QUOTA", "REM_SPACE_QUOTA",
				"DIR_COUNT", "FILE_COUNT", "CONTENT_SIZE", "PATHNAME")
		}
	} else if showEC {
		fmt.Printf("%12s %12s %18s %s %s\n",
			"DIR_COUNT", "FILE_COUNT", "CONTENT_SIZE", "ERASURECODING_POLICY", "PATHNAME")
	} else {
		fmt.Printf("%12s %12s %18s %s\n",
			"DIR_COUNT", "FILE_COUNT", "CONTENT_SIZE", "PATHNAME")
	}
}

func quotaValues(summary *hdfs.ContentSummary) (string, string) {
	quota := summary.NameQuota()
	if quota < 0 {
		return "none", "inf"
	}

	remaining := quota - (summary.DirectoryCount() + summary.FileCount())
	return fmt.Sprintf("%d", quota), fmt.Sprintf("%d", remaining)
}

func spaceQuotaValues(summary *hdfs.ContentSummary) (string, string) {
	quota := summary.SpaceQuota()
	if quota < 0 {
		return "none", "inf"
	}

	remaining := quota - summary.SizeAfterReplication()
	return fmt.Sprintf("%d", quota), fmt.Sprintf("%d", remaining)
}

func formatCountNumber(value int64, humanReadable bool) string {
	if humanReadable {
		return formatCountSize(value)
	}

	return fmt.Sprintf("%d", value)
}

func formatCountSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d", size)
	}

	units := []string{"K", "M", "G", "T", "P"}
	value := float64(size)
	for _, unit := range units {
		value /= 1024
		if value < 1024 || unit == "P" {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%d", size)
}

func parseCountSize(value string) int64 {
	var size int64
	fmt.Sscanf(value, "%d", &size)
	return size
}
