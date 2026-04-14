package timehandler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/httputil"
)

// Handler serves the /time endpoint.
type Handler struct {
    now func() time.Time
}

// NewHandler constructs a time handler.
func NewHandler(now func() time.Time) *Handler {
    if now == nil {
        now = time.Now
    }
    return &Handler{now: now}
}

// Get returns the time in the requested timezone.
func (h *Handler) Get(c *gin.Context) {
    timezone := c.GetHeader("X-Timezone")
    if timezone == "" {
        timezone = "UTC"
    }

    location, err := time.LoadLocation(timezone)
    if err != nil {
        httputil.WriteError(c, http.StatusBadRequest, "invalid_timezone", "invalid timezone")
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "time":     h.now().In(location).Format(time.RFC3339),
        "timezone": timezone,
    })
}
