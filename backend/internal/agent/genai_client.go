package agent

import (
	"context"

	"google.golang.org/genai"
)

// GeminiLLMClient implements LLMClient using the Gemini API
type GeminiLLMClient struct {
	client *genai.Client
	model  string
}

// NewGeminiLLMClient creates a new GeminiLLMClient
func NewGeminiLLMClient(client *genai.Client, model string) *GeminiLLMClient {
	return &GeminiLLMClient{
		client: client,
		model:  model,
	}
}

// GenerateContent generates content using the Gemini API
func (c *GeminiLLMClient) GenerateContent(ctx context.Context, prompt string, temperature float32, maxOutputTokens int32) (string, error) {
	config := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(temperature),
		MaxOutputTokens: maxOutputTokens,
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.model, []*genai.Content{
		{
			Role:  "user",
			Parts: []*genai.Part{{Text: prompt}},
		},
	}, config)
	if err != nil {
		return "", err
	}

	// Extract text from response
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				return part.Text, nil
			}
		}
	}

	return "", nil
}
