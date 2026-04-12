package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorEnvelope struct {
	Error ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func Error(c *gin.Context, status int, code, message string) {
	requestID, _ := c.Get("request_id")
	c.JSON(status, ErrorEnvelope{Error: ErrorPayload{Code: code, Message: message, RequestID: toString(requestID)}})
}

func ValidationError(c *gin.Context, msg string) {
	Error(c, http.StatusBadRequest, "validation_error", msg)
}

func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
}

func Forbidden(c *gin.Context, msg string) {
	Error(c, http.StatusForbidden, "forbidden", msg)
}

func Internal(c *gin.Context) {
	Error(c, http.StatusInternalServerError, "internal_error", "An internal error occurred")
}

func toString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
