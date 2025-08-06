package cluster

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
)

// CommandType represents the type of command.
type CommandType int

// Command types for the FSM
const (
	// AddShardCommand adds a new shard
	AddShardCommand CommandType = iota
	// RemoveShardCommand removes a shard
	RemoveShardCommand
	// UpdateShardCommand updates shard information
	UpdateShardCommand
	// IndexDistributionCommand updates index distribution
	IndexDistributionCommand
)

// Command represents a command in the Raft log.
type Command struct {
	Type    CommandType `json:"type"`
	ShardID string      `json:"shard_id"`
	Data    interface{} `json:"data,omitempty"`
}

// FSM implements the raft.FSM interface for our cluster state machine.
type FSM struct {
	shards      map[string]interface{} // shard_id -> shard_data
	indexShards map[string][]string    // index_name -> shard_ids
}

// NewFSM creates a new FSM.
func NewFSM() *FSM {
	return &FSM{
		shards:      make(map[string]interface{}),
		indexShards: make(map[string][]string),
	}
}

// Apply applies a Raft log entry to the FSM.
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %v", err)
	}

	switch cmd.Type {
	case AddShardCommand:
		f.shards[cmd.ShardID] = cmd.Data
		return fmt.Sprintf("shard %s added", cmd.ShardID)

	case RemoveShardCommand:
		delete(f.shards, cmd.ShardID)
		return fmt.Sprintf("shard %s removed", cmd.ShardID)

	case UpdateShardCommand:
		f.shards[cmd.ShardID] = cmd.Data
		return fmt.Sprintf("shard %s updated", cmd.ShardID)

	case IndexDistributionCommand:
		// Handle index distribution changes
		if shardInfo, ok := cmd.Data.(map[string]interface{}); ok {
			if indexName, exists := shardInfo["index_name"].(string); exists {
				if shardList, exists := shardInfo["shards"].([]string); exists {
					f.indexShards[indexName] = shardList
					return fmt.Sprintf("index %s distribution updated", indexName)
				}
			}
		}
		return fmt.Errorf("invalid index distribution data")

	default:
		return fmt.Errorf("unknown command type: %v", cmd.Type)
	}
}

// Snapshot returns a snapshot of the current state.
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	// Clone the state
	shards := make(map[string]interface{})
	for k, v := range f.shards {
		shards[k] = v
	}

	indexShards := make(map[string][]string)
	for k, v := range f.indexShards {
		indexShards[k] = append([]string(nil), v...)
	}

	return &FSMSnapshot{
		shards:      shards,
		indexShards: indexShards,
	}, nil
}

// Restore restores the FSM from a snapshot.
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var state struct {
		Shards      map[string]interface{} `json:"shards"`
		IndexShards map[string][]string    `json:"index_shards"`
	}

	if err := json.NewDecoder(rc).Decode(&state); err != nil {
		return err
	}

	f.shards = state.Shards
	f.indexShards = state.IndexShards
	return nil
}

// GetShards returns the current shard state
func (f *FSM) GetShards() map[string]interface{} {
	return f.shards
}

// GetIndexShards returns the index shard mappings
func (f *FSM) GetIndexShards() map[string][]string {
	return f.indexShards
}

// FSMSnapshot implements the raft.FSMSnapshot interface.
type FSMSnapshot struct {
	shards      map[string]interface{}
	indexShards map[string][]string
}

// Persist saves the snapshot to the given sink.
func (s *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	state := map[string]interface{}{
		"shards":       s.shards,
		"index_shards": s.indexShards,
	}

	if err := json.NewEncoder(sink).Encode(state); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

// Release is called when the snapshot is no longer needed.
func (s *FSMSnapshot) Release() {
	// Nothing to release in this simple implementation
}
