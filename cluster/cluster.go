package cluster

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/serialx/hashring"
)

// IndexDistribution represents how an index is distributed
type IndexDistribution struct {
	IndexName string
	Replicas  int
	Shards    int
}

// Cluster manages Raft consensus and sharding
type Cluster struct {
	raft        *raft.Raft
	fsm         *FSM
	ring        *hashring.HashRing
	nodeID      string
	bindAddr    string
	indexes     map[string]IndexDistribution
	isLeader    bool
}

// NewCluster initializes a new cluster.
func NewCluster(cfg Config) (*Cluster, error) {
	// Setup Raft
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.NodeID)

	store := raft.NewInmemStore()
	snapshots := raft.NewInmemSnapshotStore()

	transport, err := raft.NewTCPTransport(cfg.BindAddr, nil, 3, raft.DefaultTimeout, log.Writer())
	if err != nil {
		return nil, err
	}

	ra, err := raft.NewRaft(raftConfig, nil, store, store, snapshots, transport)
	if err != nil {
		return nil, err
	}

	// Setup Consistent Hashing
	ring := hashring.New([]string{})
	for i := 0; i < cfg.NumReplicas; i++ {
		for j := 0; j < cfg.NumShards; j++ {
			shardID := cfg.NodeID + ":" + strconv.Itoa(i) + ":" + strconv.Itoa(j)
			ring = ring.AddNode(shardID)
		}
	}

	// Setup Memberlist
	memberConfig := memberlist.DefaultLocalConfig()
	memberConfig.BindPort = cfg.RaftPort
	memberConfig.Name = cfg.NodeID
	
	mlist, err := memberlist.Create(memberConfig)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		raft:        ra,
		ring:        ring,
		memberList:  mlist,
		numReplicas: cfg.NumReplicas,
		numShards:   cfg.NumShards,
	}, nil
}

// AddMember adds a new member to the cluster.
func (c *Cluster) AddMember(nodeID string) {
	for i := 0; i < c.numReplicas; i++ {
		for j := 0; j < c.numShards; j++ {
			shardID := nodeID + ":" + strconv.Itoa(i) + ":" + strconv.Itoa(j)
			c.ring = c.ring.AddNode(shardID)
		}
	}
}

// RemoveMember removes a member from the cluster.
func (c *Cluster) RemoveMember(nodeID string) {
	for i := 0; i < c.numReplicas; i++ {
		for j := 0; j < c.numShards; j++ {
			shardID := nodeID + ":" + strconv.Itoa(i) + ":" + strconv.Itoa(j)
			c.ring = c.ring.RemoveNode(shardID)
		}
	}
}

// GetShard gets the shard for a given key.
func (c *Cluster) GetShard(key string) string {
	shard, _ := c.ring.GetNode(key)
	return shard
}

