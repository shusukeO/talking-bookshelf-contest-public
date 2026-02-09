package prompt

import (
	"fmt"
	"strings"

	"talking-bookshelf/backend/internal/model"
)

// ContextOptions holds options for building message context
type ContextOptions struct {
	Language           string
	SelectedBook       *model.Book
	PreviousBooks      []string // Book IDs that were already recommended
	RecentConversation string   // Recent conversation preserved from previous compaction
}

// BuildMessageContext adds context to a user message
func BuildMessageContext(message string, opts ContextOptions) string {
	result := message

	// Add language instruction
	langInstruction := BuildLanguageInstruction(opts.Language)
	result = langInstruction + "\n\n" + result

	// Add selected book context if provided
	if opts.SelectedBook != nil {
		bookContext := fmt.Sprintf("[選択中の本: 「%s」（%s著）ID: %s]",
			opts.SelectedBook.Title, opts.SelectedBook.Author, opts.SelectedBook.ID)
		result = bookContext + "\n\n" + result
	}

	// Add recent conversation context if available (from previous compaction)
	if opts.RecentConversation != "" {
		result = opts.RecentConversation + "\n\n" + result
	}

	// Add previously recommended books exclusion instruction
	if len(opts.PreviousBooks) > 0 {
		var exclusionNotice string
		bookList := strings.Join(opts.PreviousBooks, ", ")
		if opts.Language == "ja" {
			exclusionNotice = fmt.Sprintf("[重要: 以下の本は既にこの会話で紹介済みです。別の本をおすすめしてください: %s]", bookList)
		} else {
			exclusionNotice = fmt.Sprintf("[IMPORTANT: The following books were already recommended in this conversation. Please recommend different books: %s]", bookList)
		}
		result = exclusionNotice + "\n\n" + result
	}

	return result
}

// BuildLanguageInstruction creates a language instruction for the agent
func BuildLanguageInstruction(language string) string {
	switch language {
	case "ja":
		return "[言語指定: 日本語で回答してください。サジェスチョン(SUGGESTIONS)も日本語で出力してください。日本語の書籍(language: ja)のみを紹介してください。]"
	case "en":
		return "[Language instruction: Please respond in English. Output suggestions (SUGGESTIONS) in English as well. Only recommend English books (language: en).]"
	default:
		return "[Language instruction: Please respond in English. Output suggestions (SUGGESTIONS) in English as well. Only recommend English books (language: en).]"
	}
}

// BuildSelectedBookContext creates a context string for a selected book
func BuildSelectedBookContext(book *model.Book, language string) string {
	if book == nil {
		return ""
	}

	if language == "ja" {
		return fmt.Sprintf(`
【選択中の本（この本について回答すること）】
[book::%s::%s]（%s著）
メモ: %s
`, book.Title, book.ID, book.Author, book.PrivateNotes)
	}

	return fmt.Sprintf(`
[Selected book (respond about this book)]
[book::%s::%s] (by %s)
Notes: %s
`, book.Title, book.ID, book.Author, book.PrivateNotes)
}
