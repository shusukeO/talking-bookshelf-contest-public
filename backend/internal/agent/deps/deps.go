package deps

import (
	"context"

	"talking-bookshelf/backend/internal/model"
)

// LLMClient abstracts LLM API calls for validation/summarization
type LLMClient interface {
	GenerateContent(ctx context.Context, prompt string, temperature float32, maxOutputTokens int32) (string, error)
}

// BookRepository abstracts book data access
type BookRepository interface {
	GetByID(id string) *model.Book
	GetAll() []model.Book
	Search(query string) []model.Book
}
