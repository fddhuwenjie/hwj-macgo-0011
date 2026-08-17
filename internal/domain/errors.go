package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrInvalidStatus      = errors.New("invalid status transition")
	ErrInvalidParameter   = errors.New("invalid parameter")
	ErrVersionConflict    = errors.New("version conflict")
	ErrImmutableViolation = errors.New("immutable violation")
	ErrBudgetExceeded     = errors.New("resource budget exceeded")
	ErrLeaseExpired       = errors.New("lease expired")
	ErrLeaseGeneration    = errors.New("lease generation mismatch")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	ErrLineageCycle       = errors.New("lineage cycle detected")
	ErrTagDrift           = errors.New("release tag drift")
)
