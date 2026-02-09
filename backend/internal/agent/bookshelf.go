package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"

	"talking-bookshelf/backend/internal/agent/deps"
	"talking-bookshelf/backend/internal/agent/prompt"
	"talking-bookshelf/backend/internal/agent/response"
	"talking-bookshelf/backend/internal/agent/validation"
	"talking-bookshelf/backend/internal/model"
	"talking-bookshelf/backend/internal/portfolio"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

const (
	// DefaultModel is the Gemini model to use for the main agent
	DefaultModel = "gemini-2.5-flash"
	// ValidationModel is the Gemini model to use for validation (faster, cheaper)
	ValidationModel = "gemini-2.5-flash-lite"
	// RecentTurnsToKeep is the number of recent turns to preserve (3 turns = 6 events)
	RecentTurnsToKeep = 3
	// RecentConversationStateKey is the key for storing recent conversation in session state
	RecentConversationStateKey = "recent_conversation"
)

// ChatResponse is the parsed response from the agent (re-exported for handler compatibility)
type ChatResponse = response.ChatResponse

// BookshelfAgent wraps the ADK agent and runner
type BookshelfAgent struct {
	runner           *runner.Runner
	sessionService   session.Service
	genaiClient      *genai.Client
	bookRepo         deps.BookRepository
	portfolio        *portfolio.Portfolio
	promptBuilder    *prompt.Builder
	pipeline         *validation.Pipeline
	mu               sync.Mutex
	recommendedBooks map[string][]string // sessionID -> recommended book IDs
}

