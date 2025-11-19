package api

import "github.com/gin-gonic/gin"

// StandardResponse defines the structure for a standardized API response.
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

// ErrorResponse defines the structure for a detailed error message.
type ErrorResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewSuccessResponse creates a standardized success response.
func NewSuccessResponse(data interface{}, meta interface{}) StandardResponse {
	return StandardResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}

// NewErrorResponse creates a standardized error response.
func NewErrorResponse(code int, message string, details string) StandardResponse {
	return StandardResponse{
		Success: false,
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// Respond sends a JSON response with a status code.
func Respond(c *gin.Context, statusCode int, response StandardResponse) {
	c.JSON(statusCode, response)
}
