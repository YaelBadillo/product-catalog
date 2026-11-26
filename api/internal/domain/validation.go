package domain

import "strings"

// FieldError is a broken rule on one field. Field is the JSON name; empty
// means the whole body.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError lists every field that breaks a rule.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		if f.Field == "" {
			parts = append(parts, f.Message)
			continue
		}
		parts = append(parts, f.Field+" "+f.Message)
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// fieldErrors collects broken rules; err returns nil when there are none.
type fieldErrors []FieldError

func (fe *fieldErrors) add(field, message string) {
	*fe = append(*fe, FieldError{Field: field, Message: message})
}

func (fe fieldErrors) err() error {
	if len(fe) == 0 {
		return nil
	}
	return &ValidationError{Fields: fe}
}
