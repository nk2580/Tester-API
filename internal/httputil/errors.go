package httputil

import "github.com/gin-gonic/gin"

// WriteError sends a consistent JSON error response.
func WriteError(c *gin.Context, status int, code string, message string) {
    c.JSON(status, gin.H{
        "error": gin.H{
            "code":    code,
            "message": message,
        },
    })
}
