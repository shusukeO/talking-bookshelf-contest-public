package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"sync"

	"talking-bookshelf/backend/internal/model"

	"github.com/gin-gonic/gin"
)

var (
	books     []model.Book
	booksOnce sync.Once
)

func loadBooks() {
	booksOnce.Do(func() {
		data, err := os.ReadFile("data/books.json")
		if err != nil {
			return
		}
		json.Unmarshal(data, &books)
	})
}

func GetBooks() []model.Book {
	loadBooks()
	return books
}

func GetBookByID(id string) *model.Book {
	loadBooks()
	for _, book := range books {
		if book.ID == id {
			return &book
		}
	}
	return nil
}

func HandleGetBooks(c *gin.Context) {
	loadBooks()
	lang := c.Query("lang")

	// Create a copy to avoid mutating the original slice
	sortedBooks := make([]model.Book, len(books))
	copy(sortedBooks, books)

	// Sort by language priority: current language first
	if lang != "" {
		sort.SliceStable(sortedBooks, func(i, j int) bool {
			iMatch := sortedBooks[i].Language == lang
			jMatch := sortedBooks[j].Language == lang
			if iMatch != jMatch {
				return iMatch
			}
			return false
		})
	}

	responses := make([]model.BookResponse, len(sortedBooks))
	for i, book := range sortedBooks {
		responses[i] = book.ToResponse()
	}
	c.JSON(http.StatusOK, responses)
}

func HandleGetBook(c *gin.Context) {
	id := c.Param("id")
	book := GetBookByID(id)
	if book == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}
	c.JSON(http.StatusOK, book.ToResponse())
}
