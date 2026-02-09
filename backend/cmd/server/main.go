package main

import (
	"log"
	"os"
	"strings"
	"time"

	"talking-bookshelf/backend/internal/handler"
	"talking-bookshelf/backend/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	godotenv.Load(".env.local")

	env := os.Getenv("ENV")
	log.Printf("[INFO] Starting Talking Bookshelf env=%s", env)

	if err := handler.InitBookshelfAgent(); err != nil {
		log.Printf("[WARN] Failed to initialize Bookshelf agent: %v", err)
		log.Println("[WARN] Chat functionality will be unavailable")
	} else {
		log.Println("[INFO] Bookshelf agent initialized successfully")
	}

	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Security headers (before CORS)
	r.Use(middleware.SecurityHeaders())

	allowedOrigins := []string{}
	if gin.Mode() != gin.ReleaseMode {
		allowedOrigins = append(allowedOrigins, "http://localhost:5173")
	}
	if cloudRunURL := os.Getenv("CLOUD_RUN_URL"); cloudRunURL != "" {
		allowedOrigins = append(allowedOrigins, cloudRunURL)
	}
	if extraOrigins := os.Getenv("ALLOWED_ORIGINS"); extraOrigins != "" {
		allowedOrigins = append(allowedOrigins, strings.Split(extraOrigins, ",")...)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Accept-Language"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize rate limiters
	// 具体的な値は公開リポジトリから省略
	ipLimiter := middleware.NewIPRateLimiter(rate.Every(1*time.Second), 1)
	dailyQuota := middleware.NewDailyQuota(1)

	log.Printf("[INFO] Rate limiting enabled")

	// Health check endpoints (outside /api group, no rate limiting)
	r.GET("/health", handler.HandleHealth)
	r.GET("/ready", handler.HandleReadiness)

	api := r.Group("/api")
	{
		api.GET("/books", handler.HandleGetBooks)
		api.GET("/books/:id", handler.HandleGetBook)
		api.GET("/owner", handler.HandleGetOwner)
		api.POST("/chat", middleware.RateLimitMiddleware(ipLimiter, dailyQuota), handler.HandleChat)
	}

	if env == "production" {
		r.Static("/assets", "/app/static/assets")

		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(404, gin.H{"error": "Not found"})
				return
			}
			c.File("/app/static/index.html")
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[INFO] Server ready port=%s allowed_origins=%v", port, allowedOrigins)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("[FATAL] Failed to start server: %v", err)
	}
}
