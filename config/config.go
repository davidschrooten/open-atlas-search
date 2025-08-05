package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	MongoDB  MongoDBConfig  `mapstructure:"mongodb"`
	Search   SearchConfig   `mapstructure:"search"`
	Indexes  []IndexConfig  `mapstructure:"indexes"`
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
	FlushInterval int    `mapstructure:"flush_interval"` // in seconds
	SyncStatePath string `mapstructure:"sync_state_path"` // Path to store sync state for persistence
}

// IndexConfig represents a search index configuration similar to MongoDB Atlas Search
type IndexConfig struct {
	Name           string                 `mapstructure:"name"`
	Database       string                 `mapstructure:"database"`
	Collection     string                 `mapstructure:"collection"`
	Definition     IndexDefinition        `mapstructure:"definition"`
	TimestampField string                 `mapstructure:"timestamp_field,omitempty"` // Custom field for polling timestamps
	IDField        string                 `mapstructure:"id_field,omitempty"`        // Custom field name for document ID (defaults to "_id")
	PollInterval   int                    `mapstructure:"poll_interval,omitempty"`   // Collection-specific poll interval in seconds
}

// IndexDefinition mirrors MongoDB Atlas Search index structure
type IndexDefinition struct {
	Mappings IndexMappings `mapstructure:"mappings"`
}

// IndexMappings contains field mappings for the index
type IndexMappings struct {
	Dynamic bool                   `mapstructure:"dynamic"`
	Fields  map[string]FieldConfig `mapstructure:"fields"`
}

// FieldConfig represents field-specific indexing configuration
type FieldConfig struct {
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
