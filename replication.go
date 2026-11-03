package hdfs

import (
	"errors"
	"os"

	hdfs "github.com/colinmarc/hdfs/v2/internal/protocol/hadoop_hdfs"
	"google.golang.org/protobuf/proto"
)

// SetReplication sets the block replication factor for a file. HDFS will
// asynchronously create or remove replicas to match the new factor.
func (c *Client) SetReplication(name string, replication uint16) error {
	req := &hdfs.SetReplicationRequestProto{
		Src:         proto.String(name),
		Replication: proto.Uint32(uint32(replication)),
	}
	resp := &hdfs.SetReplicationResponseProto{}

	err := c.namenode.Execute("setReplication", req, resp)
	if err != nil {
		return &os.PathError{"set replication", name, interpretException(err)}
	} else if !resp.GetResult() {
		return &os.PathError{
			"set replication",
			name,
			errors.New("the namenode did not set the replication factor"),
		}
	}

	return nil
}

// BlockReplicaCounts returns the number of live replicas for each block of a
// file. It is useful for waiting until a replication factor change has been
// carried out by the datanodes.
func (c *Client) BlockReplicaCounts(name string) ([]int, error) {
	fi, err := c.getFileInfo(name)
	if err != nil {
		return nil, &os.PathError{"get block locations", name, interpretException(err)}
	} else if fi.IsDir() {
		return nil, &os.PathError{
			"get block locations",
			name,
			errors.New("the file is a directory"),
		}
	}

	req := &hdfs.GetBlockLocationsRequestProto{
		Src:    proto.String(name),
		Offset: proto.Uint64(0),
		Length: proto.Uint64(uint64(fi.Size())),
	}
	resp := &hdfs.GetBlockLocationsResponseProto{}

	err = c.namenode.Execute("getBlockLocations", req, resp)
	if err != nil {
		return nil, &os.PathError{"get block locations", name, interpretException(err)}
	}

	blocks := resp.GetLocations().GetBlocks()
	counts := make([]int, len(blocks))
	for i, block := range blocks {
		counts[i] = len(block.GetLocs())
	}

	return counts, nil
}
