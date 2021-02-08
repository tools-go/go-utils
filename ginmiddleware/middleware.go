package ginmiddleware

import (
	"github.com/gin-gonic/gin"
)

// Middleware is a chainable preprocessor for Endpoint
type Middleware func(next gin.HandlerFunc) gin.HandlerFunc

// HandlerFunc will return the HandlerFunc of the middleware
func (m Middleware) HandlerFunc(next gin.HandlerFunc) gin.HandlerFunc {
	return m(next)
}

