package cluster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
)

func TestFSM_Apply(t *testing.T) {
	fsm := NewFSM()

	tests := []struct {
		name        string
		command     Command
		expectedRes string
	}{
		{
			name: "AddShardCommand",
			command: Command{
				Type:    AddShardCommand,
				ShardID: "shard-1",
				Data:    map[string]interface{}{"key": "value"},
			},
			expectedRes: "shard shard-1 added",
		},
		{
			name: "UpdateShardCommand",
			command: Command{
				Type:    UpdateShardCommand,
				ShardID: "shard-1",
				Data:    map[string]interface{}{"key": "updated-value"},
			},
			expectedRes: "shard shard-1 updated",
		},
		{
			name: "RemoveShardCommand",
			command: Command{
				Type:    RemoveShardCommand,
				ShardID: "shard-1",
			},
			expectedRes: "shard shard-1 removed",
		},
		{
			name: "IndexDistributionCommand",
			command: Command{
				Type: IndexDistributionCommand,
				Data: map[string]interface{}{
					"index_name": "test-index",
					"shards":     []interface{}{"shard-1", "shard-2"},
				},
			},
			expectedRes: "index test-index distribution updated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.command)
			assert.NoError(t, err)

			log := &raft.Log{
				Data: data,
			}

			result := fsm.Apply(log)
			assert.Equal(t, tt.expectedRes, result)
		})
	}
}

func TestFSM_Apply_InvalidCommand(t *testing.T) {
	fsm := NewFSM()

	// Test with invalid JSON
	log := &raft.Log{
		Data: []byte("invalid json"),
	}

	result := fsm.Apply(log)
	assert.Contains(t, result.(error).Error(), "failed to unmarshal command")
}

func TestFSM_Apply_UnknownCommandType(t *testing.T) {
	fsm := NewFSM()

	command := Command{
		Type:    CommandType(999), // Unknown command type
		ShardID: "shard-1",
	}

	data, err := json.Marshal(command)
	assert.NoError(t, err)

	log := &raft.Log{
		Data: data,
	}

	result := fsm.Apply(log)
	assert.Contains(t, result.(error).Error(), "unknown command type")
}

func TestFSM_Snapshot(t *testing.T) {
	fsm := NewFSM()

	// Add some data to the FSM
	fsm.shards["shard-1"] = map[string]interface{}{"key": "value"}
	fsm.indexShards["index-1"] = []string{"shard-1", "shard-2"}

	snapshot, err := fsm.Snapshot()
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)

	// Check that the snapshot contains the expected data
	fsmSnapshot := snapshot.(*FSMSnapshot)
	assert.Equal(t, map[string]interface{}{"key": "value"}, fsmSnapshot.shards["shard-1"])
	assert.Equal(t, []string{"shard-1", "shard-2"}, fsmSnapshot.indexShards["index-1"])
}

func TestFSM_Restore(t *testing.T) {
	fsm := NewFSM()

	// Create test data
	state := map[string]interface{}{
		"shards": map[string]interface{}{
			"shard-1": map[string]interface{}{"key": "value"},
		},
		"index_shards": map[string][]string{
			"index-1": {"shard-1", "shard-2"},
		},
	}

	// Create a reader with the serialized state
	data, err := json.Marshal(state)
	assert.NoError(t, err)
	reader := &readCloser{bytes.NewReader(data)}

	// Restore the FSM from the snapshot
	err = fsm.Restore(reader)
	assert.NoError(t, err)

	// Verify the restored data
	assert.Equal(t, map[string]interface{}{"key": "value"}, fsm.shards["shard-1"])
	assert.Equal(t, []string{"shard-1", "shard-2"}, fsm.indexShards["index-1"])
}

func TestFSM_GetShards(t *testing.T) {
	fsm := NewFSM()
	fsm.shards["shard-1"] = map[string]interface{}{"key": "value"}

	shards := fsm.GetShards()
	assert.Equal(t, map[string]interface{}{"key": "value"}, shards["shard-1"])
}

func TestFSM_GetIndexShards(t *testing.T) {
	fsm := NewFSM()
	fsm.indexShards["index-1"] = []string{"shard-1", "shard-2"}

	indexShards := fsm.GetIndexShards()
	assert.Equal(t, []string{"shard-1", "shard-2"}, indexShards["index-1"])
}

func TestFSMSnapshot_Persist(t *testing.T) {
	snapshot := &FSMSnapshot{
		shards: map[string]interface{}{
			"shard-1": map[string]interface{}{"key": "value"},
		},
		indexShards: map[string][]string{
			"index-1": {"shard-1", "shard-2"},
		},
	}

	// Create a mock sink
	buf := &bytes.Buffer{}
	sink := &mockSnapshotSink{
		buffer: buf,
	}

	err := snapshot.Persist(sink)
	assert.NoError(t, err)

	// Verify the persisted data
	var state map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &state)
	assert.NoError(t, err)

	shards := state["shards"].(map[string]interface{})
	assert.Equal(t, map[string]interface{}{"key": "value"}, shards["shard-1"])

	indexShards := state["index_shards"].(map[string]interface{})
	assert.Equal(t, []interface{}{"shard-1", "shard-2"}, indexShards["index-1"])
}

func TestFSMSnapshot_Release(t *testing.T) {
	snapshot := &FSMSnapshot{}
	// Release should not panic or cause errors
	snapshot.Release()
}

// mockSnapshotSink implements raft.SnapshotSink for testing
type mockSnapshotSink struct {
	buffer   *bytes.Buffer
	canceled bool
	closed   bool
}

func (m *mockSnapshotSink) Write(p []byte) (n int, err error) {
	if m.canceled {
		return 0, fmt.Errorf("snapshot aborted")
	}
	return m.buffer.Write(p)
}

func (m *mockSnapshotSink) Close() error {
	m.closed = true
	return nil
}

func (m *mockSnapshotSink) Cancel() error {
	m.canceled = true
	return nil
}

func (m *mockSnapshotSink) ID() string {
	return "test-snapshot"
}

// readCloser wraps a bytes.Reader to implement io.ReadCloser
type readCloser struct {
	*bytes.Reader
}

func (rc *readCloser) Close() error {
	return nil
}
