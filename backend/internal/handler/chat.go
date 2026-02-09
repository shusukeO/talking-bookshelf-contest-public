package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"talking-bookshelf/backend/internal/agent"
	"talking-bookshelf/backend/internal/portfolio"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// ChatTimeout is the maximum time allowed for a chat request
	ChatTimeout = 30 * time.Second
	// MaxRetries is the maximum number of retries for failed requests
	MaxRetries = 2
	// MaxMessageLength is the maximum allowed message length
	MaxMessageLength = 250
)

type ChatRequest struct {
	Message   string  `json:"message" binding:"required,max=250"`
	BookID    *string `json:"bookId,omitempty"`
	SessionID *string `json:"sessionId,omitempty"`
	Language  *string `json:"language,omitempty"`
}

type ChatResponseDTO struct {
	Response    string   `json:"response"`
	Emotion     string   `json:"emotion"`
	Suggestions []string `json:"suggestions"`
	SessionID   string   `json:"sessionId"`
}

var (
	bookshelfAgent *agent.BookshelfAgent
	agentMu        sync.RWMutex
)

// InitBookshelfAgent initializes the ADK-based bookshelf agent
func InitBookshelfAgent() error {
	ctx := context.Background()

	// Load portfolio data
	p, err := portfolio.LoadPortfolio("data/portfolio.json")
	if err != nil {
		log.Printf("Warning: Failed to load portfolio: %v", err)
		p = nil
	}

	// Get books
	books := GetBooks()

	// Create agent
	agentMu.Lock()
	defer agentMu.Unlock()

	bookshelfAgent, err = agent.NewBookshelfAgent(ctx, books, p)
	if err != nil {
		return err
	}

	log.Println("Bookshelf agent initialized successfully with ADK")
	return nil
}

func HandleChat(c *gin.Context) {
	startTime := time.Now()

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if strings.Contains(err.Error(), "max") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Message is too long (max 250 characters)",
				"code":  "MESSAGE_TOO_LONG",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: message is required",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Normalize Unicode to NFC form before security checks
	// This prevents bypass attacks using Unicode lookalike characters
	req.Message = norm.NFC.String(req.Message)

	// Check for prompt injection attempts
	if isInjectionAttempt(req.Message) {
		c.JSON(http.StatusOK, ChatResponseDTO{
			Response:    "その質問にはお答えできないよ。本についておしゃべりしよう！",
			Emotion:     "idle",
			Suggestions: []string{"おすすめの本は？", "最近読んだ本は？"},
			SessionID:   "",
		})
		return
	}

	// Validate bookId exists if provided
	if req.BookID != nil && *req.BookID != "" {
		if GetBookByID(*req.BookID) == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "The specified book was not found",
				"code":  "BOOK_NOT_FOUND",
			})
			return
		}
	}

	agentMu.RLock()
	currentAgent := bookshelfAgent
	agentMu.RUnlock()

	if currentAgent == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI service is not available",
			"code":  "SERVICE_UNAVAILABLE",
		})
		return
	}

	// Get user ID first (used for both session creation and chat)
	userID := generateUserID(c)

	// Get or create session ID
	sessionID := ""
	if req.SessionID != nil && *req.SessionID != "" {
		sessionID = *req.SessionID
	} else {
		// Create new session
		newSessionID, err := currentAgent.CreateSession(c.Request.Context(), userID)
		if err != nil {
			log.Printf("Warning: Failed to create session: %v, using random ID", err)
			sessionID = uuid.New().String()
		} else {
			sessionID = newSessionID
		}
	}

	// Determine response language
	language := determineLanguage(req, c)
	log.Printf("[LANG] Determined language: %s", language)

	setupDuration := time.Since(startTime)

	// Call agent with timeout and retry
	chatStart := time.Now()
	resp, err := chatWithRetry(c.Request.Context(), currentAgent, userID, sessionID, req.Message, req.BookID, language)
	chatDuration := time.Since(chatStart)

	if err != nil {
		log.Printf("[PERF] Chat failed after %v (setup: %v)", chatDuration, setupDuration)
		log.Printf("Agent chat error after retries: %v", err)

		// Return fallback response for better UX
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error":    "Request timed out. Please try again.",
				"code":     "TIMEOUT",
				"fallback": true,
			})
			return
		}

		// Check for Gemini API rate limit (ResourceExhausted)
		if isRateLimitError(err) {
			log.Printf("[QUOTA] Gemini API rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"response":    "I've been talking too much today! Please come back in a bit.",
				"emotion":     "surprised",
				"suggestions": []string{},
				"fallback":    true,
				"code":        "GEMINI_RATE_LIMITED",
				"retryAfter":  60,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "Failed to generate response. Please try again.",
			"code":     "INTERNAL_ERROR",
			"fallback": true,
		})
		return
	}

	totalDuration := time.Since(startTime)
	log.Printf("[PERF] Chat completed in %v (setup: %v, agent: %v)", totalDuration, setupDuration, chatDuration)

	c.JSON(http.StatusOK, ChatResponseDTO{
		Response:    resp.Response,
		Emotion:     resp.Emotion,
		Suggestions: resp.Suggestions,
		SessionID:   sessionID,
	})
}

