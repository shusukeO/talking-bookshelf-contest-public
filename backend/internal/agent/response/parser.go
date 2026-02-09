package response

import (
	"regexp"
	"strings"
)

// ChatResponse is the parsed response from the agent
type ChatResponse struct {
	Response    string   `json:"response"`
	Emotion     string   `json:"emotion"`
	Suggestions []string `json:"suggestions"`
}

const defaultEmotion = "talking"

var (
	emotionRegex     = regexp.MustCompile(`\[EMOTION:(idle|thinking|talking|surprised|greeting)\]`)
	suggestionsRegex = regexp.MustCompile(`\[SUGGESTIONS:([^\]]+)\]`)
)

// Parse extracts emotion and suggestions from the raw response text
func Parse(text string) *ChatResponse {
	emotion := extractEmotion(text)
	suggestions := extractSuggestions(text)
	responseText := cleanResponse(text)

	return &ChatResponse{
		Response:    responseText,
		Emotion:     emotion,
		Suggestions: suggestions,
	}
}

// extractEmotion extracts the emotion from [EMOTION:xxx] format
func extractEmotion(text string) string {
	matches := emotionRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return defaultEmotion
}

// extractSuggestions extracts suggestions from [SUGGESTIONS:a|b|c] format
func extractSuggestions(text string) []string {
	matches := suggestionsRegex.FindStringSubmatch(text)
	if len(matches) <= 1 {
		return nil
	}

	var suggestions []string
	parts := strings.Split(matches[1], "|")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			suggestions = append(suggestions, trimmed)
		}
	}
	return suggestions
}

// cleanResponse removes emotion and suggestion markers from the response
func cleanResponse(text string) string {
	result := emotionRegex.ReplaceAllString(text, "")
	result = suggestionsRegex.ReplaceAllString(result, "")
	return strings.TrimSpace(result)
}
