package config

const (
	ExpensesFile    = "expenses.csv"
	DefinitionsFile = "definitions.csv"
	ExclusionsFile  = "exclusions.csv"
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
