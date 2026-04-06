package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	ExpensesFile    = "expenses.csv"
	DefinitionsFile = "definitions.csv"
	ExclusionsFile  = "exclusions.csv"
	ColorsFile      = "colors.csv"
)

type KvitConfig struct {
	FolderID  string   `json:"folder_id,omitempty"`
	Currency  string   `json:"currency,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

func ExpensesPath() string {
	return ExpensesFile
}

func DefinitionsPath() string {
	return DefinitionsFile
}

func ExclusionsPath() string {
	return ExclusionsFile
}

func ColorsPath() string {
	return ColorsFile
}

// SyncableFiles returns all CSV files that should be synced
func SyncableFiles() []string {
	return []string{ExpensesFile, DefinitionsFile, ExclusionsFile, ColorsFile}
}

func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	dir := filepath.Join(home, ".config", "kvit")
	os.MkdirAll(dir, 0700)
	return dir
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func Load() KvitConfig {
	var c KvitConfig
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return c
	}
	json.Unmarshal(data, &c)
	return c
}

func Save(c KvitConfig) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}

// IsConfigured returns true if the user has completed initial setup
func IsConfigured() bool {
	return Load().Currency != ""
}

// Currency returns the configured currency code
func Currency() string {
	return Load().Currency
}

// Languages returns the configured receipt languages
func Languages() []string {
	return Load().Languages
}
