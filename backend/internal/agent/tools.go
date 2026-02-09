package agent

import (
	"log"
	"strings"

	"talking-bookshelf/backend/internal/agent/sanitize"
	"talking-bookshelf/backend/internal/model"
	"talking-bookshelf/backend/internal/portfolio"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// ============================================
// Tool Input/Output Types
// ============================================

// search_books tool
type searchBooksInput struct {
	Query string `json:"query" jsonschema:"検索キーワード（タイトル、著者、メモから検索）"`
}

type bookSummary struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Author       string `json:"author"`
	Link         string `json:"link"` // [book:タイトル:book-id] format for AI to use directly
	NotesExcerpt string `json:"notes_excerpt"`
}

type searchBooksOutput struct {
	Books []bookSummary `json:"books"`
	Count int           `json:"count"`
}

// get_book_details tool
type getBookDetailsInput struct {
	BookID string `json:"book_id" jsonschema:"本のID（例: book-001）"`
}

type getBookDetailsOutput struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	Link       string `json:"link"` // [book:タイトル:book-id] format for AI to use directly
	FinishedAt string `json:"finished_at"`
	Notes      string `json:"notes"`
	Error      string `json:"error,omitempty"`
}

// get_reading_stats tool (no input needed)
type getReadingStatsOutput struct {
	TotalBooks   int            `json:"total_books"`
	BooksPerYear map[string]int `json:"books_per_year"`
	TopAuthors   []authorCount  `json:"top_authors"`
}

type authorCount struct {
	Author string `json:"author"`
	Count  int    `json:"count"`
}

// ============================================
// get_owner_info tool
// ============================================

type getOwnerInfoOutput struct {
	Name     string        `json:"name"`
	Title    string        `json:"title"`
	Projects []projectInfo `json:"projects"`
	Skills   skillsInfo    `json:"skills"`
	Social   []socialInfo  `json:"social"`
}

type projectInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Link        string `json:"link,omitempty"`
}

type skillsInfo struct {
	Backend  []string `json:"backend"`
	Frontend []string `json:"frontend"`
}

type socialInfo struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

// ============================================
// BookshelfTools - holds the book and portfolio data
// ============================================

type BookshelfTools struct {
	books     []model.Book
	portfolio *portfolio.Portfolio
}

func NewBookshelfTools(books []model.Book, p *portfolio.Portfolio) *BookshelfTools {
	return &BookshelfTools{books: books, portfolio: p}
}

// ============================================
// Tool Handlers
// ============================================

func (t *BookshelfTools) searchBooks(ctx tool.Context, input searchBooksInput) (searchBooksOutput, error) {
	log.Printf("[TOOL] search_books called with query: %s", input.Query)
	query := strings.ToLower(input.Query)
	var results []bookSummary

	for _, book := range t.books {
		if strings.Contains(strings.ToLower(book.Title), query) ||
			strings.Contains(strings.ToLower(book.Author), query) ||
			strings.Contains(strings.ToLower(book.PrivateNotes), query) {
			// メモの抜粋を作成（最大200文字、rune単位で切る）
			notesExcerpt := book.PrivateNotes
			runes := []rune(notesExcerpt)
			if len(runes) > 200 {
				notesExcerpt = string(runes[:200]) + "..."
			}
			notesExcerpt = "<private_notes>" + sanitize.Notes(notesExcerpt) + "</private_notes>"
			results = append(results, bookSummary{
				ID:           book.ID,
				Title:        book.Title,
				Author:       book.Author,
				Link:         book.Link,
				NotesExcerpt: notesExcerpt,
			})
		}
	}

	log.Printf("[TOOL] search_books found %d results", len(results))
	return searchBooksOutput{Books: results, Count: len(results)}, nil
}

func (t *BookshelfTools) getBookDetails(ctx tool.Context, input getBookDetailsInput) (getBookDetailsOutput, error) {
	log.Printf("[TOOL] get_book_details called with book_id: %s", input.BookID)
	for _, book := range t.books {
		if book.ID == input.BookID {
			log.Printf("[TOOL] get_book_details found: %s", book.Title)
			return getBookDetailsOutput{
				ID:         book.ID,
				Title:      book.Title,
				Author:     book.Author,
				Link:       book.Link,
				FinishedAt: book.FinishedAt,
				Notes:      "<private_notes>" + sanitize.Notes(book.PrivateNotes) + "</private_notes>",
			}, nil
		}
	}
	log.Printf("[TOOL] get_book_details: book not found")
	return getBookDetailsOutput{Error: "本が見つかりません"}, nil
}

