package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Agent     string `json:"agent"`
}

// HandleHealth returns the health status of the service
// Used for Cloud Run liveness probe
func HandleHealth(c *gin.Context) {
	agentMu.RLock()
	agentStatus := "unavailable"
	if bookshelfAgent != nil {
		agentStatus = "ready"
	}
	agentMu.RUnlock()

	status := "healthy"
	if agentStatus == "unavailable" {
		status = "degraded"
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Agent:     agentStatus,
	})
}

// HandleReadiness returns whether the service is ready to accept traffic
// Used for Cloud Run startup probe - stricter than health
func HandleReadiness(c *gin.Context) {
	agentMu.RLock()
	agentReady := bookshelfAgent != nil
	agentMu.RUnlock()

	if !agentReady {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"reason": "agent_not_initialized",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
