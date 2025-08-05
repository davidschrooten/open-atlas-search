package cluster

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
)

// CommandType represents the type of command.
type CommandType int

const (
	AddShardCommand CommandType = iota
	RemoveShardCommand
	UpdateShardCommand
)

// Command represents a command in the Raft log.
type Command struct {
	Type    CommandType `json:"type"`
	ShardID string      `json:"shard_id"`
	Data    interface{} `json:"data,omitempty"`
}

// FSM implements the raft.FSM interface for our cluster state machine.
type FSM struct {
	shards map[string]interface{}
}

// NewFSM creates a new FSM.
func NewFSM() *FSM {
	return &FSM{
		shards: make(map[string]interface{}),
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
	default:
		return fmt.Errorf("unknown command type: %v", cmd.Type)
	}
}

// Snapshot returns a snapshot of the current state.
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	// Clone the shards map
	shards := make(map[string]interface{})
	for k, v := range f.shards {
		shards[k] = v
	}
	
	return &FSMSnapshot{shards: shards}, nil
}

// Restore restores the FSM from a snapshot.
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var shards map[string]interface{}
	if err := json.NewDecoder(rc).Decode(&shards); err != nil {
		return err
	}

	f.shards = shards
	return nil
}

// FSMSnapshot implements the raft.FSMSnapshot interface.
type FSMSnapshot struct {
	shards map[string]interface{}
}

// Persist saves the snapshot to the given sink.
func (s *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	if err := json.NewEncoder(sink).Encode(s.shards); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

// Release is called when the snapshot is no longer needed.
func (s *FSMSnapshot) Release() {
	// Nothing to release in this simple implementation
}