// NewBookshelfAgent creates a new ADK-based bookshelf agent
func NewBookshelfAgent(ctx context.Context, books []model.Book, p *portfolio.Portfolio) (*BookshelfAgent, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	// Create genai client for summarization and validation
	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	// Create Gemini model for ADK
	geminiModel, err := gemini.NewModel(ctx, DefaultModel, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini model: %w", err)
	}

	// Build tools (with portfolio for get_owner_info)
	toolBuilder := NewBookshelfTools(books, p)
	tools, err := toolBuilder.BuildTools()
	if err != nil {
		return nil, fmt.Errorf("failed to build tools: %w", err)
	}

	// Create prompt builder and system prompt (portfolio removed - now a tool)
	promptBuilder := prompt.NewBuilder()
	systemPrompt := promptBuilder.BuildSystemPrompt()

	// Create LLM agent
	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        "talking_bookshelf",
		Model:       geminiModel,
		Description: "A talking bookshelf that represents the owner's reading experience and portfolio.",
		Instruction: systemPrompt,
		Tools:       tools,
		GenerateContentConfig: &genai.GenerateContentConfig{
			Temperature:     genai.Ptr[float32](0.2),
			MaxOutputTokens: 2048,
			TopK:            genai.Ptr[float32](64),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM agent: %w", err)
	}

	// Create session service
	sessionService := session.InMemoryService()

	// Create runner
	r, err := runner.New(runner.Config{
		AppName:        "talking_bookshelf",
		Agent:          llmAgent,
		SessionService: sessionService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	// Create book repository and LLM client
	bookRepo := NewInMemoryBookRepository(books)
	llmClient := NewGeminiLLMClient(genaiClient, ValidationModel)

	// Create validation pipeline
	corrector := validation.NewResponseCorrector(llmClient, bookRepo, promptBuilder)
	pipeline := validation.NewPipeline(
		[]validation.Validator{
			validation.NewPromptLeakValidator(),             // First: check for prompt leaks
			validation.NewBookAnnotationValidator(bookRepo), // Second: validate book annotations
		},
		corrector,
	)

	return &BookshelfAgent{
		runner:           r,
		sessionService:   sessionService,
		genaiClient:      genaiClient,
		bookRepo:         bookRepo,
		portfolio:        p,
		promptBuilder:    promptBuilder,
		pipeline:         pipeline,
		recommendedBooks: make(map[string][]string),
	}, nil
}

// Chat processes a user message and returns the agent's response
func (a *BookshelfAgent) Chat(ctx context.Context, userID, sessionID, message string, bookID *string, language string) (*ChatResponse, error) {
	// Check and compact history if needed
	newSessionID, err := a.compactSessionHistory(ctx, userID, sessionID)
	if err != nil {
		log.Printf("[HISTORY] Warning: Failed to compact history: %v", err)
		newSessionID = sessionID
	}

	// Get recent conversation from session state (if any, from previous compaction)
	recentConversation, _ := a.getRecentConversation(ctx, userID, newSessionID)
	if recentConversation != "" {
		log.Printf("[HISTORY] Including recent conversation context")
	}

	// Get previously recommended books from internal map (BEFORE running agent)
	a.mu.Lock()
	previousBooks := a.recommendedBooks[newSessionID]
	a.mu.Unlock()
	if len(previousBooks) > 0 {
		log.Printf("[BOOKS] Previously recommended books: %v", previousBooks)
	}

	// Build message context
	var selectedBook *model.Book
	if bookID != nil && *bookID != "" {
		selectedBook = a.bookRepo.GetByID(*bookID)
		if selectedBook != nil {
			log.Printf("[CONTEXT] Including selected book: %s", selectedBook.Title)
		}
	}

	messageWithContext := prompt.BuildMessageContext(message, prompt.ContextOptions{
		Language:           language,
		SelectedBook:       selectedBook,
		PreviousBooks:      previousBooks,
		RecentConversation: recentConversation,
	})

	// Create user message
	userMessage := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: messageWithContext},
		},
	}

	// Run config (non-streaming)
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	}

	// Collect response
	var responseText string
	for event, err := range a.runner.Run(ctx, userID, newSessionID, userMessage, runConfig) {
		if err != nil {
			return nil, fmt.Errorf("agent run error: %w", err)
		}

		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					responseText += part.Text
				}
			}
		}
	}

	if responseText == "" {
		return nil, fmt.Errorf("no response from agent")
	}

	log.Printf("[DEBUG] Raw agent response: %s", responseText)

	// Parse response
	parsed := response.Parse(responseText)

	// Validate response through pipeline
	validatedResponse, err := a.pipeline.Validate(ctx, validation.ValidationInput{
		UserQuestion:  message,
		Response:      parsed.Response,
		BookID:        bookID,
		Language:      language,
		PreviousBooks: previousBooks,
	})
	if err != nil {
		log.Printf("[VALIDATE] Warning: Validation failed: %v", err)
		validatedResponse = parsed.Response
	}

	// Extract book IDs from the VALIDATED response and save to internal map
	newBookIDs := extractBookIDsFromText(validatedResponse)
	if len(newBookIDs) > 0 {
		log.Printf("[BOOKS] Books in validated response: %v", newBookIDs)
		allBooks := append(previousBooks, newBookIDs...)
		// Deduplicate
		allBooks = deduplicateStrings(allBooks)
		// Save to internal map (session state doesn't persist with ADK InMemoryService)
		a.mu.Lock()
		a.recommendedBooks[newSessionID] = allBooks
		a.mu.Unlock()
		log.Printf("[BOOKS] Saved recommended books to internal map: %v", allBooks)
	}

	// Clean any remaining tags from validatedResponse (safety measure)
	cleanedResponse := response.Parse(validatedResponse)

	return &ChatResponse{
		Response:    cleanedResponse.Response,
		Emotion:     parsed.Emotion,
		Suggestions: parsed.Suggestions,
	}, nil
}

// CreateSession creates a new session for a user
func (a *BookshelfAgent) CreateSession(ctx context.Context, userID string) (string, error) {
	resp, err := a.sessionService.Create(ctx, &session.CreateRequest{
		AppName: "talking_bookshelf",
		UserID:  userID,
	})
	if err != nil {
		return "", err
	}
	return resp.Session.ID(), nil
}

