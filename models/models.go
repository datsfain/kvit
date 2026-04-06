package models

import (
	"fmt"
	"time"
)

type Expense struct {
	Date    string
	Store   string
	Product string
	Price   float64
}

func (e Expense) CSVRow() []string {
	return []string{e.Date, e.Store, e.Product, fmt.Sprintf("%.2f", e.Price)}
}

type Definition struct {
	Product  string
	Category string
}

func (d Definition) CSVRow() []string {
	return []string{d.Product, d.Category}
}

// StoreEntry groups products for one store in an add command
type StoreEntry struct {
	Store    string
	Date     string
	Products []ProductPrice
}

type ProductPrice struct {
	Product string
	Price   float64
}

func Today() string {
	return time.Now().Format("2006-01-02")
}
