package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// HasField checks if a field exists in a struct or pointer to struct
func HasField(input interface{}, field string) (err error) {
	val := reflect.ValueOf(input)

	// Ensure we're dealing with a pointer or struct
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("input is not a struct or pointer to struct")
	}

	// Handle nested fields using dot notation
	if strings.Contains(field, ".") {
		parts := strings.SplitN(field, ".", 2) // Split into two parts
		subField := val.FieldByName(parts[0])
		if !subField.IsValid() {
			return fmt.Errorf("missing required field: %s", parts[0])
		}
		return HasField(subField.Interface(), parts[1]) // Recursive call for nested fields
	}

	// Check if the field exists
	if !val.FieldByName(field).IsValid() {
		return fmt.Errorf("missing required field: %s", field)
	}

	return nil
}
