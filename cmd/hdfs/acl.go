package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/colinmarc/hdfs/v2"
)

func getfacl(args []string, recursive bool) {
	if len(args) == 0 {
		fatalWithUsage()
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

				printAcl(client, filePath)
				return nil
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		} else {
			printAcl(client, p)
		}
	}
}

func printAcl(client *hdfs.Client, p string) {
	aclStatus, err := client.GetAclStatus(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		status = 1
		return
	}

	hasAcl := false
	if info, err := client.Stat(p); err == nil {
		if hfi, ok := info.(*hdfs.FileInfo); ok {
			hasAcl = hfi.HasAcl()
		}
	}

	fmt.Printf("# file: %s\n", p)
	fmt.Printf("# owner: %s\n", aclStatus.Owner)
	fmt.Printf("# group: %s\n", aclStatus.Group)

	for _, entry := range orderedAclEntries(aclStatus, hasAcl) {
		fmt.Println(formatAclEntry(entry))
	}

	fmt.Println()
}

func orderedAclEntries(aclStatus *hdfs.AclStatus, hasAcl bool) []hdfs.AclEntry {
	var accessUser, accessNamedUsers []hdfs.AclEntry
	var accessGroup, accessNamedGroups []hdfs.AclEntry
	var accessMask, accessOther []hdfs.AclEntry
	var defaultUser, defaultNamedUsers []hdfs.AclEntry
	var defaultGroup, defaultNamedGroups []hdfs.AclEntry
	var defaultMask, defaultOther []hdfs.AclEntry
	hasDefault := false
	hasAccessExtended := false

	for _, entry := range aclStatus.Entries {
		if entry.Scope == hdfs.AclDefaultScope {
			hasDefault = true
		} else if entry.Name != "" || entry.Type == hdfs.AclMaskEntry {
			hasAccessExtended = true
		}

		switch entry.Scope {
		case hdfs.AclDefaultScope:
			switch entry.Type {
			case hdfs.AclUserEntry:
				if entry.Name == "" {
					defaultUser = append(defaultUser, entry)
				} else {
					defaultNamedUsers = append(defaultNamedUsers, entry)
				}
			case hdfs.AclGroupEntry:
				if entry.Name == "" {
					defaultGroup = append(defaultGroup, entry)
				} else {
					defaultNamedGroups = append(defaultNamedGroups, entry)
				}
			case hdfs.AclMaskEntry:
				defaultMask = append(defaultMask, entry)
			case hdfs.AclOtherEntry:
				defaultOther = append(defaultOther, entry)
			}
		default:
			switch entry.Type {
			case hdfs.AclUserEntry:
				if entry.Name == "" {
					accessUser = append(accessUser, entry)
				} else {
					accessNamedUsers = append(accessNamedUsers, entry)
				}
			case hdfs.AclGroupEntry:
				if entry.Name == "" {
					accessGroup = append(accessGroup, entry)
				} else {
					accessNamedGroups = append(accessNamedGroups, entry)
				}
			case hdfs.AclMaskEntry:
				accessMask = append(accessMask, entry)
			case hdfs.AclOtherEntry:
				accessOther = append(accessOther, entry)
			}
		}
	}

	permissions := aclStatus.Permissions
	if len(accessUser) == 0 {
		accessUser = append(accessUser, hdfs.AclEntry{
			Scope: hdfs.AclAccessScope, Type: hdfs.AclUserEntry, Permissions: permissions >> 6,
		})
	}
	if len(accessGroup) == 0 {
		accessGroup = append(accessGroup, hdfs.AclEntry{
			Scope: hdfs.AclAccessScope, Type: hdfs.AclGroupEntry, Permissions: (permissions >> 3) & 7,
		})
	}
	if len(accessOther) == 0 {
		accessOther = append(accessOther, hdfs.AclEntry{
			Scope: hdfs.AclAccessScope, Type: hdfs.AclOtherEntry, Permissions: permissions & 7,
		})
	}

	showAccessMask := len(accessMask) > 0 || hasAccessExtended || (hasAcl && !hasDefault)
	if showAccessMask {
		if len(accessGroup) > 0 && accessGroup[0].Permissions != (permissions>>3)&7 {
			showAccessMask = true
		}

		if len(accessMask) == 0 {
			accessMask = append(accessMask, hdfs.AclEntry{
				Scope: hdfs.AclAccessScope, Type: hdfs.AclMaskEntry,
				Permissions: (permissions >> 3) & 7,
			})
		}
	} else {
		accessMask = nil
	}

	if hasDefault {
		if len(defaultUser) == 0 {
			defaultUser = append(defaultUser, hdfs.AclEntry{
				Scope: hdfs.AclDefaultScope, Type: hdfs.AclUserEntry,
			})
		}
		if len(defaultGroup) == 0 {
			defaultGroup = append(defaultGroup, hdfs.AclEntry{
				Scope: hdfs.AclDefaultScope, Type: hdfs.AclGroupEntry,
			})
		}
		if len(defaultOther) == 0 {
			defaultOther = append(defaultOther, hdfs.AclEntry{
				Scope: hdfs.AclDefaultScope, Type: hdfs.AclOtherEntry,
			})
		}
	}

	result := make([]hdfs.AclEntry, 0, len(aclStatus.Entries)+6)
	result = append(result, accessUser...)
	result = append(result, accessNamedUsers...)
	result = append(result, accessGroup...)
	result = append(result, accessNamedGroups...)
	result = append(result, accessMask...)
	result = append(result, accessOther...)
	result = append(result, defaultUser...)
	result = append(result, defaultNamedUsers...)
	result = append(result, defaultGroup...)
	result = append(result, defaultNamedGroups...)
	result = append(result, defaultMask...)
	result = append(result, defaultOther...)

	return result
}

