package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/davidschrooten/open-atlas-search/config"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/serialx/hashring"
)

// ShardInfo represents information about a shard
type ShardInfo struct {
	IndexName string `json:"index_name"`
	ShardID   int    `json:"shard_id"`
	Replica   int    `json:"replica"`
	NodeID    string `json:"node_id"`
}

// Manager handles cluster operations and coordination
type Manager struct {
	config      *config.Config
	raft        *raft.Raft
	fsm         *FSM
	ring        *hashring.HashRing
	nodeID      string
	shards      map[string][]ShardInfo // index_name -> shards
	isLeader    bool
	ctx         context.Context
	cancel      context.CancelFunc
	isRunning   bool
}

// NewManager creates a new cluster manager
func NewManager(cfg *config.Config) (*Manager, error) {
	if !cfg.Cluster.Enabled {
		return nil, fmt.Errorf("cluster mode is not enabled")
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	nodeID := cfg.Cluster.NodeID
	if nodeID == "" {
		// Generate a unique node ID if not provided
		hostname, _ := os.Hostname()
		nodeID = fmt.Sprintf("node-%s-%d", hostname, time.Now().Unix())
	}

	m := &Manager{
		config:    cfg,
		nodeID:    nodeID,
		shards:    make(map[string][]ShardInfo),
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}

	return m, nil
}

// Start initializes and starts the cluster
func (m *Manager) Start() error {
	if m.isRunning {
		return fmt.Errorf("cluster manager is already running")
	}

	// Create directories
	if err := os.MkdirAll(m.config.Cluster.RaftDir, 0755); err != nil {
		return fmt.Errorf("failed to create raft directory: %w", err)
	}

	if err := os.MkdirAll(m.config.Cluster.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Setup Raft
	if err := m.setupRaft(); err != nil {
		return fmt.Errorf("failed to setup raft: %w", err)
	}

	// Initialize sharding for indexes
	if err := m.initializeSharding(); err != nil {
		return fmt.Errorf("failed to initialize sharding: %w", err)
	}

	// Start leadership monitoring
	go m.monitorLeadership()

	m.isRunning = true
	log.Printf("Cluster manager started for node %s", m.nodeID)
	
	return nil
}

// Stop shuts down the cluster manager
func (m *Manager) Stop() error {
	if !m.isRunning {
		return nil
	}

	m.cancel()
	
	if m.raft != nil {
		if err := m.raft.Shutdown().Error(); err != nil {
			return fmt.Errorf("failed to shutdown raft: %w", err)
		}
	}

	m.isRunning = false
	log.Printf("Cluster manager stopped for node %s", m.nodeID)
	
	return nil
}

// setupRaft configures and starts the Raft consensus protocol
func (m *Manager) setupRaft() error {
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(m.nodeID)
	raftConfig.Logger = log.New(os.Stdout, "[RAFT] ", log.LstdFlags)

	// Create transport
	addr, err := net.ResolveTCPAddr("tcp", m.config.Cluster.BindAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve bind address: %w", err)
	}

	transport, err := raft.NewTCPTransport(m.config.Cluster.BindAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create raft transport: %w", err)
	}

	// Create stores
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(m.config.Cluster.RaftDir, "raft-log.bolt"))
	if err != nil {
		return fmt.Errorf("failed to create log store: %w", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(m.config.Cluster.RaftDir, "raft-stable.bolt"))
	if err != nil {
		return fmt.Errorf("failed to create stable store: %w", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(m.config.Cluster.RaftDir, 3, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create snapshot store: %w", err)
	}

	// Create FSM
	m.fsm = NewFSM()

	// Create Raft
	m.raft, err = raft.NewRaft(raftConfig, m.fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return fmt.Errorf("failed to create raft: %w", err)
	}

	// Bootstrap or join cluster
	if m.config.Cluster.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(m.nodeID),
					Address: transport.LocalAddr(),
				},
			},
		}
		m.raft.BootstrapCluster(configuration)
		log.Printf("Bootstrapped cluster with node %s", m.nodeID)
	} else if len(m.config.Cluster.JoinAddr) > 0 {
		// Join existing cluster
		for _, addr := range m.config.Cluster.JoinAddr {
			if err := m.joinCluster(addr); err != nil {
				log.Printf("Failed to join cluster at %s: %v", addr, err)
				continue
			}
			log.Printf("Successfully joined cluster at %s", addr)
			break
		}
	}

	return nil
}

