package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func isModelPerformanceRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}
	// This header delegates to a background job that clones the Gin keys.
	// Do not count its acknowledgement or share a trace with that job.
	switch strings.ToLower(strings.TrimSpace(c.GetHeader("X-Oneapi-Background"))) {
	case "1", "true", "yes":
		return false
	}
	switch c.FullPath() {
	case "/v1/chat/completions", "/v1/completions", "/v1/messages", "/v1/responses":
		return true
	case "/v1/models/*path", "/v1beta/models/*path":
		path := c.Param("path")
		return strings.HasSuffix(path, ":generateContent") || strings.HasSuffix(path, ":streamGenerateContent")
	default:
		return false
	}
}

// No body read: Distribute supplies the already-parsed model. Auth failures
// and early limits where the model is not yet known are outside this dataset.
func ModelPerformance() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isModelPerformanceRequest(c) {
			if t := service.BeginModelPerformance(c); t != nil {
				c.Next()
				t.Finish(c)
				return
			}
		}
		c.Next()
	}
}
