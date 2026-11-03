package hdfs

import (
	"errors"
	"os"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"google.golang.org/protobuf/proto"
)

// AclEntryScope is the scope of an ACL entry: access entries apply to the
// file or directory itself, and default entries are inherited by children of
// a directory.
type AclEntryScope int

const (
	AclAccessScope AclEntryScope = iota
	AclDefaultScope
)

// AclEntryType is the type of an ACL entry.
type AclEntryType int

const (
	AclUserEntry AclEntryType = iota
	AclGroupEntry
	AclMaskEntry
	AclOtherEntry
)

// AclEntry represents a single access control list entry.
type AclEntry struct {
	Scope       AclEntryScope
	Type        AclEntryType
	Name        string
	Permissions os.FileMode
}

// AclStatus represents the ACL and permission information for a path.
type AclStatus struct {
	Owner       string
	Group       string
	Sticky      bool
	Entries     []AclEntry
	Permissions os.FileMode
}

// GetAclStatus returns the ACL status for the named path.
func (c *Client) GetAclStatus(name string) (*AclStatus, error) {
	req := &hdfs.GetAclStatusRequestProto{Src: proto.String(name)}
	resp := &hdfs.GetAclStatusResponseProto{}

	err := c.namenode.Execute("getAclStatus", req, resp)
	if err != nil {
		return nil, &os.PathError{"get acl status", name, interpretException(err)}
	}

	result := resp.GetResult()
	if result == nil {
		return nil, &os.PathError{"get acl status", name, errors.New("unexpected empty response")}
	}

	status := &AclStatus{
		Owner:       result.GetOwner(),
		Group:       result.GetGroup(),
		Sticky:      result.GetSticky(),
		Permissions: os.FileMode(result.GetPermission().GetPerm() & 0777),
	}

	for _, entry := range result.GetEntries() {
		status.Entries = append(status.Entries, aclEntryFromProto(entry))
	}

	return status, nil
}

// SetAcl replaces the ACL of the named path with the given entries.
func (c *Client) SetAcl(name string, entries []AclEntry) error {
	req := &hdfs.SetAclRequestProto{
		Src:     proto.String(name),
		AclSpec: aclEntriesToProto(entries),
	}
	return c.executeAcl("setAcl", "set acl", name, req, &hdfs.SetAclResponseProto{})
}

// ModifyAclEntries adds or updates the given ACL entries on the named path.
func (c *Client) ModifyAclEntries(name string, entries []AclEntry) error {
	req := &hdfs.ModifyAclEntriesRequestProto{
		Src:     proto.String(name),
		AclSpec: aclEntriesToProto(entries),
	}
	return c.executeAcl("modifyAclEntries", "modify acl entries", name, req, &hdfs.ModifyAclEntriesResponseProto{})
}

// RemoveAclEntries removes the given ACL entries from the named path.
func (c *Client) RemoveAclEntries(name string, entries []AclEntry) error {
	req := &hdfs.RemoveAclEntriesRequestProto{
		Src:     proto.String(name),
		AclSpec: aclEntriesToProto(entries),
	}
	return c.executeAcl("removeAclEntries", "remove acl entries", name, req, &hdfs.RemoveAclEntriesResponseProto{})
}

// RemoveDefaultAcl removes the default ACL from a directory.
func (c *Client) RemoveDefaultAcl(name string) error {
	req := &hdfs.RemoveDefaultAclRequestProto{Src: proto.String(name)}
	return c.executeAcl("removeDefaultAcl", "remove default acl", name, req, &hdfs.RemoveDefaultAclResponseProto{})
}

// RemoveAcl removes all extended ACL entries from the named path, leaving
// only the base user, group, and other entries.
func (c *Client) RemoveAcl(name string) error {
	req := &hdfs.RemoveAclRequestProto{Src: proto.String(name)}
	return c.executeAcl("removeAcl", "remove acl", name, req, &hdfs.RemoveAclResponseProto{})
}

func (c *Client) executeAcl(method, op, name string, req, resp proto.Message) error {
	err := c.namenode.Execute(method, req, resp)
	if err != nil {
		return &os.PathError{op, name, interpretException(err)}
	}

	return nil
}

func aclEntryFromProto(entry *hdfs.AclEntryProto) AclEntry {
	result := AclEntry{
		Name:        entry.GetName(),
		Permissions: os.FileMode(entry.GetPermissions() & 7),
	}

	if entry.GetScope() == hdfs.AclEntryProto_DEFAULT {
		result.Scope = AclDefaultScope
	} else {
		result.Scope = AclAccessScope
	}

	switch entry.GetType() {
	case hdfs.AclEntryProto_GROUP:
		result.Type = AclGroupEntry
	case hdfs.AclEntryProto_MASK:
		result.Type = AclMaskEntry
	case hdfs.AclEntryProto_OTHER:
		result.Type = AclOtherEntry
	default:
		result.Type = AclUserEntry
	}

	return result
}

func aclEntriesToProto(entries []AclEntry) []*hdfs.AclEntryProto {
	result := make([]*hdfs.AclEntryProto, 0, len(entries))
	for _, entry := range entries {
		protoEntry := &hdfs.AclEntryProto{
			Type:        aclEntryTypeToProto(entry.Type).Enum(),
			Scope:       aclEntryScopeToProto(entry.Scope).Enum(),
			Permissions: hdfs.AclEntryProto_FsActionProto(entry.Permissions & 7).Enum(),
		}
		if entry.Name != "" {
			protoEntry.Name = proto.String(entry.Name)
		}

		result = append(result, protoEntry)
	}

	return result
}

func aclEntryTypeToProto(entryType AclEntryType) hdfs.AclEntryProto_AclEntryTypeProto {
	switch entryType {
	case AclGroupEntry:
		return hdfs.AclEntryProto_GROUP
	case AclMaskEntry:
		return hdfs.AclEntryProto_MASK
	case AclOtherEntry:
		return hdfs.AclEntryProto_OTHER
	default:
		return hdfs.AclEntryProto_USER
	}
}

func aclEntryScopeToProto(scope AclEntryScope) hdfs.AclEntryProto_AclEntryScopeProto {
	if scope == AclDefaultScope {
		return hdfs.AclEntryProto_DEFAULT
	}

	return hdfs.AclEntryProto_ACCESS
}
