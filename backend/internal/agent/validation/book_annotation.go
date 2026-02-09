package validation

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"talking-bookshelf/backend/internal/agent/deps"
)

var bookAnnotationRegex = regexp.MustCompile(`\[book::(.+?)::([^\]]+)\]`)

// BookAnnotation represents a parsed book annotation from the response
type BookAnnotation struct {
	Title  string
	BookID string
}

// BookAnnotationValidator validates [book::title::id] annotations in responses
type BookAnnotationValidator struct {
	bookRepo deps.BookRepository
}

// NewBookAnnotationValidator creates a new BookAnnotationValidator
func NewBookAnnotationValidator(bookRepo deps.BookRepository) *BookAnnotationValidator {
	return &BookAnnotationValidator{bookRepo: bookRepo}
}

// Name returns the validator name
func (v *BookAnnotationValidator) Name() string {
	return "BookAnnotationValidator"
}

// Validate checks if all book annotations in the response are valid
func (v *BookAnnotationValidator) Validate(ctx context.Context, input ValidationInput) ValidationResult {
	annotations := ExtractBookAnnotations(input.Response)

	if len(annotations) == 0 {
		// No book annotation found in response
		// If a specific book was selected but not mentioned, need regeneration
		if input.BookID != nil && *input.BookID != "" {
			book := v.bookRepo.GetByID(*input.BookID)
			if book != nil {
				log.Printf("[%s] Selected book '%s' not mentioned in response", v.Name(), book.Title)
				return Fail(fmt.Sprintf("selected book '%s' not mentioned", book.Title))
			}
		}
		return OK()
	}

	log.Printf("[%s] Found %d book annotation(s) to validate", v.Name(), len(annotations))

	// Validate each annotation
	for _, ann := range annotations {
		log.Printf("[%s] Checking annotation: [book::%s::%s]", v.Name(), ann.Title, ann.BookID)

		// Check if book ID exists
		book := v.bookRepo.GetByID(ann.BookID)
		if book == nil {
			log.Printf("[%s] HALLUCINATION: book ID '%s' does not exist", v.Name(), ann.BookID)
			return Fail(fmt.Sprintf("book ID '%s' does not exist", ann.BookID))
		}

		// Check if title matches the actual book
		if book.Title != ann.Title {
			log.Printf("[%s] TITLE MISMATCH: response claims '%s' but %s is actually '%s'",
				v.Name(), ann.Title, ann.BookID, book.Title)
			return Fail(fmt.Sprintf("title mismatch for %s: expected '%s', got '%s'",
				ann.BookID, book.Title, ann.Title))
		}

		log.Printf("[%s] Book annotation valid: '%s' (%s)", v.Name(), ann.Title, ann.BookID)
	}

	return OK()
}

// ExtractBookAnnotations extracts all [book::title::id] annotations from text
func ExtractBookAnnotations(text string) []BookAnnotation {
	matches := bookAnnotationRegex.FindAllStringSubmatch(text, -1)
	var annotations []BookAnnotation

	for _, match := range matches {
		if len(match) > 2 {
			annotations = append(annotations, BookAnnotation{
				Title:  match[1],
				BookID: match[2],
			})
		}
	}

	return annotations
}

// CollectNotesContext collects notes from all mentioned books for content validation
func (v *BookAnnotationValidator) CollectNotesContext(response string) (notesContext string, bookFormats string, foundBooks []string) {
	annotations := ExtractBookAnnotations(response)

	var notesBuilder strings.Builder
	var formatsBuilder strings.Builder

	for _, ann := range annotations {
		book := v.bookRepo.GetByID(ann.BookID)
		if book != nil {
			notesBuilder.WriteString(fmt.Sprintf("<book>【%s のメモ】\n<notes>%s</notes></book>\n\n", book.Title, book.PrivateNotes))
			formatsBuilder.WriteString(fmt.Sprintf("- [book::%s::%s]\n", ann.Title, ann.BookID))
			foundBooks = append(foundBooks, book.Title)
		}
	}

	return notesBuilder.String(), formatsBuilder.String(), foundBooks
}
