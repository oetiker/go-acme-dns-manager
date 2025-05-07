package manager

import (
	"fmt"
	"strings"

	"github.com/kaptinlin/jsonschema"
)

// FormatValidationError converts the detailed JSON Schema validation errors
// to a more concise, user-friendly error message
func FormatValidationError(result *jsonschema.EvaluationResult) error {
	// Convert to a list for easier processing
	list := result.ToList()

	// Store our friendly error messages
	var errorMessages []string

	// Check for unknown fields (additional properties)
	if errMsg, ok := list.Errors["additionalProperties"]; ok && errMsg != "" {
		// Try to extract field names from the error message
		fields := extractUnknownFields(errMsg)
		if len(fields) > 0 {
			errorMessages = append(errorMessages,
				fmt.Sprintf("Unrecognized option(s) in config file: %s",
					strings.Join(fields, ", ")))
		}
	}

	// Check for missing required fields
	if errMsg, ok := list.Errors["required"]; ok && errMsg != "" {
		errorMessages = append(errorMessages,
			fmt.Sprintf("Missing required option(s): %s", errMsg))
	}

	// Process other errors
	processListErrors(list, &errorMessages)

	// If we didn't extract any specific errors, fall back to a generic message
	if len(errorMessages) == 0 {
		return fmt.Errorf("use -debug for details")
	}

	return fmt.Errorf("\n - %s",
		strings.Join(errorMessages, "\n - "))
}

// processListErrors processes errors from the validation list
func processListErrors(list *jsonschema.List, messages *[]string) {
	// Check other errors at this level
	for errorType, errMsg := range list.Errors {
		// Skip additionalProperties and required errors (handled separately)
		if errorType == "additionalProperties" || errorType == "required" {
			continue
		}

		// Process other specific errors
		if errorType == "properties" {
			// Processed through details below
			continue
		}

		// Add other high-level errors
		if errorType != "" && errMsg != "" {
			*messages = append(*messages, fmt.Sprintf("Error: %s", errMsg))
		}
	}

	// Process nested errors in details
	for _, detail := range list.Details {
		// Skip valid entries
		if detail.Valid {
			continue
		}

		// Get field name from the instance location path
		fieldName := getFieldNameFromPath(detail.InstanceLocation)

		// Process specific error for this field
		for _, errMsg := range detail.Errors {
			if fieldName != "" && errMsg != "" {
				// Make messages more user-friendly
				message := errMsg
				if strings.Contains(message, "No values are allowed because the schema is set to 'false'") {
					message = "not a valid configuration option"
				}
				*messages = append(*messages,
					fmt.Sprintf("Problem with option '%s': %s", fieldName, message))
			}
		}

		// Process nested field errors recursively
		for _, nestedDetail := range detail.Details {
			if !nestedDetail.Valid {
				nestedField := getFieldNameFromPath(nestedDetail.InstanceLocation)
				// Process nested field errors
				for _, errMsg := range nestedDetail.Errors {
					if nestedField != "" && errMsg != "" {
						// Make messages more user-friendly
						message := errMsg
						if strings.Contains(message, "No values are allowed because the schema is set to 'false'") {
							message = "not a valid configuration option"
						}

						parentField := getFieldNameFromPath(detail.InstanceLocation)
						if parentField != "" {
							*messages = append(*messages,
								fmt.Sprintf("Problem with option '%s.%s': %s",
									parentField, nestedField, message))
						} else {
							*messages = append(*messages,
								fmt.Sprintf("Problem with option '%s': %s", nestedField, message))
						}
					}
				}
			}
		}
	}
}

// extractUnknownFields extracts field names from an additionalProperties error message
func extractUnknownFields(errMsg string) []string {
	// The format is usually: "Additional properties 'field1', 'field2' do not match the schema"
	if !strings.Contains(errMsg, "Additional properties") {
		return nil
	}

	fieldsText := strings.TrimPrefix(errMsg, "Additional properties ")
	fieldsText = strings.TrimSuffix(fieldsText, " do not match the schema")

	// Extract individual field names
	var fields []string
	for _, field := range strings.Split(fieldsText, ", ") {
		field = strings.Trim(field, "'")
		if field != "" {
			fields = append(fields, field)
		}
	}
	return fields
}

// getFieldNameFromPath extracts a readable field name from a JSON path
func getFieldNameFromPath(path string) string {
	if path == "" {
		return ""
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// For root path or empty path
	if path == "" {
		return ""
	}

	// Get last segment for simple field name
	segments := strings.Split(path, "/")
	return segments[len(segments)-1]
}