// getRecentConversation retrieves recent conversation from session state
func (a *BookshelfAgent) getRecentConversation(ctx context.Context, userID, sessionID string) (string, error) {
	getResp, err := a.sessionService.Get(ctx, &session.GetRequest{
		AppName:   "talking_bookshelf",
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		return "", err
	}

	recentConv, err := getResp.Session.State().Get(RecentConversationStateKey)
	if err != nil {
		return "", nil
	}

	if s, ok := recentConv.(string); ok {
		return s, nil
	}
	return "", nil
}

// compactSessionHistory checks if history needs compaction and creates a new session
// Preserves recent conversation (last 5 turns) in session state
func (a *BookshelfAgent) compactSessionHistory(ctx context.Context, userID, sessionID string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	getResp, err := a.sessionService.Get(ctx, &session.GetRequest{
		AppName:   "talking_bookshelf",
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		return sessionID, fmt.Errorf("failed to get session: %w", err)
	}

	sess := getResp.Session
	eventCount := sess.Events().Len()
	maxEvents := RecentTurnsToKeep * 2 // 5 turns = 10 events

	log.Printf("[HISTORY] Session %s has %d events (max: %d)", sessionID, eventCount, maxEvents)

	if eventCount < maxEvents {
		return sessionID, nil
	}

	log.Printf("[HISTORY] Compacting session %s (keeping recent %d turns)...", sessionID, RecentTurnsToKeep)

	// Extract recent conversation (last 5 turns = 10 events)
	recentConversation := a.extractRecentConversation(sess.Events(), RecentTurnsToKeep*2)
	if recentConversation != "" {
		log.Printf("[HISTORY] Preserved recent conversation: %d chars", len(recentConversation))
	}

	// Delete old session
	if err := a.sessionService.Delete(ctx, &session.DeleteRequest{
		AppName:   "talking_bookshelf",
		UserID:    userID,
		SessionID: sessionID,
	}); err != nil {
		log.Printf("[HISTORY] Warning: Failed to delete old session: %v", err)
	}

	// Create new session with recent conversation in state
	initialState := make(map[string]any)
	if recentConversation != "" {
		initialState[RecentConversationStateKey] = recentConversation
	}

	createResp, err := a.sessionService.Create(ctx, &session.CreateRequest{
		AppName:   "talking_bookshelf",
		UserID:    userID,
		SessionID: sessionID,
		State:     initialState,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create new session: %w", err)
	}

	log.Printf("[HISTORY] Created new session with recent conversation, ID: %s", createResp.Session.ID())

	return createResp.Session.ID(), nil
}

// extractRecentConversation extracts the last N events as formatted text
func (a *BookshelfAgent) extractRecentConversation(events session.Events, count int) string {
	totalEvents := events.Len()
	if totalEvents == 0 {
		return ""
	}

	// Determine start index for recent events
	startIdx := 0
	if totalEvents > count {
		startIdx = totalEvents - count
	}

	var parts []string
	for i := startIdx; i < totalEvents; i++ {
		event := events.At(i)
		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					role := "User"
					if event.Content.Role == "model" {
						role = "Assistant"
					}
					// Truncate very long messages
					text := part.Text
					if len(text) > 500 {
						text = text[:500] + "..."
					}
					parts = append(parts, fmt.Sprintf("%s: %s", role, text))
				}
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("[Recent conversation]\n%s", joinStrings(parts, "\n"))
}

// joinStrings joins strings with a separator (helper to avoid importing strings package)
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// deduplicateStrings removes duplicates from a string slice while preserving order
func deduplicateStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// bookIDPattern matches [book::title::book-xxx] patterns
var bookIDPattern = regexp.MustCompile(`\[book::[^:]+::(book-\d+)\]`)

// extractBookIDsFromText extracts book IDs from text that contains [book::title::id] patterns
func extractBookIDsFromText(text string) []string {
	matches := bookIDPattern.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	var ids []string
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			ids = append(ids, match[1])
			seen[match[1]] = true
		}
	}
	return ids
}
