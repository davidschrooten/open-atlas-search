package cluster

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/davidschrooten/open-atlas-search/config"
	"github.com/stretchr/testify/assert"
)

func newTestRaftConfig(t *testing.T, nodeID, bindAddr string) *config.Config {
	tmpDir, err := os.MkdirTemp("", "raft-test-"+nodeID)
	assert.NoError(t, err)

	return &config.Config{
		Cluster: config.ClusterConfig{
			Enabled:   true,
			NodeID:    nodeID,
			BindAddr:  bindAddr,
			RaftDir:   tmpDir,
			DataDir:   tmpDir,
			Bootstrap: true,
		},
	}
}

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			Enabled: true,
			NodeID:  "test-node-1",
		},
	}

	m, err := NewManager(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, "test-node-1", m.GetNodeID())
}

func TestNewManager_ClusterDisabled(t *testing.T) {
	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			Enabled: false,
		},
	}

	m, err := NewManager(cfg)
	assert.Error(t, err)
	assert.Nil(t, m)
	assert.Contains(t, err.Error(), "cluster mode is not enabled")
}

func TestManagerStartStop(t *testing.T) {
	cfg := newTestRaftConfig(t, "test-node-1", "127.0.0.1:0")
	defer os.RemoveAll(cfg.Cluster.RaftDir)

	m, err := NewManager(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	// Initially not running
	assert.False(t, m.isRunning)

	// Start the manager
	err = m.Start()
	assert.NoError(t, err)
	assert.True(t, m.isRunning)

	// Starting again should fail
	err = m.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster manager is already running")

	// Stop the manager
	err = m.Stop()
	assert.NoError(t, err)
	assert.False(t, m.isRunning)

	// Stopping again should be fine
	err = m.Stop()
	assert.NoError(t, err)
}

func TestRaft_SingleNode_Bootstrap(t *testing.T) {
	cfg := newTestRaftConfig(t, "test-node-1", "127.0.0.1:0")
	defer os.RemoveAll(cfg.Cluster.RaftDir)

	m, err := NewManager(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, m)

	err = m.Start()
	assert.NoError(t, err)
	defer m.Stop()

	// Wait for the node to become the leader
	waitForLeader(t, m, 10*time.Second)

	assert.True(t, m.IsLeader(), "node should be the leader")
}

func TestRaft_MultiNode_Join(t *testing.T) {
	// Create the bootstrap node first
	bootstrapCfg := newTestRaftConfig(t, "test-node-1", "127.0.0.1:0")
	bootstrapCfg.Cluster.Bootstrap = true
	defer os.RemoveAll(bootstrapCfg.Cluster.RaftDir)

	bootstrapNode, err := NewManager(bootstrapCfg)
	assert.NoError(t, err)

	err = bootstrapNode.Start()
	assert.NoError(t, err)
	defer bootstrapNode.Stop()

	// Wait for bootstrap node to become leader
	waitForLeader(t, bootstrapNode, 10*time.Second)
	assert.True(t, bootstrapNode.IsLeader(), "bootstrap node should be leader")

	// Create follower nodes and add them to the cluster
	followers := make([]*Manager, 2)
	for i := 0; i < 2; i++ {
		nodeID := fmt.Sprintf("test-node-%d", i+2)
		bindAddr := fmt.Sprintf("127.0.0.1:%d", 50061+i)
		cfg := newTestRaftConfig(t, nodeID, bindAddr)
		cfg.Cluster.Bootstrap = false
		cfg.Cluster.JoinAddr = []string{}
		defer os.RemoveAll(cfg.Cluster.RaftDir)

		m, err := NewManager(cfg)
		assert.NoError(t, err)

		err = m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		followers[i] = m

		// Add the follower to the bootstrap node's cluster
		err = bootstrapNode.AddNode(nodeID, bindAddr)
		assert.NoError(t, err)
	}

	// Give some time for the cluster to stabilize
	time.Sleep(2 * time.Second)

	// Check that all nodes see 3 nodes in the cluster
	assert.Len(t, bootstrapNode.GetNodeIDs(), 3, "bootstrap node should see 3 nodes")
	for i, follower := range followers {
		assert.Len(t, follower.GetNodeIDs(), 3, "follower %d should see 3 nodes", i+1)
	}

	// Verify only one leader exists
	allNodes := append([]*Manager{bootstrapNode}, followers...)
	leaders := 0
	for _, m := range allNodes {
		if m.IsLeader() {
			leaders++
		}
	}
	assert.Equal(t, 1, leaders, "expected exactly one leader")
}

func TestSharding_GetShardNode(t *testing.T) {
	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			Enabled: true,
			NodeID:  "test-node-1",
		},
		Indexes: []config.IndexConfig{
			{
				Name: "test-index",
				Distribution: config.IndexDistribution{
					Replicas: 1,
					Shards:   2,
				},
			},
		},
	}

	m, err := NewManager(cfg)
	assert.NoError(t, err)

	err = m.initializeSharding()
	assert.NoError(t, err)

	// Test getting shard node for a key
	nodeID, err := m.GetShardNode("test-index", "test-key")
	assert.NoError(t, err)
	assert.Equal(t, "test-node-1", nodeID)
}

func TestSharding_IsResponsibleForShard(t *testing.T) {
	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			Enabled: true,
			NodeID:  "test-node-1",
		},
		Indexes: []config.IndexConfig{
			{
				Name: "test-index",
				Distribution: config.IndexDistribution{
					Replicas: 1,
					Shards:   2,
				},
			},
		},
	}

	m, err := NewManager(cfg)
	assert.NoError(t, err)

	err = m.initializeSharding()
	assert.NoError(t, err)

	// Test responsibility for shard
	isResponsible := m.IsResponsibleForShard("test-index", "test-key")
	assert.True(t, isResponsible)
}

func TestSharding_GetIndexShards(t *testing.T) {
	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			Enabled: true,
			NodeID:  "test-node-1",
		},
		Indexes: []config.IndexConfig{
			{
				Name: "test-index",
				Distribution: config.IndexDistribution{
					Replicas: 2,
					Shards:   3,
				},
			},
		},
	}

	m, err := NewManager(cfg)
	assert.NoError(t, err)

	err = m.initializeSharding()
	assert.NoError(t, err)

	// Test getting index shards
	shards := m.GetIndexShards("test-index")
	assert.Len(t, shards, 6) // 2 replicas * 3 shards = 6 total shards

	// Verify shard info
	for _, shard := range shards {
		assert.Equal(t, "test-index", shard.IndexName)
		assert.Equal(t, "test-node-1", shard.NodeID)
		assert.True(t, shard.Replica >= 0 && shard.Replica < 2)
		assert.True(t, shard.ShardID >= 0 && shard.ShardID < 3)
	}
}


// waitForLeader waits for a node to become the leader.
func waitForLeader(t *testing.T, m *Manager, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timed out waiting for leader")
		default:
			if m.IsLeader() {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

