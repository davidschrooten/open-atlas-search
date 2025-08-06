package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  host: "localhost"
  port: 9090

mongodb:
  uri: "mongodb://localhost:27017"
  database: "testdb"
  timeout: 60

search:
  index_path: "/tmp/indexes"
  batch_size: 500
  flush_interval: 15
  sync_state_path: "/tmp/sync_state.json"

indexes:
  - name: "test_index"
    database: "testdb"
    collection: "testcol"
    id_field: "custom_id"
    timestamp_field: "modified_at"
    poll_interval: 10
    definition:
      mappings:
        dynamic: true
        fields:
          - name: "title"
            type: "text"
            analyzer: "standard"
          - name: "price"
            type: "numeric"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify server config
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected server host 'localhost', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected server port 9090, got %d", cfg.Server.Port)
	}

	// Verify mongodb config
	if cfg.MongoDB.URI != "mongodb://localhost:27017" {
		t.Errorf("Expected mongodb uri 'mongodb://localhost:27017', got '%s'", cfg.MongoDB.URI)
	}
	if cfg.MongoDB.Database != "testdb" {
		t.Errorf("Expected mongodb database 'testdb', got '%s'", cfg.MongoDB.Database)
	}
	if cfg.MongoDB.Timeout != 60 {
		t.Errorf("Expected mongodb timeout 60, got %d", cfg.MongoDB.Timeout)
	}

	// Verify search config
	if cfg.Search.IndexPath != "/tmp/indexes" {
		t.Errorf("Expected search index_path '/tmp/indexes', got '%s'", cfg.Search.IndexPath)
	}
	if cfg.Search.BatchSize != 500 {
		t.Errorf("Expected search batch_size 500, got %d", cfg.Search.BatchSize)
	}
	if cfg.Search.FlushInterval != 15 {
		t.Errorf("Expected search flush_interval 15, got %d", cfg.Search.FlushInterval)
	}
	if cfg.Search.SyncStatePath != "/tmp/sync_state.json" {
		t.Errorf("Expected search sync_state_path '/tmp/sync_state.json', got '%s'", cfg.Search.SyncStatePath)
	}

	// Verify indexes config
	if len(cfg.Indexes) != 1 {
		t.Fatalf("Expected 1 index, got %d", len(cfg.Indexes))
	}

	index := cfg.Indexes[0]
	if index.Name != "test_index" {
		t.Errorf("Expected index name 'test_index', got '%s'", index.Name)
	}
	if index.Database != "testdb" {
		t.Errorf("Expected index database 'testdb', got '%s'", index.Database)
	}
	if index.Collection != "testcol" {
		t.Errorf("Expected index collection 'testcol', got '%s'", index.Collection)
	}
	if index.IDField != "custom_id" {
		t.Errorf("Expected index id_field 'custom_id', got '%s'", index.IDField)
	}
	if index.TimestampField != "modified_at" {
		t.Errorf("Expected index timestamp_field 'modified_at', got '%s'", index.TimestampField)
	}
	if index.PollInterval != 10 {
		t.Errorf("Expected index poll_interval 10, got %d", index.PollInterval)
	}

	// Verify index definition
	if !index.Definition.Mappings.Dynamic {
		t.Error("Expected index mappings to be dynamic")
	}

	if len(index.Definition.Mappings.Fields) != 2 {
		t.Errorf("Expected 2 field mappings, got %d", len(index.Definition.Mappings.Fields))
	}

	// Find title field
	var titleField *FieldConfig
	for i := range index.Definition.Mappings.Fields {
		if index.Definition.Mappings.Fields[i].Name == "title" {
			titleField = &index.Definition.Mappings.Fields[i]
			break
		}
	}
	if titleField == nil {
		t.Error("Title field not found")
	} else {
		if titleField.Type != "text" {
			t.Errorf("Expected title field type 'text', got '%s'", titleField.Type)
		}
		if titleField.Analyzer != "standard" {
			t.Errorf("Expected title field analyzer 'standard', got '%s'", titleField.Analyzer)
		}
	}

	// Find price field
	var priceField *FieldConfig
	for i := range index.Definition.Mappings.Fields {
		if index.Definition.Mappings.Fields[i].Name == "price" {
			priceField = &index.Definition.Mappings.Fields[i]
			break
		}
	}
	if priceField == nil {
		t.Error("Price field not found")
	} else {
		if priceField.Type != "numeric" {
			t.Errorf("Expected price field type 'numeric', got '%s'", priceField.Type)
		}
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	_, err := LoadConfig("/tmp/non_existent_config.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent config file")
	}
}

