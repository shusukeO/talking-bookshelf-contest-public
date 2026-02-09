package validation

import (
	"context"
	"log"

	"talking-bookshelf/backend/internal/agent/deps"
	"talking-bookshelf/backend/internal/agent/prompt"
	"talking-bookshelf/backend/internal/model"
)

// ResponseCorrector generates corrected responses when validation fails
type ResponseCorrector struct {
	llmClient     deps.LLMClient
	bookRepo      deps.BookRepository
	promptBuilder *prompt.Builder
}

// NewResponseCorrector creates a new ResponseCorrector
func NewResponseCorrector(llmClient deps.LLMClient, bookRepo deps.BookRepository, promptBuilder *prompt.Builder) *ResponseCorrector {
	return &ResponseCorrector{
		llmClient:     llmClient,
		bookRepo:      bookRepo,
		promptBuilder: promptBuilder,
	}
}

// Generate creates a follow-up response when validation fails
// For general queries (no selected book), asks follow-up questions instead of recommending books.
func (c *ResponseCorrector) Generate(ctx context.Context, question string, bookID *string, language string) (string, error) {
	log.Printf("[ResponseCorrector] Generating corrected response for: %s", truncateForLog(question, 50))

	// Get selected book if specified
	var selectedBook *model.Book
	if bookID != nil && *bookID != "" {
		selectedBook = c.bookRepo.GetByID(*bookID)
		if selectedBook != nil {
			log.Printf("[ResponseCorrector] Using selected book: %s", selectedBook.Title)
		}
	}

	// Build correction prompt (no book list for general queries - just asks follow-up questions)
	correctionPrompt := c.promptBuilder.BuildCorrectionPrompt(question, language, selectedBook)

	// Generate corrected response (limit to 256 tokens for concise output)
	result, err := c.llmClient.GenerateContent(ctx, correctionPrompt, 0.2, 256)
	if err != nil {
		log.Printf("[ResponseCorrector] API error: %v", err)
		return getFallbackMessage(language), err
	}

	if result == "" {
		return getFallbackMessage(language), nil
	}

	log.Printf("[ResponseCorrector] Generated: %s", truncateForLog(result, 100))
	return result, nil
}

func getFallbackMessage(language string) string {
	if language == "ja" {
		return prompt.FallbackMessageJa
	}
	return prompt.FallbackMessageEn
}
