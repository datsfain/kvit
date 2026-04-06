package config

import (
	"os"
	"path/filepath"
)

const (
	DataDir         = "data"
	ExpensesFile    = "expenses.csv"
	DefinitionsFile = "definitions.csv"
	ExclusionsFile  = "exclusions.csv"
)

func ExpensesPath() string {
	return filepath.Join(DataDir, ExpensesFile)
}

func DefinitionsPath() string {
	return filepath.Join(DataDir, DefinitionsFile)
}

func ExclusionsPath() string {
	return filepath.Join(DataDir, ExclusionsFile)
}

// EnsureDataDir creates the data directory if it doesn't exist
func EnsureDataDir() error {
	return os.MkdirAll(DataDir, 0755)
}
