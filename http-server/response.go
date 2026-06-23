package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// send success response
func Ok(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Error:   nil,
		Meta:    nil,
	})
}

// send error response
func Error(ctx *gin.Context, status int, code string, message string) {
	ctx.JSON(status, Response{
		Success: false,
		Data:    nil,
		Error:   &ErrorInfo{Code: code, Message: message},
		Meta:    nil,
	})
}
