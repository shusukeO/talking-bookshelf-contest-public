package model

type Book struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Author       string `json:"author"`
	ISBN         string `json:"isbn"`
	Cover        string `json:"cover"`
	FinishedAt   string `json:"finished_at"`
	PrivateNotes string `json:"private_notes,omitempty"`
	Link         string `json:"link"`     // [book::タイトル::book-id] format for AI to use directly
	Language     string `json:"language"` // "ja" or "en"
}

type BookResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Author     string `json:"author"`
	ISBN       string `json:"isbn"`
	Cover      string `json:"cover"`
	FinishedAt string `json:"finished_at"`
	Language   string `json:"language"`
}

func (b *Book) ToResponse() BookResponse {
	return BookResponse{
		ID:         b.ID,
		Title:      b.Title,
		Author:     b.Author,
		ISBN:       b.ISBN,
		Cover:      b.Cover,
		FinishedAt: b.FinishedAt,
		Language:   b.Language,
	}
}
