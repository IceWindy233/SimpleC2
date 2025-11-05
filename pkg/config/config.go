package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// TeamServerConfig holds all configuration for the TeamServer.
type TeamServerConfig struct {
	GRPC struct {
		Port    string `yaml:"port"`
		Certs struct {
			ServerCert string `yaml:"server_cert"`
			ServerKey  string `yaml:"server_key"`
			CACert     string `yaml:"ca_cert"`
		} `yaml:"certs"`
	} `yaml:"grpc"`
	API struct {
		Port string `yaml:"port"`
	} `yaml:"api"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	LootDir  string         `yaml:"loot_dir"`
	UploadsDir string       `yaml:"uploads_dir"`
}

// DatabaseConfig holds database-specific configuration.
type DatabaseConfig struct {
	Type string `yaml:"type"`           // "postgres" or "sqlite"
	DSN  string `yaml:"dsn,omitempty"`  // Optional: For Postgres
	Path string `yaml:"path,omitempty"` // Optional: For SQLite
}

// AuthConfig holds authentication-related configuration.
type AuthConfig struct {
	APIKey           string `yaml:"api_key"`
	OperatorPassword string `yaml:"operator_password"`
}

// ListenerConfig holds all configuration for the Listener.
type ListenerConfig struct {
	TeamServer struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"teamserver"`
	Listener struct {
		Name string `yaml:"name"`
		Port string `yaml:"port"`
	} `yaml:"listener"`
	Auth struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"auth"`
	Certs struct {
		ClientCert string `yaml:"client_cert"`
		ClientKey  string `yaml:"client_key"`
		CACert     string `yaml:"ca_cert"`
		PrivateKey string `yaml:"private_key"`
	} `yaml:"certs"`
}

// LoadConfig reads a YAML file from the given path and unmarshals it into the provided config struct.
func LoadConfig(path string, config interface{}) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(file, config)
}
