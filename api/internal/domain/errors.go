// Package domain holds the product model, its normalization and its rules.
// The rules live here once: the handler's validators and the service both
// call them.
package domain

import "errors"

// Sentinel errors that the HTTP layer maps to status codes.
var (
	ErrNotFound         = errors.New("product not found")
	ErrSKUAlreadyExists = errors.New("sku already exists")
	ErrVersionConflict  = errors.New("version conflict")
)