func TestLoadConfig_WithDefaults(t *testing.T) {
	// Create minimal config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
mongodb:
  uri: "mongodb://localhost:27017"

indexes:
  - name: "minimal_index"
    database: "testdb"
    collection: "testcol"
    definition:
      mappings:
        dynamic: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults are applied
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default server host '0.0.0.0', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default server port 8080, got %d", cfg.Server.Port)
	}
	if cfg.MongoDB.Timeout != 30 {
		t.Errorf("Expected default mongodb timeout 30, got %d", cfg.MongoDB.Timeout)
	}
	if cfg.Search.IndexPath != "./indexes" {
		t.Errorf("Expected default search index_path './indexes', got '%s'", cfg.Search.IndexPath)
	}
	if cfg.Search.BatchSize != 1000 {
		t.Errorf("Expected default search batch_size 1000, got %d", cfg.Search.BatchSize)
	}
	if cfg.Search.FlushInterval != 30 {
		t.Errorf("Expected default search flush_interval 30, got %d", cfg.Search.FlushInterval)
	}
	if cfg.Search.SyncStatePath != "./sync_state.json" {
		t.Errorf("Expected default search sync_state_path './sync_state.json', got '%s'", cfg.Search.SyncStatePath)
	}

	// Verify index uses defaults for optional fields
	index := cfg.Indexes[0]
	if index.IDField != "" {
		t.Errorf("Expected empty id_field (defaults to '_id'), got '%s'", index.IDField)
	}
	if index.TimestampField != "" {
		t.Errorf("Expected empty timestamp_field (defaults to 'updated_at'), got '%s'", index.TimestampField)
	}
	if index.PollInterval != 0 {
		t.Errorf("Expected empty poll_interval (uses default), got %d", index.PollInterval)
	}
}

func TestMongoDBConfig_GetMongoURI(t *testing.T) {
	tests := []struct {
		name     string
		config   MongoDBConfig
		expected string
	}{
		{
			name: "URI provided",
			config: MongoDBConfig{
				URI: "mongodb://custom:27017/mydb",
			},
			expected: "mongodb://custom:27017/mydb",
		},
		{
			name:     "No URI, no credentials",
			config:   MongoDBConfig{},
			expected: "mongodb://localhost:27017",
		},
		{
			name: "No URI, with credentials",
			config: MongoDBConfig{
				Username: "user",
				Password: "pass",
			},
			expected: "mongodb://user:pass@localhost:27017",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetMongoURI()
			if result != tt.expected {
				t.Errorf("Expected URI '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("OAS_SERVER_PORT", "9999")
	os.Setenv("OAS_MONGODB_DATABASE", "env_db")
	os.Setenv("OAS_SEARCH_BATCH_SIZE", "2000")
	defer func() {
		os.Unsetenv("OAS_SERVER_PORT")
		os.Unsetenv("OAS_MONGODB_DATABASE")
		os.Unsetenv("OAS_SEARCH_BATCH_SIZE")
	}()

	// Create minimal config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
server:
  port: 8080
mongodb:
  database: "config_db"
search:
  batch_size: 1000
indexes:
  - name: "test_index"
    database: "testdb"
    collection: "testcol"
    definition:
      mappings:
        dynamic: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Reset viper to ensure clean state
	viper.Reset()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variables override config file values
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected port 9999 from env var, got %d", cfg.Server.Port)
	}
	if cfg.MongoDB.Database != "env_db" {
		t.Errorf("Expected database 'env_db' from env var, got '%s'", cfg.MongoDB.Database)
	}
	if cfg.Search.BatchSize != 2000 {
		t.Errorf("Expected batch_size 2000 from env var, got %d", cfg.Search.BatchSize)
	}
}

func TestSetDefaults(t *testing.T) {
	// Reset viper to ensure clean state
	viper.Reset()

	setDefaults()

	// Verify all defaults are set
	if viper.GetString("server.host") != "0.0.0.0" {
		t.Errorf("Expected default server.host '0.0.0.0', got '%s'", viper.GetString("server.host"))
	}
	if viper.GetInt("server.port") != 8080 {
		t.Errorf("Expected default server.port 8080, got %d", viper.GetInt("server.port"))
	}
	if viper.GetInt("mongodb.timeout") != 30 {
		t.Errorf("Expected default mongodb.timeout 30, got %d", viper.GetInt("mongodb.timeout"))
	}
	if viper.GetString("search.index_path") != "./indexes" {
		t.Errorf("Expected default search.index_path './indexes', got '%s'", viper.GetString("search.index_path"))
	}
	if viper.GetInt("search.batch_size") != 1000 {
		t.Errorf("Expected default search.batch_size 1000, got %d", viper.GetInt("search.batch_size"))
	}
	if viper.GetInt("search.flush_interval") != 30 {
		t.Errorf("Expected default search.flush_interval 30, got %d", viper.GetInt("search.flush_interval"))
	}
	if viper.GetString("search.sync_state_path") != "./sync_state.json" {
		t.Errorf("Expected default search.sync_state_path './sync_state.json', got '%s'", viper.GetString("search.sync_state_path"))
	}
}
