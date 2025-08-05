package gai

import (
	"log"
	"os"
	"path/filepath"
	
	"github.com/joho/godotenv"
)

// LoadEnvFromFile loads environment variables from a specific .env file.
// This is a convenience function for users who want to load environment variables
// before creating the client.
func LoadEnvFromFile(path string) error {
	return godotenv.Load(path)
}

// FindAndLoadEnv searches for a .env file by traversing up from the current working directory
// and loads it if found. This is useful for development environments.
// Returns the path to the loaded file, or an error if no file was found.
func FindAndLoadEnv() (string, error) {
	envPath := findEnvFile()
	if envPath == "" {
		return "", os.ErrNotExist
	}
	
	if err := godotenv.Load(envPath); err != nil {
		return "", err
	}
	
	return envPath, nil
}

// findEnvFile searches for a .env file by traversing up from the current working directory.
// It returns the path to the .env file if found, or an empty string otherwise.
// It limits the search to a reasonable number of parent directories.
func findEnvFile() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: findEnvFile: Error getting current working directory: %v", err)
		return ""
	}

	// Limit search depth to avoid infinite loops or excessive searching
	const maxDepth = 5

	for i := 0; i < maxDepth; i++ {
		envPath := filepath.Join(cwd, ".env")
		if _, err := os.Stat(envPath); err == nil {
			// .env file found
			return envPath
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// Reached the root directory
			break
		}
		cwd = parent
	}

	// .env file not found within the search depth
	return ""
}