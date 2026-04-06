package storage

import (
	"encoding/csv"
	"kvit/config"
	"kvit/models"
	"os"
	"strconv"
)

var expensesHeader = []string{"date", "store", "product", "price"}

// LoadExpenses reads all expenses from CSV
func LoadExpenses() ([]models.Expense, error) {
	f, err := os.Open(config.ExpensesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var expenses []models.Expense
	for i, row := range records {
		if i == 0 { // skip header
			continue
		}
		if len(row) < 4 {
			continue
		}
		price, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			continue
		}
		expenses = append(expenses, models.Expense{
			Date:    row[0],
			Store:   row[1],
			Product: row[2],
			Price:   price,
		})
	}
	return expenses, nil
}

// AppendExpenses adds expenses to the CSV file, creating it with header if needed
func AppendExpenses(expenses []models.Expense) error {
	if err := config.EnsureDataDir(); err != nil {
		return err
	}

	fileExists := true
	if _, err := os.Stat(config.ExpensesPath()); os.IsNotExist(err) {
		fileExists = false
	}

	f, err := os.OpenFile(config.ExpensesPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if !fileExists {
		if err := writer.Write(expensesHeader); err != nil {
			return err
		}
	}

	for _, e := range expenses {
		if err := writer.Write(e.CSVRow()); err != nil {
			return err
		}
	}
	return nil
}

// UniqueStores returns all unique store names from expenses
func UniqueStores() []string {
	expenses, err := LoadExpenses()
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var stores []string
	for _, e := range expenses {
		if !seen[e.Store] {
			seen[e.Store] = true
			stores = append(stores, e.Store)
		}
	}
	return stores
}
