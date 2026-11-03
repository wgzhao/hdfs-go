package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/colinmarc/hdfs/v2"
)

func getfattr(args []string, recursive bool, name string, dumpAll bool, encoding string) {
	if len(args) == 0 {
		fatalWithUsage()
	} else if name == "" && !dumpAll {
		fatal("You must specify either -n or -d.")
	}

	if encoding != "text" && encoding != "hex" && encoding != "base64" {
		fatal("invalid encoding:", encoding)
	}

	expanded, client, err := getClientAndExpandedPaths(args)
	if err != nil {
		fatal(err)
	}

	for _, p := range expanded {
		if recursive {
			err := client.Walk(p, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					status = 1
					return nil
				}

				printXAttrs(client, filePath, name, dumpAll, encoding)
				return nil
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		} else {
			printXAttrs(client, p, name, dumpAll, encoding)
		}
	}
}

func printXAttrs(client *hdfs.Client, p, name string, dumpAll bool, encoding string) {
	var attrs map[string]string
	var err error

	if !dumpAll {
		attrs, err = client.GetXAttrs(p, name)
	} else {
		attrs, err = client.ListXAttrs(p)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		status = 1
		return
	}

	fmt.Printf("# file: %s\n", p)

	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("%s=%s\n", key, encodeXAttr(attrs[key], encoding))
	}
}

func encodeXAttr(value, encoding string) string {
	switch encoding {
	case "hex":
		return "0x" + hex.EncodeToString([]byte(value))
	case "base64":
		return "0s" + base64.StdEncoding.EncodeToString([]byte(value))
	default:
		return strconv.Quote(value)
	}
}

func setfattr(args []string, name, value, removeName string) {
	if len(args) == 0 {
		fatalWithUsage()
	} else if (name == "") == (removeName == "") {
		fatal("Exactly one of -n or -x must be specified.")
	}

	decodedValue := value
	if removeName == "" {
		var err error
		decodedValue, err = decodeXAttrValue(value)
		if err != nil {
			fatal(err)
		}
	}

	expanded, client, err := getClientAndExpandedPaths(args)
	if err != nil {
		fatal(err)
	}

	for _, p := range expanded {
		if removeName != "" {
			err = client.RemoveXAttr(p, removeName)
		} else {
			err = client.SetXAttr(p, name, decodedValue)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}
}

func decodeXAttrValue(value string) (string, error) {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1], nil
	}

	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		decoded, err := hex.DecodeString(value[2:])
		if err != nil {
			return "", err
		}

		return string(decoded), nil
	}

	if strings.HasPrefix(value, "0s") || strings.HasPrefix(value, "0S") {
		decoded, err := base64.StdEncoding.DecodeString(value[2:])
		if err != nil {
			return "", err
		}

		return string(decoded), nil
	}

	return value, nil
}
