// Package sanitize provides functions to sanitize user-provided content
// to prevent prompt injection attacks.
// Reference: OWASP LLM Prompt Injection Prevention Cheat Sheet
// https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html
package sanitize

import (
	"regexp"
)

// instructionPatterns detects instruction-like content in external data (e.g., book notes).
// 60+ patterns across Japanese, English, Chinese, and Korean covering:
// - Output format/length manipulation
// - Role reassignment
// - Instruction override
// - Output manipulation
// - Hypothetical/roleplay scenarios
// - Developer/debug mode requests
// - Prompt extraction attempts
// - Delimiter injection
// Patterns are omitted from the public repository.
var instructionPatterns = []*regexp.Regexp{
	// TODO: Add your indirect injection sanitization patterns here.
	// Example: regexp.MustCompile(`(?i)ignore.*(previous|all)\s*instructions`),
}

// Notes neutralizes instruction-like patterns by wrapping them in 【】 brackets.
// This prevents indirect prompt injection from external content (e.g., book notes).
// The bracketed content signals to the LLM that this is quoted text, not an instruction.
func Notes(notes string) string {
	result := notes
	for _, pattern := range instructionPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return "【" + match + "】"
		})
	}
	return result
}