func formatAclEntry(entry hdfs.AclEntry) string {
	var buf strings.Builder
	if entry.Scope == hdfs.AclDefaultScope {
		buf.WriteString("default:")
	}

	switch entry.Type {
	case hdfs.AclGroupEntry:
		buf.WriteString("group")
	case hdfs.AclMaskEntry:
		buf.WriteString("mask")
	case hdfs.AclOtherEntry:
		buf.WriteString("other")
	default:
		buf.WriteString("user")
	}

	buf.WriteByte(':')
	buf.WriteString(entry.Name)
	buf.WriteByte(':')
	buf.WriteString(formatAclPermissions(entry.Permissions))

	return buf.String()
}

func formatAclPermissions(permissions os.FileMode) string {
	var buf [3]byte
	for i := 0; i < 3; i++ {
		buf[i] = '-'
	}

	if permissions&4 != 0 {
		buf[0] = 'r'
	}
	if permissions&2 != 0 {
		buf[1] = 'w'
	}
	if permissions&1 != 0 {
		buf[2] = 'x'
	}

	return string(buf[:])
}

func setfacl(args []string) {
	var recursive, removeAll, removeDefault bool
	var modifySpec, removeSpec, setSpec string
	var paths []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-R":
			recursive = true
		case "-b":
			removeAll = true
		case "-k":
			removeDefault = true
		case "-m":
			i++
			if i == len(args) {
				fatalWithUsage()
			}
			modifySpec = args[i]
		case "-x":
			i++
			if i == len(args) {
				fatalWithUsage()
			}
			removeSpec = args[i]
		case "--set", "-set":
			i++
			if i == len(args) {
				fatalWithUsage()
			}
			setSpec = args[i]
		case "--":
			paths = append(paths, args[i+1:]...)
			i = len(args)
		default:
			if strings.HasPrefix(args[i], "-") {
				fatal("Unknown option:", args[i])
			}
			paths = append(paths, args[i])
		}
	}

	if len(paths) == 0 {
		fatalWithUsage()
	}

	operations := 0
	if modifySpec != "" {
		operations++
	}
	if removeSpec != "" {
		operations++
	}
	if setSpec != "" {
		operations++
	}
	if operations == 0 {
		if !removeAll && !removeDefault {
			fatalWithUsage()
		}
	} else if operations > 1 {
		fatal("Only one of -m, -x, or --set may be specified.")
	}

	var modifyEntries, removeEntries, setEntries []hdfs.AclEntry
	var err error
	if modifySpec != "" {
		modifyEntries, err = parseAclSpec(modifySpec, false)
		if err != nil {
			fatal(err)
		}
	}
	if removeSpec != "" {
		removeEntries, err = parseAclSpec(removeSpec, true)
		if err != nil {
			fatal(err)
		}
	}
	if setSpec != "" {
		setEntries, err = parseAclSpec(setSpec, false)
		if err != nil {
			fatal(err)
		}
		if !hasBaseAclEntries(setEntries) {
			fatal("Invalid ACL: the user, group and other entries are required.")
		}
	}

	apply := func(client *hdfs.Client, p string) {
		if removeAll {
			if err := client.RemoveAcl(p); err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
				return
			}
		}
		if removeDefault {
			if err := client.RemoveDefaultAcl(p); err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
				return
			}
		}
		if setSpec != "" {
			if err := client.SetAcl(p, setEntries); err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		} else if modifySpec != "" {
			if err := client.ModifyAclEntries(p, modifyEntries); err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		} else if removeSpec != "" {
			if err := client.RemoveAclEntries(p, removeEntries); err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		}
	}

	expanded, client, err := getClientAndExpandedPaths(paths)
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

				apply(client, filePath)
				return nil
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				status = 1
			}
		} else {
			apply(client, p)
		}
	}
}

