package validation

import (
	"context"
)

// ValidationInput contains all data needed for validation
type ValidationInput struct {
	UserQuestion  string
	Response      string
	BookID        *string
	Language      string
	PreviousBooks []string // Book IDs mentioned in previous conversation (to avoid recommending the same books)
}

// ValidationResult is the outcome of a validation
type ValidationResult struct {
	IsValid   bool
	Reason    string
	Corrected string // Non-empty if correction is available
	NeedsRedo bool   // True if response needs to be regenerated from scratch
}

// OK returns a successful validation result
func OK() ValidationResult {
	return ValidationResult{IsValid: true}
}

// Fail returns a failed validation result
func Fail(reason string) ValidationResult {
	return ValidationResult{IsValid: false, Reason: reason, NeedsRedo: true}
}

// FailWithCorrection returns a failed validation result with a corrected response
func FailWithCorrection(reason, corrected string) ValidationResult {
	return ValidationResult{IsValid: false, Reason: reason, Corrected: corrected}
}

// Validator is the interface for validation rules
type Validator interface {
	// Name returns the validator's name for logging
	Name() string
	// Validate checks the response and returns a validation result
	Validate(ctx context.Context, input ValidationInput) ValidationResult
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}
