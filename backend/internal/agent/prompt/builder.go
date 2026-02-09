package prompt

import (
	"fmt"
	"strings"

	"talking-bookshelf/backend/internal/model"
)

// Builder constructs prompts for the agent
type Builder struct{}

// NewBuilder creates a new prompt builder
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildSystemPrompt creates the minimal system prompt for Gemini Flash
// Portfolio info is now fetched via get_owner_info tool
func (b *Builder) BuildSystemPrompt() string {
	return SystemPromptFlash
}

// BuildValidationPrompt creates a prompt to validate a response against notes
func (b *Builder) BuildValidationPrompt(notesContext, response, bookFormats string) string {
	return fmt.Sprintf(ValidationPromptJa, notesContext, response, bookFormats)
}

// BuildCorrectionPrompt creates a prompt to generate a corrected response
// For general queries (no selected book), this generates a follow-up question instead of recommending books.
func (b *Builder) BuildCorrectionPrompt(question string, language string, selectedBook *model.Book) string {
	if selectedBook != nil {
		var bookContext string
		if language == "ja" {
			bookContext = fmt.Sprintf("[book::%s::%s]（%s著）\nメモ: %s",
				selectedBook.Title, selectedBook.ID, selectedBook.Author, selectedBook.PrivateNotes)
			return fmt.Sprintf(CorrectionPromptJaWithBook, bookContext, question)
		}
		bookContext = fmt.Sprintf("[book::%s::%s] (by %s)\nNotes: %s",
			selectedBook.Title, selectedBook.ID, selectedBook.Author, selectedBook.PrivateNotes)
		return fmt.Sprintf(CorrectionPromptEnWithBook, bookContext, question)
	}

	// General query: ask follow-up questions instead of recommending books
	if language == "ja" {
		return fmt.Sprintf(CorrectionPromptJaGeneral, question)
	}
	return fmt.Sprintf(CorrectionPromptEnGeneral, question)
}

// BuildBookListForCorrection creates a formatted book list for correction prompts
// excludeIDs contains book IDs to exclude from the list (e.g., previously recommended books)
func BuildBookListForCorrection(books []model.Book, excludeIDs []string) string {
	// Build exclude set for O(1) lookup
	excludeSet := make(map[string]bool, len(excludeIDs))
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}

	var sb strings.Builder
	for _, book := range books {
		if excludeSet[book.ID] {
			continue // Skip excluded books
		}
		sb.WriteString(fmt.Sprintf("- [book::%s::%s]: %s\n",
			book.Title, book.ID, truncateString(book.PrivateNotes, 100)))
	}
	return sb.String()
}

// truncateString truncates a string to maxLen runes
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
}