// joinCluster attempts to join an existing cluster
func (m *Manager) joinCluster(leaderAddr string) error {
	// This is a simplified join process
	// In a real implementation, you'd need a proper join RPC
	configFuture := m.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(m.nodeID) {
			log.Printf("Node %s already part of cluster", m.nodeID)
			return nil
		}
	}

	// Add this node to the cluster
	addFuture := m.raft.AddVoter(raft.ServerID(m.nodeID), raft.ServerAddress(m.config.Cluster.BindAddr), 0, 0)
	return addFuture.Error()
}

// initializeSharding sets up consistent hashing for indexes
func (m *Manager) initializeSharding() error {
	nodes := []string{}
	
	// Add current node's shards to the ring
	for _, indexCfg := range m.config.Indexes {
		replicas := indexCfg.Distribution.Replicas
		shards := indexCfg.Distribution.Shards
		
		if replicas == 0 {
			replicas = 1
		}
		if shards == 0 {
			shards = 1
		}

		var indexShards []ShardInfo
		for r := 0; r < replicas; r++ {
			for s := 0; s < shards; s++ {
				shardInfo := ShardInfo{
					IndexName: indexCfg.Name,
					ShardID:   s,
					Replica:   r,
					NodeID:    m.nodeID,
				}
				indexShards = append(indexShards, shardInfo)
				
				// Add to consistent hash ring
				nodeKey := fmt.Sprintf("%s:%s:r%d:s%d", m.nodeID, indexCfg.Name, r, s)
				nodes = append(nodes, nodeKey)
			}
		}
		
		m.shards[indexCfg.Name] = indexShards
	}

	m.ring = hashring.New(nodes)
	return nil
}

// monitorLeadership monitors Raft leadership changes
func (m *Manager) monitorLeadership() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			wasLeader := m.isLeader
			m.isLeader = m.raft.State() == raft.Leader
			
			if m.isLeader && !wasLeader {
				log.Printf("Node %s became leader", m.nodeID)
				// Handle leadership transition
				m.onBecomeLeader()
			} else if !m.isLeader && wasLeader {
				log.Printf("Node %s lost leadership", m.nodeID)
				// Handle leadership loss
				m.onLoseLeadership()
			}
		}
	}
}

// onBecomeLeader handles becoming the cluster leader
func (m *Manager) onBecomeLeader() {
	// Redistribute shards if needed
	// Sync cluster state
	log.Printf("Node %s is now the cluster leader", m.nodeID)
}

// onLoseLeadership handles losing cluster leadership
func (m *Manager) onLoseLeadership() {
	log.Printf("Node %s is no longer the cluster leader", m.nodeID)
}

// GetShardNode returns the node responsible for a given key
func (m *Manager) GetShardNode(indexName, key string) (string, error) {
	if m.ring == nil {
		return m.nodeID, nil // Standalone mode
	}

	node, ok := m.ring.GetNode(fmt.Sprintf("%s:%s", indexName, key))
	if !ok {
		return "", fmt.Errorf("no node found for key %s in index %s", key, indexName)
	}

	// Extract node ID from the node key
	parts := strings.Split(node, ":")
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid node key format: %s", node)
	}

	return parts[0], nil
}

// IsResponsibleForShard checks if this node is responsible for a given shard
func (m *Manager) IsResponsibleForShard(indexName, key string) bool {
	nodeID, err := m.GetShardNode(indexName, key)
	if err != nil {
		return true // Default to handling locally on error
	}
	return nodeID == m.nodeID
}

// GetIndexShards returns shard information for an index
func (m *Manager) GetIndexShards(indexName string) []ShardInfo {
	return m.shards[indexName]
}

// IsClusterEnabled returns whether cluster mode is enabled
func (m *Manager) IsClusterEnabled() bool {
	return m.config.Cluster.Enabled
}

// IsLeader returns whether this node is the cluster leader
func (m *Manager) IsLeader() bool {
	return m.isLeader
}

// GetNodeID returns the current node's ID
func (m *Manager) GetNodeID() string {
	return m.nodeID
}
