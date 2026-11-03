package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/colinmarc/hdfs/v2"
)

type findCondition struct {
	pattern    string
	ignoreCase bool
}

type findExpression struct {
	conditions []findCondition
	prints     []bool // true for -print0
}

func find(args []string) {
	var paths []string
	i := 0
	for i < len(args) && !strings.HasPrefix(args[i], "-") {
		paths = append(paths, args[i])
		i++
	}

	expr, err := parseFindExpression(args[i:])
	if err != nil {
		fatal(err)
	}

	var expanded []string
	var client *hdfs.Client
	if len(paths) == 0 {
		client, err = getClient("")
		if err != nil {
			fatal(err)
		}

		expanded = []string{userDir(client)}
	} else {
		expanded, client, err = getClientAndExpandedPaths(paths)
		if err != nil {
			fatal(err)
		}
	}

	for _, p := range expanded {
		err := client.Walk(p, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
				return nil
			}

			for _, condition := range expr.conditions {
				if !condition.matches(info.Name()) {
					return nil
				}
			}

			for _, print0 := range expr.prints {
				if print0 {
					fmt.Printf("%s\x00", filePath)
				} else {
					fmt.Println(filePath)
				}
			}

			return nil
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			status = 1
		}
	}
}

func parseFindExpression(args []string) (*findExpression, error) {
	expr := &findExpression{}
	lastWasOperator := true

	for i := 0; i < len(args); {
		switch args[i] {
		case "-a", "-and":
			if lastWasOperator || i+1 == len(args) {
				return nil, fmt.Errorf("find: Unexpected argument: %s", args[i])
			}
			lastWasOperator = true
			i++
		case "-name", "-iname":
			if i+1 == len(args) {
				return nil, fmt.Errorf("find: Missing argument to %s", args[i])
			}
			expr.conditions = append(expr.conditions, findCondition{
				pattern:    args[i+1],
				ignoreCase: args[i] == "-iname",
			})
			lastWasOperator = false
			i += 2
		case "-print":
			expr.prints = append(expr.prints, false)
			lastWasOperator = false
			i++
		case "-print0":
			expr.prints = append(expr.prints, true)
			lastWasOperator = false
			i++
		default:
			return nil, fmt.Errorf("find: Unexpected argument: %s", args[i])
		}
	}

	if lastWasOperator && len(args) > 0 {
		return nil, fmt.Errorf("find: Unexpected end of expression")
	}

	if len(expr.prints) == 0 {
		expr.prints = append(expr.prints, false)
	}

	return expr, nil
}

func (c findCondition) matches(name string) bool {
	pattern := c.pattern
	if c.ignoreCase {
		name = strings.ToLower(name)
		pattern = strings.ToLower(pattern)
	}

	matched, err := path.Match(pattern, name)
	if err != nil {
		return false
	}

	return matched
}
