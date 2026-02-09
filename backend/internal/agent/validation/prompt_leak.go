package validation

import (
	"context"
	"log"
	"regexp"
	"strings"
)

// PromptLeakValidator validates that responses don't leak system prompts or internal information
type PromptLeakValidator struct {
	// sensitivePatterns are regex patterns that indicate prompt leakage
	sensitivePatterns []*regexp.Regexp
	// sensitiveKeywords are exact keywords that should not appear in responses
	sensitiveKeywords []string
}

// NewPromptLeakValidator creates a new PromptLeakValidator
func NewPromptLeakValidator() *PromptLeakValidator {
	// Regex patterns detecting prompt/internal information leakage in responses.
	// Covers: system prompt markers, role markers, API keys, tool definitions,
	// injection echoing, jailbreak mode, internal notes exposure, config leaks.
	// Patterns are omitted from the public repository.
	patterns := []*regexp.Regexp{
		// TODO: Add your prompt leak detection patterns here.
		// Example: regexp.MustCompile(`(?i)system\s*prompt`),
	}

	// Keywords that should never appear in responses.
	// Covers: role self-identification, internal variable names, backend paths, prompt structure markers.
	// Keywords are omitted from the public repository.
	keywords := []string{
		// TODO: Add your sensitive keywords here.
		// Example: "system prompt", "backend/internal",
	}

	return &PromptLeakValidator{
		sensitivePatterns: patterns,
		sensitiveKeywords: keywords,
	}
}

// Name returns the validator name
func (v *PromptLeakValidator) Name() string {
	return "PromptLeakValidator"
}

// Validate checks if the response contains leaked system prompts or internal information
func (v *PromptLeakValidator) Validate(ctx context.Context, input ValidationInput) ValidationResult {
	response := input.Response
	responseLower := strings.ToLower(response)

	// Check for sensitive patterns
	for _, pattern := range v.sensitivePatterns {
		if pattern.MatchString(response) {
			match := pattern.FindString(response)
			log.Printf("[%s] LEAK DETECTED (pattern): %s", v.Name(), truncateForLog(match, 50))
			return Fail("potential system prompt leak detected")
		}
	}

	// Check for sensitive keywords
	for _, keyword := range v.sensitiveKeywords {
		if strings.Contains(responseLower, strings.ToLower(keyword)) {
			log.Printf("[%s] LEAK DETECTED (keyword): %s", v.Name(), keyword)
			return Fail("potential internal information leak detected")
		}
	}

	// Check for suspicious repetition of user input (prompt injection echo)
	if containsPromptInjectionEcho(input.UserQuestion, response) {
		log.Printf("[%s] LEAK DETECTED: prompt injection echo", v.Name())
		return Fail("prompt injection attempt echoed in response")
	}

	log.Printf("[%s] No leaks detected", v.Name())
	return OK()
}

// containsPromptInjectionEcho checks if the response echoes suspicious parts of user input.
// Maintains a list of injection phrases in Japanese, English, Chinese, and Korean.
// If a phrase appears in both the user question and the response, it's flagged as suspicious.
// Phrases are omitted from the public repository.
func containsPromptInjectionEcho(userQuestion, response string) bool {
	injectionPhrases := []string{
		// TODO: Add injection phrases that should not be echoed in responses.
		// Example: "ignore previous", "jailbreak", "脱獄",
	}

	questionLower := strings.ToLower(userQuestion)
	responseLower := strings.ToLower(response)

	for _, phrase := range injectionPhrases {
		phraseLower := strings.ToLower(phrase)
		if strings.Contains(questionLower, phraseLower) && strings.Contains(responseLower, phraseLower) {
			return true
		}
	}

	return false
}
