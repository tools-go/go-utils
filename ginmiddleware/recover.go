package ginmiddleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tools-go/go-utils/dtrace"
	"github.com/tools-go/go-utils/errors"
)

// RecoverWithTrace middleware is a RecoverMiddleware wraps with a trace handler
func RecoverWithTrace(name string) Middleware {
	return func(next gin.HandlerFunc) gin.HandlerFunc {
		return func(ctx *gin.Context) {
			var rw *responseWriter
			if defaultResponseInterceptor != nil {
				rw = &responseWriter{
					ResponseWriter: ctx.Writer,
				}
				ctx.Writer = rw
			}
			recoverHandler := func(c *gin.Context) {
				tracer := dtrace.GetTraceFromContext(c)
					if rw, ok := c.Writer.(interface {
						Record(ctx context.Context, recorder Recorder)
					}); ok {
						defer rw.Record(c, defaultResponseInterceptor)
					}
				defer func() {
					if err := recover(); err != nil {
						tracer.Errorf("panic: err:%v, %v", err, tracer.Stack())
						if err, ok := err.(error); ok {
						    myErr := errors.ErrSwitch(err)
								msg := fmt.Sprintf("%s, [tid:%s]",myErr.Msg,tracer.ID() )
								http.Error(c.Writer, msg, myErr.Code)
							}else{
								http.Error(c.Writer, fmt.Sprintf("internal panic error, %v, [tid:%s]",err, tracer.ID()), http.StatusInternalServerError)
							}
						}
				}()
				next(c)
			}
			dtrace.HandlerFunc(name, recoverHandler)(ctx)
		}
	}
}
