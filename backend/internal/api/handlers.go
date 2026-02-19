package api

import (
	"net/http"

	"github.com/alanmaizon/homer/backend/internal/agents"
	"github.com/alanmaizon/homer/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	router.POST("/api/task", func(c *gin.Context) {
		var req domain.TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task payload"})
			return
		}

		if req.Task == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task is required"})
			return
		}
		if req.Task == domain.TaskSummarize && len(req.Documents) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "documents are required for summarize"})
			return
		}
		if req.Task == domain.TaskRewrite && req.Text == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "text is required for rewrite"})
			return
		}
		if req.Task != domain.TaskSummarize && req.Task != domain.TaskRewrite {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported task"})
			return
		}

		response, err := agents.ExecuteTask(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})
}
