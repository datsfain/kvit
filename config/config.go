package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	ExpensesFile    = "expenses.csv"
	DefinitionsFile = "definitions.csv"
	ExclusionsFile  = "exclusions.csv"
	ConfigFile      = "kvit.json"
)

func ExpensesPath() string {
	return ExpensesFile
}

func DefinitionsPath() string {
	return DefinitionsFile
}

func ExclusionsPath() string {
	return ExclusionsFile
}

// SyncableFiles returns all CSV files that should be synced
func SyncableFiles() []string {
	return []string{ExpensesFile, DefinitionsFile, ExclusionsFile}
}

// Settings holds persistent configuration
type Settings struct {
	Remote string `json:"remote"` // rclone remote path, e.g. "gdrive:expense-tracker"
}

func LoadSettings() (Settings, error) {
	var s Settings
	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func SaveSettings(s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFile, data, 0644)
}

func GetSetting(key string) (string, error) {
	s, err := LoadSettings()
	if err != nil {
		return "", err
	}
	switch key {
	case "remote":
		return s.Remote, nil
	default:
		return "", fmt.Errorf("unknown setting: %s", key)
	}
}

func SetSetting(key, value string) error {
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	switch key {
	case "remote":
		s.Remote = value
	default:
		return fmt.Errorf("unknown setting: %s (available: remote)", key)
	}
	return SaveSettings(s)
}
