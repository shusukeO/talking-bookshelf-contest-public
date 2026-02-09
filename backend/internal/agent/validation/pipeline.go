package validation

import (
	"context"
	"log"
)

// Pipeline runs multiple validators in sequence
type Pipeline struct {
	validators []Validator
	corrector  *ResponseCorrector
}

// NewPipeline creates a new validation pipeline
func NewPipeline(validators []Validator, corrector *ResponseCorrector) *Pipeline {
	return &Pipeline{
		validators: validators,
		corrector:  corrector,
	}
}

// Validate runs all validators and returns the final (possibly corrected) response
func (p *Pipeline) Validate(ctx context.Context, input ValidationInput) (string, error) {
	log.Printf("[Pipeline] Starting validation for response: %s", truncateForLog(input.Response, 100))

	for _, v := range p.validators {
		result := v.Validate(ctx, input)

		if result.IsValid {
			log.Printf("[Pipeline] %s: PASS", v.Name())
			continue
		}

		log.Printf("[Pipeline] %s: FAIL - %s", v.Name(), result.Reason)

		// If correction is available, use it
		if result.Corrected != "" {
			log.Printf("[Pipeline] Using corrected response from %s", v.Name())
			return result.Corrected, nil
		}

		// If regeneration is needed, use the corrector
		if result.NeedsRedo {
			log.Printf("[Pipeline] Generating new response due to %s failure", v.Name())
			return p.corrector.Generate(ctx, input.UserQuestion, input.BookID, input.Language)
		}
	}

	log.Printf("[Pipeline] All validators passed, using original response")
	return input.Response, nil
}

