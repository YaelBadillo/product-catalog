package handler

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"

	"github.com/YaelBadillo/product-catalog/api/internal/domain"
)

// NewValidator returns a validator that reports JSON field names and knows
// the custom tags, which delegate to the domain:
//   - sku: the value, normalized, is a valid SKU.
//   - trimmed_len=min-max: the length, after trimming, is within bounds.
func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(jsonName)
	mustRegister(v, "sku", func(fl validator.FieldLevel) bool {
		return domain.ValidateSKU(domain.NormalizeSKU(fl.Field().String())) == nil
	})
	mustRegister(v, "trimmed_len", validTrimmedLen)
	return v
}

func mustRegister(v *validator.Validate, tag string, fn validator.Func) {
	if err := v.RegisterValidation(tag, fn); err != nil {
		panic("register validation " + tag + ": " + err.Error()) // programming error at startup
	}
}

func jsonName(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

func validTrimmedLen(fl validator.FieldLevel) bool {
	minLen, maxLen, ok := parseRange(fl.Param())
	if !ok {
		return false
	}
	n := utf8.RuneCountInString(domain.NormalizeName(fl.Field().String()))
	return n >= minLen && n <= maxLen
}

func parseRange(param string) (minLen, maxLen int, ok bool) {
	lo, hi, found := strings.Cut(param, "-")
	if !found {
		return 0, 0, false
	}
	minLen, errMin := strconv.Atoi(lo)
	maxLen, errMax := strconv.Atoi(hi)
	return minLen, maxLen, errMin == nil && errMax == nil
}

// translateValidation turns validator errors into a *domain.ValidationError
// with the domain's rule message for each field.
func translateValidation(err error) error {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err
	}
	fields := make([]domain.FieldError, 0, len(verrs))
	for _, fe := range verrs {
		fields = append(fields, domain.FieldError{Field: fe.Field(), Message: fieldMessage(fe)})
	}
	return &domain.ValidationError{Fields: fields}
}

func fieldMessage(fe validator.FieldError) string {
	if fe.Tag() == "required" {
		return "is required"
	}
	if msg := domain.RuleMessage(fe.Field()); msg != "" {
		return msg
	}
	return "is invalid"
}
