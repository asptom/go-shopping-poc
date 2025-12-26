package csv

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Parser provides generic CSV parsing functionality
type Parser[T any] struct {
	filePath string
}

// NewParser creates a new generic CSV parser
func NewParser[T any](filePath string) *Parser[T] {
	return &Parser[T]{filePath: filePath}
}

// Parse reads and parses the CSV file into a slice of the generic type T
func (p *Parser[T]) Parse() ([]T, error) {
	file, err := os.Open(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Create header index map
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	var results []T
	rowNum := 1 // Header is row 0, data starts at row 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row %d: %w", rowNum, err)
		}

		item, err := p.parseRow(record, headerMap, rowNum)
		if err != nil {
			return nil, fmt.Errorf("failed to parse row %d: %w", rowNum, err)
		}

		results = append(results, item)
		rowNum++
	}

	return results, nil
}

// parseRow parses a single CSV row into the generic type T using reflection
func (p *Parser[T]) parseRow(record []string, headerMap map[string]int, rowNum int) (T, error) {
	var item T
	itemValue := reflect.ValueOf(&item).Elem()
	itemType := itemValue.Type()

	// Iterate through struct fields
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		fieldValue := itemValue.Field(i)

		// Get CSV column name from tag
		csvTag := field.Tag.Get("csv")
		if csvTag == "" || csvTag == "-" {
			continue // Skip fields without csv tag or marked to skip
		}

		// Handle comma-separated alternative column names
		columnNames := strings.Split(csvTag, ",")
		var columnIndex int
		var found bool

		for _, colName := range columnNames {
			colName = strings.ToLower(strings.TrimSpace(colName))
			if idx, exists := headerMap[colName]; exists {
				columnIndex = idx
				found = true
				break
			}
		}

		if !found {
			// Optional field - skip if not found
			if !strings.Contains(csvTag, "optional") {
				return item, fmt.Errorf("required column '%s' not found in CSV header", csvTag)
			}
			continue
		}

		// Check bounds
		if columnIndex >= len(record) {
			return item, fmt.Errorf("column index %d out of bounds for row %d", columnIndex, rowNum)
		}

		cellValue := strings.TrimSpace(record[columnIndex])
		if cellValue == "" {
			// Skip empty values for optional fields
			if strings.Contains(csvTag, "optional") {
				continue
			}
		}

		// Parse the value based on field type
		if err := p.setFieldValue(fieldValue, field.Type, cellValue, rowNum); err != nil {
			return item, fmt.Errorf("failed to parse field '%s': %w", field.Name, err)
		}
	}

	return item, nil
}

