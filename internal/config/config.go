package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

// Config represents the application configuration.
type Config struct {
	// DBURL is the database connection string
	DBURL string `json:"db_url"`

	// CurrentUserName is the username for the current user
	CurrentUserName string `json:"current_user_name"`
}

// Read reads the configuration file from the given home directory
// and returns the parsed Config struct.
func Read(homeDir string) (Config, error) {
	configPath := filepath.Join(homeDir, ".gatorconfig.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func SetUser(homeDir string, userName string) error {
	configPath := filepath.Join(homeDir, ".gatorconfig.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	config.CurrentUserName = userName

	data, err = json.Marshal(config)
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Write writes the given Config struct to the configuration file
// in the given home directory.
func Write(homeDir string, config Config) error {
	configPath := filepath.Join(homeDir, configFileName)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath(homeDir string) string {
	configPath := filepath.Join(homeDir, configFileName)
	return configPath
}
