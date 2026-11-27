package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetModelPerformance(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	minutes, err := strconv.Atoi(c.DefaultQuery("minutes", "60"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid performance window"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	overview, err := service.ReadModelPerformanceOverview(ctx, minutes, c.Query("model"), time.Now())
	if errors.Is(err, service.ErrInvalidModelPerformanceModel) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid performance filter"})
		return
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "model performance temporarily unavailable"})
		return
	}
	common.ApiSuccess(c, overview)
}
