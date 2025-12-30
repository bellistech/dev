// Package config provides configuration management for the metrics system.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentConfig represents the agent configuration.
type AgentConfig struct {
	Server     AgentServerConfig  `yaml:"server"`
	Collection CollectionConfig   `yaml:"collection"`
	Agent      AgentInfo          `yaml:"agent"`
	Logging    LoggingConfig      `yaml:"logging"`
}

// AgentServerConfig represents server connection settings for the agent.
type AgentServerConfig struct {
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
	TLS     TLSConfig     `yaml:"tls"`
}

// CollectionConfig represents metric collection settings.
type CollectionConfig struct {
	Interval   time.Duration `yaml:"interval"`
	Collectors []string      `yaml:"collectors"`
	BatchSize  int           `yaml:"batch_size"`
}

// AgentInfo represents agent identification.
type AgentInfo struct {
	ID     string            `yaml:"id"`
	Labels map[string]string `yaml:"labels"`
}

// LoggingConfig represents logging configuration.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// TLSConfig represents TLS configuration.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// ServerConfig represents the server configuration.
type ServerConfig struct {
	GRPC     GRPCConfig     `yaml:"grpc"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// GRPCConfig represents gRPC server settings.
type GRPCConfig struct {
	Port    int       `yaml:"port"`
	TLS     TLSConfig `yaml:"tls"`
	MaxRecv int       `yaml:"max_recv_msg_size"`
}

// DatabaseConfig represents PostgreSQL configuration.
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// LoadAgentConfig loads agent configuration from a YAML file.
func LoadAgentConfig(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &AgentConfig{
		// Set defaults
		Server: AgentServerConfig{
			Address: "localhost:9090",
			Timeout: 30 * time.Second,
		},
		Collection: CollectionConfig{
			Interval:   60 * time.Second,
			Collectors: []string{"cpu", "memory", "disk", "network", "uptime"},
			BatchSize:  100,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// LoadServerConfig loads server configuration from a YAML file.
func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &ServerConfig{
		// Set defaults
		GRPC: GRPCConfig{
			Port:    9090,
			MaxRecv: 16 * 1024 * 1024, // 16MB
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "metrics",
			Password:        "metrics",
			Database:        "metrics",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// ConnectionString returns a PostgreSQL connection string.
func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}
