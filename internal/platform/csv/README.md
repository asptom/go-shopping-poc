# Generic CSV Parser

This package provides a generic, type-safe CSV parser that can parse CSV files into any Go struct type using reflection and struct tags.

## Features

- **Generic Type Support**: Parse CSV data into any struct type using Go generics
- **Struct Tag Mapping**: Use `csv` struct tags to map CSV columns to struct fields
- **JSON Field Parsing**: Automatically parse JSON arrays and objects in CSV fields
- **Optional Fields**: Mark fields as optional with the `optional` tag
- **Alternative Column Names**: Support multiple possible column names for the same field
- **Type Safety**: Compile-time type checking with generics
- **Error Handling**: Comprehensive error reporting with context

## Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "time"
    "go-shopping-poc/internal/platform/csv"
)

type Product struct {
    ID          int64     `csv:"product_id"`
    Name        string    `csv:"product_name"`
    Price       float64   `csv:"price"`
    InStock     bool      `csv:"in_stock"`
    Tags        []string  `csv:"tags"`
    CreatedAt   time.Time `csv:"created_at"`
}

func main() {
    parser := csv.NewParser[Product]("products.csv")
    products, err := parser.Parse()
    if err != nil {
        panic(err)
    }

    for _, product := range products {
        fmt.Printf("Product: %+v\n", product)
    }
}
```

### CSV File Format

```csv
product_id,product_name,price,in_stock,tags,created_at
1,Widget A,29.99,true,"[""electronics"",""gadget""]",2023-01-01T12:00:00Z
2,Widget B,49.99,false,"[""tool""]",2023-01-02T12:00:00Z
```

### Struct Tag Options

- `csv:"column_name"`: Maps the field to a CSV column
- `csv:"col1,col2,col3"`: Alternative column names (first match wins)
- `csv:"column_name,optional"`: Optional field (won't fail if column is missing)
- `csv:"-"`: Skip field (not parsed from CSV)

### Supported Field Types

- **Primitives**: `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `string`, `bool`
- **Time**: `time.Time` (parses RFC3339 or YYYY-MM-DD formats)
- **Slices**: `[]T` (JSON arrays or comma-separated values)
- **Maps**: `map[string]interface{}` (JSON objects)
- **Pointers**: `*T` (nil if empty, allocated if value present)

### JSON Field Parsing

The parser automatically detects JSON content in fields:

- **JSON Arrays**: `["item1", "item2"]` → `[]string`
- **JSON Objects**: `{"key": "value"}` → `map[string]interface{}`
- **Comma-separated**: `item1,item2,item3` → `[]string` (fallback for non-JSON)

### Header Validation

Validate CSV headers before parsing:

```go
parser := csv.NewParser[Product]("products.csv")
if err := parser.ValidateHeaders(); err != nil {
    fmt.Printf("Invalid headers: %v\n", err)
    return
}

// Now safe to parse
products, err := parser.Parse()
```

### Error Handling

The parser provides detailed error messages:

```go
products, err := parser.Parse()
if err != nil {
    // Errors include:
    // - File not found
    // - Invalid CSV format
    // - Missing required columns
    // - Type conversion errors
    // - JSON parsing errors
    log.Printf("Parse error: %v", err)
}
```

## Migration from Legacy Parser

To migrate from the old product-loader CSV parser:

1. **Move to platform**: `temp/product-loader/internal/csv/parser.go` → `internal/platform/csv/parser.go`
2. **Update imports**: Change import paths to use the new location
3. **Use generics**: Replace `Parser` with `Parser[YourType]`
4. **Add struct tags**: Add `csv` tags to your struct fields
5. **Handle optionals**: Mark optional fields with `,optional`

### Before (Legacy)

```go
// Old parser (specific to Product)
parser := csv.NewParser("products.csv")
products, err := parser.Parse()
```

### After (Generic)

```go
// New generic parser
parser := csv.NewParser[Product]("products.csv")
products, err := parser.Parse()
```

## Testing

Run the test suite:

```bash
go test ./internal/platform/csv/
```

The package includes comprehensive tests covering:
- Basic type parsing
- JSON array/object parsing
- Optional fields
- Error conditions
- Header validation
- Alternative column names