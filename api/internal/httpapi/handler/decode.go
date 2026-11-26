package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// RequestError is a malformed request: invalid JSON, an unknown field, a
// wrong type, or an invalid path or query parameter. Field is the JSON or
// parameter name, empty when the whole body is at fault.
type RequestError struct {
	Field   string
	Message string
}

func (e *RequestError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

const unknownFieldPrefix = "json: unknown field "

// DecodeJSON decodes the request body into v, rejecting unknown fields and
// trailing data, and validates it with validate. Malformed input returns a
// *RequestError, broken rules a *domain.ValidationError, and a body over the
// size limit the *http.MaxBytesError unchanged.
func DecodeJSON(c *gin.Context, v any, validate *validator.Validate) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return decodeError(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			return err
		}
		return &RequestError{Message: "unexpected data after the JSON body"}
	}
	if err := validate.Struct(v); err != nil {
		return translateValidation(err)
	}
	return nil
}

func decodeError(err error) error {
	var (
		maxBytes  *http.MaxBytesError
		syntax    *json.SyntaxError
		typeError *json.UnmarshalTypeError
	)
	switch {
	case errors.As(err, &maxBytes):
		return err
	case errors.Is(err, io.EOF):
		return &RequestError{Message: "request body is required"}
	case errors.As(err, &syntax), errors.Is(err, io.ErrUnexpectedEOF):
		return &RequestError{Message: "malformed JSON"}
	case errors.As(err, &typeError):
		if typeError.Field == "" {
			return &RequestError{Message: "body must be a JSON object"}
		}
		return &RequestError{Field: typeError.Field, Message: fmt.Sprintf("must be a %s", jsonType(typeError.Type.Kind().String()))}
	case strings.HasPrefix(err.Error(), unknownFieldPrefix):
		field := strings.Trim(strings.TrimPrefix(err.Error(), unknownFieldPrefix), `"`)
		return &RequestError{Field: field, Message: "unknown field"}
	default:
		return &RequestError{Message: "malformed JSON"}
	}
}

// jsonType names a Go kind as a JSON type.
func jsonType(kind string) string {
	switch {
	case strings.HasPrefix(kind, "int"), strings.HasPrefix(kind, "uint"):
		return "whole number"
	case strings.HasPrefix(kind, "float"):
		return "number"
	case kind == "bool":
		return "boolean"
	default:
		return kind
	}
}