// setFieldValue sets the value of a struct field based on its type
func (p *Parser[T]) setFieldValue(fieldValue reflect.Value, fieldType reflect.Type, cellValue string, rowNum int) error {
	if !fieldValue.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	// Handle pointer types
	if fieldType.Kind() == reflect.Ptr {
		if cellValue == "" {
			return nil // Leave nil pointers as nil
		}
		// Create new value for pointer
		elemType := fieldType.Elem()
		elemValue := reflect.New(elemType)
		if err := p.setFieldValue(elemValue.Elem(), elemType, cellValue, rowNum); err != nil {
			return err
		}
		fieldValue.Set(elemValue)
		return nil
	}

	switch fieldType.Kind() {
	case reflect.String:
		fieldValue.SetString(cellValue)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if cellValue == "" {
			fieldValue.SetInt(0)
			return nil
		}
		val, err := strconv.ParseInt(cellValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value '%s': %w", cellValue, err)
		}
		fieldValue.SetInt(val)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if cellValue == "" {
			fieldValue.SetUint(0)
			return nil
		}
		val, err := strconv.ParseUint(cellValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value '%s': %w", cellValue, err)
		}
		fieldValue.SetUint(val)

	case reflect.Float32, reflect.Float64:
		if cellValue == "" {
			fieldValue.SetFloat(0.0)
			return nil
		}
		val, err := strconv.ParseFloat(cellValue, 64)
		if err != nil {
			return fmt.Errorf("invalid float value '%s': %w", cellValue, err)
		}
		fieldValue.SetFloat(val)

	case reflect.Bool:
		if cellValue == "" {
			fieldValue.SetBool(false)
			return nil
		}
		val := strings.ToLower(cellValue) == "true" || cellValue == "1" || strings.ToLower(cellValue) == "yes"
		fieldValue.SetBool(val)

	case reflect.Struct:
		// Special handling for time.Time
		if fieldType == reflect.TypeOf(time.Time{}) {
			if cellValue == "" {
				return nil // Leave zero time as is
			}
			parsedTime, err := time.Parse(time.RFC3339, cellValue)
			if err != nil {
				// Try other common formats
				parsedTime, err = time.Parse("2006-01-02", cellValue)
				if err != nil {
					return fmt.Errorf("invalid time format '%s': %w", cellValue, err)
				}
			}
			fieldValue.Set(reflect.ValueOf(parsedTime))
			return nil
		}
		// For other structs, fall through to JSON parsing

	case reflect.Slice:
		if cellValue == "" {
			return nil // Leave empty slices as nil/empty
		}

		// Try to parse as JSON array first
		elemType := fieldType.Elem()
		sliceValue := reflect.MakeSlice(fieldType, 0, 0)

		if strings.HasPrefix(cellValue, "[") && strings.HasSuffix(cellValue, "]") {
			// JSON array parsing
			slicePtr := reflect.New(fieldType)
			slicePtr.Elem().Set(sliceValue)
			if err := json.Unmarshal([]byte(cellValue), slicePtr.Interface()); err != nil {
				return fmt.Errorf("failed to parse JSON array '%s': %w", cellValue, err)
			}
			sliceValue = slicePtr.Elem()
		} else {
			// Simple comma-separated values
			values := strings.Split(cellValue, ",")
			for _, val := range values {
				val = strings.TrimSpace(val)
				if val == "" {
					continue
				}

				elemValue := reflect.New(elemType).Elem()
				if err := p.setFieldValue(elemValue, elemType, val, rowNum); err != nil {
					return fmt.Errorf("failed to parse slice element '%s': %w", val, err)
				}
				sliceValue = reflect.Append(sliceValue, elemValue)
			}
		}

		fieldValue.Set(sliceValue)

	default:
		// Try JSON parsing for complex types (maps, structs, etc.)
		if cellValue != "" {
			targetPtr := reflect.New(fieldType)
			if err := json.Unmarshal([]byte(cellValue), targetPtr.Interface()); err != nil {
				return fmt.Errorf("failed to parse JSON value '%s' for type %s: %w", cellValue, fieldType.String(), err)
			}
			fieldValue.Set(targetPtr.Elem())
		}
	}

	return nil
}

// ValidateHeaders validates that required columns are present in the CSV header
func (p *Parser[T]) ValidateHeaders() error {
	file, err := os.Open(p.filePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file for header validation: %w", err)
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Create header set
	headerSet := make(map[string]bool)
	for _, header := range headers {
		headerSet[strings.ToLower(strings.TrimSpace(header))] = true
	}

	var item T
	itemType := reflect.TypeOf(item)

	var missingColumns []string

	// Check required fields
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		csvTag := field.Tag.Get("csv")

		if csvTag == "" || csvTag == "-" || strings.Contains(csvTag, "optional") {
			continue // Skip optional fields
		}

		// Check if any of the alternative column names exist
		columnNames := strings.Split(csvTag, ",")
		found := false
		for _, colName := range columnNames {
			colName = strings.ToLower(strings.TrimSpace(colName))
			if headerSet[colName] {
				found = true
				break
			}
		}

		if !found {
			missingColumns = append(missingColumns, csvTag)
		}
	}

	if len(missingColumns) > 0 {
		return fmt.Errorf("missing required columns: %s", strings.Join(missingColumns, ", "))
	}

	return nil
}