// Empty input struct for tools with no parameters
type emptyInput struct{}

func (t *BookshelfTools) getReadingStats(ctx tool.Context, _ emptyInput) (getReadingStatsOutput, error) {
	log.Printf("[TOOL] get_reading_stats called")
	yearCount := make(map[string]int)
	authorCountMap := make(map[string]int)

	for _, book := range t.books {
		// Count by year
		if len(book.FinishedAt) >= 4 {
			year := book.FinishedAt[:4]
			yearCount[year]++
		}
		// Count by author
		authorCountMap[book.Author]++
	}

	// Get top authors (sort by count, take top 5)
	var topAuthors []authorCount
	for author, count := range authorCountMap {
		topAuthors = append(topAuthors, authorCount{Author: author, Count: count})
	}
	// Simple sort (bubble sort for small data)
	for i := 0; i < len(topAuthors); i++ {
		for j := i + 1; j < len(topAuthors); j++ {
			if topAuthors[j].Count > topAuthors[i].Count {
				topAuthors[i], topAuthors[j] = topAuthors[j], topAuthors[i]
			}
		}
	}
	if len(topAuthors) > 5 {
		topAuthors = topAuthors[:5]
	}

	result := getReadingStatsOutput{
		TotalBooks:   len(t.books),
		BooksPerYear: yearCount,
		TopAuthors:   topAuthors,
	}
	log.Printf("[TOOL] get_reading_stats returning: %d total books", result.TotalBooks)
	return result, nil
}

func (t *BookshelfTools) getOwnerInfo(ctx tool.Context, _ emptyInput) (getOwnerInfoOutput, error) {
	log.Printf("[TOOL] get_owner_info called")
	if t.portfolio == nil {
		return getOwnerInfoOutput{}, nil
	}

	p := t.portfolio

	// Projects
	var projects []projectInfo
	for _, proj := range p.Projects {
		projects = append(projects, projectInfo{
			Name:        sanitize.Notes(proj.Name),
			Description: sanitize.Notes(proj.Description),
			Link:        proj.Link, // Links are not sanitized (URLs don't contain instructions)
		})
	}

	// Social (top 3 only)
	var social []socialInfo
	for i, s := range p.Social {
		if i >= 3 {
			break
		}
		social = append(social, socialInfo{
			Name: sanitize.Notes(s.Name),
			Link: s.Link,
		})
	}

	return getOwnerInfoOutput{
		Name:     sanitize.Notes(p.About.Name),
		Title:    sanitize.Notes(p.About.Title),
		Projects: projects,
		Skills: skillsInfo{
			Backend:  p.Skills.Backend,  // String slices (low risk)
			Frontend: p.Skills.Frontend,
		},
		Social: social,
	}, nil
}

// ============================================
// BuildTools - creates ADK tools from handlers
// ============================================

func (t *BookshelfTools) BuildTools() ([]tool.Tool, error) {
	searchTool, err := functiontool.New(functiontool.Config{
		Name:        "search_books",
		Description: "本を検索（タイトル、著者、キーワード）",
	}, t.searchBooks)
	if err != nil {
		return nil, err
	}

	detailsTool, err := functiontool.New(functiontool.Config{
		Name:        "get_book_details",
		Description: "本のメモを取得。notesの内容だけを使って回答",
	}, t.getBookDetails)
	if err != nil {
		return nil, err
	}

	statsTool, err := functiontool.New(functiontool.Config{
		Name:        "get_reading_stats",
		Description: "読書統計を取得",
	}, t.getReadingStats)
	if err != nil {
		return nil, err
	}

	ownerTool, err := functiontool.New(functiontool.Config{
		Name:        "get_owner_info",
		Description: "持ち主の情報を取得",
	}, t.getOwnerInfo)
	if err != nil {
		return nil, err
	}

	return []tool.Tool{searchTool, detailsTool, statsTool, ownerTool}, nil
}
