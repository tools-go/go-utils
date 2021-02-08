package ginmiddleware

import (
	"github.com/gin-gonic/gin"
	"github.com/tools-go/go-utils/dtrace"
)

// Trace will create a trace handler middleware
func Trace(name string) Middleware {
	return func(next gin.HandlerFunc) gin.HandlerFunc {
		return dtrace.HandlerFunc(name, next)
	}
}