// chatWithRetry calls the agent with timeout and retry logic
func chatWithRetry(ctx context.Context, currentAgent *agent.BookshelfAgent, userID, sessionID, message string, bookID *string, language string) (*agent.ChatResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("[RETRY] Attempt %d/%d for chat", attempt+1, MaxRetries+1)
			// Brief delay before retry
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}

		// Create context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, ChatTimeout)

		resp, err := currentAgent.Chat(timeoutCtx, userID, sessionID, message, bookID, language)
		cancel()

		if err == nil {
			return resp, nil
		}

		lastErr = err
		log.Printf("[RETRY] Chat attempt %d failed: %v", attempt+1, err)

		// Don't retry on context cancelled (user disconnected)
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
	}

	return nil, lastErr
}

// generateUserID generates a user ID based on request context
func generateUserID(c *gin.Context) string {
	// Use IP address as a simple user identifier
	// In production, you might want to use cookies or authentication
	ip := c.ClientIP()
	if ip == "" {
		ip = "anonymous"
	}
	return "user_" + ip
}

// supportedLanguages defines the languages we support
var supportedLanguages = map[string]bool{
	"ja": true,
	"en": true,
}

// determineLanguage determines the response language from request body and Accept-Language header
// Priority: 1. Request body language field, 2. Accept-Language header, 3. Default (en)
func determineLanguage(req ChatRequest, c *gin.Context) string {
	// 1. Check request body language field (highest priority)
	if req.Language != nil && *req.Language != "" {
		lang := normalizeLanguage(*req.Language)
		if supportedLanguages[lang] {
			return lang
		}
	}

	// 2. Check Accept-Language header
	acceptLang := c.GetHeader("Accept-Language")
	if acceptLang != "" {
		lang := parseAcceptLanguage(acceptLang)
		if lang != "" && supportedLanguages[lang] {
			return lang
		}
	}

	// 3. Default to English
	return "en"
}

// normalizeLanguage extracts the base language code (e.g., "ja-JP" -> "ja")
func normalizeLanguage(lang string) string {
	// Handle language codes like "ja-JP", "en-US"
	if idx := strings.Index(lang, "-"); idx != -1 {
		return strings.ToLower(lang[:idx])
	}
	return strings.ToLower(lang)
}

// parseAcceptLanguage extracts the preferred language from Accept-Language header
// Example: "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7" -> "ja"
func parseAcceptLanguage(header string) string {
	// Split by comma and process each language
	parts := strings.Split(header, ",")
	for _, part := range parts {
		// Remove quality value (e.g., ";q=0.9")
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		normalized := normalizeLanguage(lang)
		if supportedLanguages[normalized] {
			return normalized
		}
	}
	return ""
}

// isRateLimitError checks if the error is a Gemini API rate limit error
func isRateLimitError(err error) bool {
	// Check for gRPC ResourceExhausted status
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.ResourceExhausted
	}
	// Also check for wrapped errors and string matching as fallback
	errStr := err.Error()
	return strings.Contains(errStr, "ResourceExhausted") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "quota")
}

// injectionPatterns contains 50+ compiled regex patterns for prompt injection detection.
// Patterns cover: Japanese, English, Chinese, Korean, and encoding-based attacks.
// Categories include: role escape, direct quotation, prompt leakage, jailbreak, and obfuscation.
// Patterns are omitted from the public repository.
var injectionPatterns = []*regexp.Regexp{
	// TODO: Add your prompt injection detection patterns here.
	// Example: regexp.MustCompile(`(?i)ignore.*(previous|all|instructions)`),
}

// isInjectionAttempt checks user input against all injection patterns.
// Returns true if any pattern matches, blocking the message before it reaches the LLM.
func isInjectionAttempt(message string) bool {
	for _, pattern := range injectionPatterns {
		if pattern.MatchString(message) {
			log.Printf("[SECURITY] Injection attempt blocked")
			return true
		}
	}
	return false
}
