package ping

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/nk2580/Tester-API/internal/data"
    "github.com/nk2580/Tester-API/internal/httputil"
)

// Handler manages ping-related endpoints.
type Handler struct {
    store data.PingStore
}

// NewHandler constructs a ping handler.
func NewHandler(store data.PingStore) *Handler {
    return &Handler{store: store}
}

// Create registers a ping payload.
func (h *Handler) Create(c *gin.Context) {
    var ping data.Ping
    if err := c.ShouldBindJSON(&ping); err != nil {
        httputil.WriteError(c, http.StatusBadRequest, "invalid_request", "invalid ping payload")
        return
    }

    if err := h.store.CreatePing(c.Request.Context(), &ping); err != nil {
        httputil.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to save ping")
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Ping registered successfully!"})
}

// List returns stored pings.
func (h *Handler) List(c *gin.Context) {
    pings, err := h.store.ListPings(c.Request.Context())
    if err != nil {
        httputil.WriteError(c, http.StatusInternalServerError, "internal_error", "failed to retrieve pings")
        return
    }

    c.JSON(http.StatusOK, pings)
}
