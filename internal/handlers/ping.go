package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/api"
	"github.com/nk2580/Tester-API/internal/models"
	"gorm.io/gorm"
)

type PingHandler struct {
	db *gorm.DB
}

func NewPingHandler(db *gorm.DB) *PingHandler {
	return &PingHandler{db: db}
}

type createPingRequest struct {
	Message string `json:"message" binding:"required,max=1000"`
}

func (h *PingHandler) Create(c *gin.Context) {
	var req createPingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ValidationError(c, "Invalid ping payload")
		return
	}

	ping := models.Ping{Message: req.Message}
	if err := h.db.Create(&ping).Error; err != nil {
		api.Internal(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
}

func (h *PingHandler) List(c *gin.Context) {
	var pings []models.Ping
	if err := h.db.Order("id asc").Find(&pings).Error; err != nil {
		api.Internal(c)
		return
	}
	c.JSON(http.StatusOK, pings)
}

func (h *PingHandler) Hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
}

func (h *PingHandler) Time(c *gin.Context) {
	timezone := c.GetHeader("X-Timezone")
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		api.ValidationError(c, "Invalid timezone")
		return
	}
	c.JSON(http.StatusOK, gin.H{"time": time.Now().In(location).Format(time.RFC3339), "timezone": timezone})
}
