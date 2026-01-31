package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName      = "cali"
	credentialsFile    = "credentials.json"
	serviceAccountFile = "service-account.json"
	tokenFile          = "token.json"
	configDirPermMode  = 0o700
)

// GetConfigDir returns the configuration directory path (~/.config/cali)
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", configDirName)
	return configDir, nil
}

// GetCredentialsPath returns the path to the OAuth credentials file
func GetCredentialsPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, credentialsFile), nil
}

// GetServiceAccountPath returns the path to the service account key file
func GetServiceAccountPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, serviceAccountFile), nil
}

// GetTokenPath returns the path to the OAuth token file
func GetTokenPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, tokenFile), nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Check if directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// Create directory with restricted permissions
		if err := os.MkdirAll(configDir, configDirPermMode); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	return nil
}
