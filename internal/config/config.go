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

// GetHomeDir returns the user's home directory
func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir, nil
}

// Read reads the configuration file from the user's home directory
// and returns the parsed Config struct.
func Read(dirOverride string) (Config, error) {
	var configDir string
	var err error

	if dirOverride != "" {
		configDir = dirOverride
	} else {
		configDir, err = GetHomeDir()
		if err != nil {
			return Config{}, err
		}
	}

	configPath := filepath.Join(configDir, configFileName)

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
	configPath := filepath.Join(homeDir, configFileName)

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
// in the specified directory or user's home directory if none provided.
func Write(dirOverride string, config Config) error {
	var configDir string
	var err error

	if dirOverride != "" {
		configDir = dirOverride
	} else {
		configDir, err = GetHomeDir()
		if err != nil {
			return err
		}
	}

	configPath := filepath.Join(configDir, configFileName)

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
func GetConfigPath(dirOverride string) string {
	var configDir string

	if dirOverride != "" {
		configDir = dirOverride
	} else {
		var err error
		configDir, err = GetHomeDir()
		if err != nil {
			return ""
		}
	}

	configPath := filepath.Join(configDir, configFileName)
	return configPath
}
