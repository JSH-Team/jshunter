package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jsh-team/jshunter/internal/utils/files"
	"github.com/jsh-team/jshunter/internal/utils/logger"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	ConfigDirName  = "jshunter"
	ConfigFileName = "config.yaml"

	DefaultPort = 20450
	DefaultMode = "local"
)

func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, ConfigDirName), nil
}

func GetDefaultTargetStorageDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "targets", Target), nil
}

// LoadConfig loads the config from the config file
// If the config file does not exist, it creates a default config and saves it to the config file
func LoadConfig() {
	configPath, err := GetConfigDir()
	if err != nil {
		logger.Error("Error getting config dir: %v", err)
		return
	}
	configFile := filepath.Join(configPath, ConfigFileName)
	defaultTargetDir := filepath.Join(configPath, "targets")

	// Create the config directory if it does not exist
	if err := os.MkdirAll(configPath, 0755); err != nil {
		logger.Error("Error creating config path: %v", err)
		return
	}

	if err := os.MkdirAll(defaultTargetDir, 0755); err != nil {
		logger.Error("Error creating default target dir: %v", err)
		return
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		defaultConfig := Config{
			Targets: make(map[string]TargetConfig),
		}

		out, err := yaml.Marshal(defaultConfig)
		if err != nil {
			logger.Error("Error marshaling default config: %v", err)
			return
		}

		if err := os.WriteFile(configFile, out, 0644); err != nil {
			logger.Error("Error writing default config file: %v", err)
			return
		}
	}

	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		logger.Error("Error reading config: %v", err)
		return
	}

	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		logger.Error("Error unmarshalling config: %v", err)
		return
	}

	if GlobalConfig.Targets == nil {
		GlobalConfig.Targets = make(map[string]TargetConfig)
	}

	// Initialize binary paths
	InitializeBinaryPaths()

	return
}

// SaveConfig saves the config to the config file
func SaveConfig() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, ConfigFileName)

	out, err := yaml.Marshal(GlobalConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, out, 0644)
}

// SetupTargetStorage configures the storage directory for a target
// If the target exists and newStorageDir is provided, it moves existing files
func SetupTargetStorage(targetName, newStorageDir string) error {
	if targetName == "" {
		return fmt.Errorf("target name cannot be empty")
	}

	// Load current config
	LoadConfig()

	// Check if target exists in config
	existingTarget, targetExists := GlobalConfig.Targets[targetName]

	var finalStorageDir string

	if newStorageDir != "" {
		// User provided a new storage directory
		finalStorageDir = newStorageDir

		if targetExists && existingTarget.StorageDir != "" && existingTarget.StorageDir != newStorageDir {
			// Target exists with different storage dir - need to move files
			logger.Info("Moving existing files from %s to %s", existingTarget.StorageDir, newStorageDir)

			if err := files.MoveTargetFiles(existingTarget.StorageDir, newStorageDir); err != nil {
				return fmt.Errorf("failed to move target files: %w", err)
			}
		}
	} else {
		// No storage dir provided, use existing or default
		if targetExists && existingTarget.StorageDir != "" {
			finalStorageDir = existingTarget.StorageDir
		} else {
			// Use default storage directory
			defaultDir, err := GetDefaultTargetStorageDir()
			if err != nil {
				return fmt.Errorf("failed to get default target storage dir: %w", err)
			}
			finalStorageDir = defaultDir
		}
	}

	// Create storage directories if they don't exist
	dbPath := filepath.Join(finalStorageDir, "db")
	filesPath := filepath.Join(finalStorageDir, "files")

	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	if err := os.MkdirAll(filesPath, 0755); err != nil {
		return fmt.Errorf("failed to create files directory: %w", err)
	}

	// Update config
	GlobalConfig.Targets[targetName] = TargetConfig{
		StorageDir: finalStorageDir,
	}

	// Save updated config
	if err := SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Set global variables
	Target = targetName
	StorageDir = finalStorageDir

	return nil
}
