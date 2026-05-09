package response

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, status int, message string, data any, meta any) {
	c.JSON(status, Response{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}