func parseAclSpec(spec string, allowMissingPermissions bool) ([]hdfs.AclEntry, error) {
	if spec == "" {
		return nil, errors.New("empty ACL spec")
	}

	parts := strings.Split(spec, ",")
	entries := make([]hdfs.AclEntry, 0, len(parts))
	for _, part := range parts {
		entry, err := parseAclEntry(part, allowMissingPermissions)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func parseAclEntry(spec string, allowMissingPermissions bool) (hdfs.AclEntry, error) {
	entry := hdfs.AclEntry{Scope: hdfs.AclAccessScope}
	spec = strings.TrimSpace(spec)

	if strings.HasPrefix(spec, "default:") {
		entry.Scope = hdfs.AclDefaultScope
		spec = strings.TrimPrefix(spec, "default:")
	} else if strings.HasPrefix(spec, "d:") {
		entry.Scope = hdfs.AclDefaultScope
		spec = strings.TrimPrefix(spec, "d:")
	}

	fields := strings.SplitN(spec, ":", 3)
	if len(fields) < 2 {
		return entry, fmt.Errorf("invalid ACL entry: %s", spec)
	}

	switch fields[0] {
	case "user", "u":
		entry.Type = hdfs.AclUserEntry
	case "group", "g":
		entry.Type = hdfs.AclGroupEntry
	case "mask", "m":
		entry.Type = hdfs.AclMaskEntry
	case "other", "o":
		entry.Type = hdfs.AclOtherEntry
	default:
		return entry, fmt.Errorf("invalid ACL entry type: %s", fields[0])
	}

	entry.Name = fields[1]
	if (entry.Type == hdfs.AclMaskEntry || entry.Type == hdfs.AclOtherEntry) && entry.Name != "" {
		return entry, fmt.Errorf("invalid ACL entry: %s", spec)
	}

	if len(fields) == 3 {
		permissions, err := parseAclPermissions(fields[2])
		if err != nil {
			return entry, err
		}
		entry.Permissions = permissions
	} else if !allowMissingPermissions {
		return entry, fmt.Errorf("invalid ACL entry: %s", spec)
	}

	return entry, nil
}

func parseAclPermissions(permissions string) (os.FileMode, error) {
	if len(permissions) != 3 {
		return 0, fmt.Errorf("invalid ACL permissions: %s", permissions)
	}

	var result os.FileMode
	for i := 0; i < len(permissions); i++ {
		switch permissions[i] {
		case '-':
		case 'r':
			result |= 4
		case 'w':
			result |= 2
		case 'x':
			result |= 1
		default:
			return 0, fmt.Errorf("invalid ACL permissions: %s", permissions)
		}
	}

	return result, nil
}

func hasBaseAclEntries(entries []hdfs.AclEntry) bool {
	var hasUser, hasGroup, hasOther bool
	for _, entry := range entries {
		if entry.Scope != hdfs.AclAccessScope || entry.Name != "" {
			continue
		}

		switch entry.Type {
		case hdfs.AclUserEntry:
			hasUser = true
		case hdfs.AclGroupEntry:
			hasGroup = true
		case hdfs.AclOtherEntry:
			hasOther = true
		}
	}

	return hasUser && hasGroup && hasOther
}
