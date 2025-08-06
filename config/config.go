package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	MongoDB MongoDBConfig `mapstructure:"mongodb"`
	Search  SearchConfig  `mapstructure:"search"`
	Cluster ClusterConfig `mapstructure:"cluster"`
	Indexes []IndexConfig `mapstructure:"indexes"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// MongoDBConfig contains MongoDB connection settings
type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Timeout  int    `mapstructure:"timeout"` // in seconds
}

// SearchConfig contains search engine settings
type SearchConfig struct {
	IndexPath     string `mapstructure:"index_path"`
	BatchSize     int    `mapstructure:"batch_size"`
	FlushInterval int    `mapstructure:"flush_interval"`  // in seconds
	SyncStatePath string `mapstructure:"sync_state_path"` // Path to store sync state for persistence
	// Performance optimization settings
	WorkerCount     int  `mapstructure:"worker_count"`      // Number of concurrent indexing workers
	BulkIndexing    bool `mapstructure:"bulk_indexing"`     // Enable bulk indexing for better performance
	PrefetchCount   int  `mapstructure:"prefetch_count"`    // Number of documents to prefetch from MongoDB
	IndexBufferSize int  `mapstructure:"index_buffer_size"` // Buffer size for index operations
}

// ClusterConfig contains cluster-specific settings
type ClusterConfig struct {
	Enabled   bool     `mapstructure:"enabled"`   // Enable cluster mode
	NodeID    string   `mapstructure:"node_id"`   // Unique node identifier
	BindAddr  string   `mapstructure:"bind_addr"` // Address to bind Raft transport
	RaftPort  int      `mapstructure:"raft_port"` // Port for Raft communication
	RaftDir   string   `mapstructure:"raft_dir"`  // Directory for Raft logs and snapshots
	Bootstrap bool     `mapstructure:"bootstrap"` // Bootstrap cluster (only for first node)
	JoinAddr  []string `mapstructure:"join_addr"` // Addresses of existing cluster members to join
	DataDir   string   `mapstructure:"data_dir"`  // Directory for cluster data
}

// IndexConfig represents a search index configuration similar to MongoDB Atlas Search
type IndexConfig struct {
	Name           string            `mapstructure:"name"`
	Database       string            `mapstructure:"database"`
	Collection     string            `mapstructure:"collection"`
	Definition     IndexDefinition   `mapstructure:"definition"`
	TimestampField string            `mapstructure:"timestamp_field,omitempty"` // Custom field for polling timestamps
	IDField        string            `mapstructure:"id_field,omitempty"`        // Custom field name for document ID (defaults to "_id")
	PollInterval   int               `mapstructure:"poll_interval,omitempty"`   // Collection-specific poll interval in seconds
	Distribution   IndexDistribution `mapstructure:"distribution,omitempty"`    // Distribution settings for cluster mode
}

// IndexDistribution defines how an index is distributed across the cluster
type IndexDistribution struct {
	Replicas int `mapstructure:"replicas"` // Number of replicas for this index (default: 1)
	Shards   int `mapstructure:"shards"`   // Number of shards for this index (default: 1)
}

// IndexDefinition mirrors MongoDB Atlas Search index structure
type IndexDefinition struct {
	Mappings IndexMappings `mapstructure:"mappings"`
}

// IndexMappings contains field mappings for the index
type IndexMappings struct {
	Dynamic bool          `mapstructure:"dynamic"`
	Fields  []FieldConfig `mapstructure:"fields"`
}

// FieldConfig represents field-specific indexing configuration
type FieldConfig struct {
	Name     string                 `mapstructure:"name"`  // Field name in the index
	Field    string                 `mapstructure:"field"` // Source field name in the document
	Type     string                 `mapstructure:"type"`
	Analyzer string                 `mapstructure:"analyzer,omitempty"`
	Multi    map[string]FieldConfig `mapstructure:"multi,omitempty"`
	Facet    bool                   `mapstructure:"facet,omitempty"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("/etc/open-atlas-search")
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("OAS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("mongodb.timeout", 30)
	viper.SetDefault("search.index_path", "./indexes")
	viper.SetDefault("search.batch_size", 1000)
	viper.SetDefault("search.flush_interval", 30)
	viper.SetDefault("search.sync_state_path", "./sync_state.json")
	// Performance optimization defaults
	viper.SetDefault("search.worker_count", 4)        // 4 concurrent workers
	viper.SetDefault("search.bulk_indexing", true)    // Enable bulk indexing
	viper.SetDefault("search.prefetch_count", 5000)   // Prefetch 5000 documents
	viper.SetDefault("search.index_buffer_size", 100) // Buffer 100 operations
	// Cluster defaults
	viper.SetDefault("cluster.enabled", false)
	viper.SetDefault("cluster.node_id", "")
	viper.SetDefault("cluster.bind_addr", "0.0.0.0:7946")
	viper.SetDefault("cluster.raft_port", 7946)
	viper.SetDefault("cluster.raft_dir", "./raft")
	viper.SetDefault("cluster.bootstrap", false)
	viper.SetDefault("cluster.join_addr", []string{})
	viper.SetDefault("cluster.data_dir", "./cluster_data")
}

// GetMongoURI returns the complete MongoDB connection URI
func (c *MongoDBConfig) GetMongoURI() string {
	if c.URI != "" {
		return c.URI
	}

	// Build URI from components if not provided directly
	uri := "mongodb://"
	if c.Username != "" && c.Password != "" {
		uri += fmt.Sprintf("%s:%s@", c.Username, c.Password)
	}
	uri += "localhost:27017"
	return uri
}
