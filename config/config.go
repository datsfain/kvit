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
