package storage

import (
	"encoding/csv"
	"fmt"
	"kvit/config"
	"kvit/models"
	"os"
)

var definitionsHeader = []string{"product", "category"}

// LoadDefinitions reads all product->category mappings from CSV
func LoadDefinitions() ([]models.Definition, error) {
	f, err := os.Open(config.DefinitionsPath())
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

	var defs []models.Definition
	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) < 2 {
			continue
		}
		defs = append(defs, models.Definition{
			Product:  row[0],
			Category: row[1],
		})
	}
	return defs, nil
}

// ProductNames returns all known product names from definitions
func ProductNames() []string {
	defs, err := LoadDefinitions()
	if err != nil {
		return nil
	}
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Product
	}
	return names
}

// CategoryNames returns all unique category names
func CategoryNames() []string {
	defs, err := LoadDefinitions()
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var cats []string
	for _, d := range defs {
		if !seen[d.Category] {
			seen[d.Category] = true
			cats = append(cats, d.Category)
		}
	}
	return cats
}

// IsKnownProduct checks if a product exists in definitions
func IsKnownProduct(name string) bool {
	defs, err := LoadDefinitions()
	if err != nil {
		return false
	}
	for _, d := range defs {
		if d.Product == name {
			return true
		}
	}
	return false
}

// LoadExclusions returns a set of product names excluded from the prompt
func LoadExclusions() map[string]bool {
	f, err := os.Open(config.ExclusionsPath())
	if err != nil {
		return nil
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil
	}

	excluded := make(map[string]bool)
	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) >= 1 {
			excluded[row[0]] = true
		}
	}
	return excluded
}

// AddExclusion adds a product to the exclusions list
func AddExclusion(product string) error {
	fileExists := true
	if _, err := os.Stat(config.ExclusionsPath()); os.IsNotExist(err) {
		fileExists = false
	}

	f, err := os.OpenFile(config.ExclusionsPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if !fileExists {
		if err := writer.Write([]string{"product"}); err != nil {
			return err
		}
	}

	return writer.Write([]string{product})
}

// RemoveExclusion removes a product from the exclusions list
func RemoveExclusion(product string) error {
	exclusions := LoadExclusions()
	if exclusions == nil || !exclusions[product] {
		return fmt.Errorf("%s is not excluded", product)
	}

	delete(exclusions, product)

	f, err := os.Create(config.ExclusionsPath())
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if err := writer.Write([]string{"product"}); err != nil {
		return err
	}
	for p := range exclusions {
		if err := writer.Write([]string{p}); err != nil {
			return err
		}
	}
	return nil
}

// AppendDefinitions adds new product->category mappings
func AppendDefinitions(defs []models.Definition) error {
	fileExists := true
	if _, err := os.Stat(config.DefinitionsPath()); os.IsNotExist(err) {
		fileExists = false
	}

	f, err := os.OpenFile(config.DefinitionsPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if !fileExists {
		if err := writer.Write(definitionsHeader); err != nil {
			return err
		}
	}

	for _, d := range defs {
		if err := writer.Write(d.CSVRow()); err != nil {
			return err
		}
	}
	return nil
}
