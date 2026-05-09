package response

import "github.com/gin-gonic/gin"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, Response{
		Success: false,
		Error: ErrorResponse{
			Code: code,
			Message: message,
		},
	})
}

func Abort(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, Response{
		Success: false,
		Error: ErrorResponse{
			Code: code,
			Message: message,
		},
	})
}