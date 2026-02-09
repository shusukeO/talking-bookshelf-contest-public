package agent

import (
	"strings"

	"talking-bookshelf/backend/internal/model"
)

// InMemoryBookRepository is a simple in-memory implementation of BookRepository
type InMemoryBookRepository struct {
	books []model.Book
}

// NewInMemoryBookRepository creates a new InMemoryBookRepository
func NewInMemoryBookRepository(books []model.Book) *InMemoryBookRepository {
	return &InMemoryBookRepository{books: books}
}

// GetByID finds a book by its ID
func (r *InMemoryBookRepository) GetByID(id string) *model.Book {
	for _, book := range r.books {
		if book.ID == id {
			return &book
		}
	}
	return nil
}

// GetAll returns all books
func (r *InMemoryBookRepository) GetAll() []model.Book {
	return r.books
}

// Search finds books matching the query in title, author, or notes
func (r *InMemoryBookRepository) Search(query string) []model.Book {
	lowerQuery := strings.ToLower(query)
	var results []model.Book
	for _, book := range r.books {
		if strings.Contains(strings.ToLower(book.Title), lowerQuery) ||
			strings.Contains(strings.ToLower(book.Author), lowerQuery) ||
			strings.Contains(strings.ToLower(book.PrivateNotes), lowerQuery) {
			results = append(results, book)
		}
	}
	return results
}
