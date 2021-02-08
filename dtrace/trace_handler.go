// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dtrace

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	tracerLogHandlerID = "32702" // random key
	realIPValueID      = "16221"
	ginCtxID           = "343232"
	DefaultLoginUser   = "203832"
)

// HandleFunc wrap a trace handle func outer the original http handle func
func HandlerFunc(name string, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		tracer := GetTraceFromContext(c)
		if id := c.Request.Header.Get("x-request-id"); len(id) > 0 {
			if tracer.Parent() == nil && tracer.ID() != id {
				tracer = WithID(name, id)
			}
		}
		if tracer.Name() != name {
			tracer = WithParent(tracer, name)
		}
		c.Writer.Header().Set("x-request-id", tracer.ID())
		c.Set(tracerLogHandlerID, tracer)
		c.Set(ginCtxID, c)

		handler(c)
	}
}

// GetTraceFromRequest get the Trace var from the req context, if there is no such a trace utility, return nil
func GetTraceFromRequest(r *http.Request) Trace {
	return GetTraceFromContext(r.Context())
}

// GetTraceFromContext get the Trace var from the context, if there is no such a trace utility, return nil
func GetTraceFromContext(ctx context.Context) Trace {
	if tracer, ok := ctx.Value(tracerLogHandlerID).(Trace); ok {
		return tracer
	}
	return New("default-trace")
}

// GetRealIPFromContext get the remote endpoint from request, if not found, return an empty string
func GetRealIPFromContext(ctx context.Context) string {
	if ip, ok := ctx.Value(realIPValueID).(string); ok {
		return ip
	}
	return ""
}

// WithTraceForContext will return a new context wrapped a trace handler around the original ctx
func WithTraceForContext(ctx context.Context, traceName string, traceID ...string) context.Context {
	return context.WithValue(ctx, tracerLogHandlerID, New(traceName, traceID...))
}

// WithTraceForContext will return a new context wrapped a trace handler around the original ctx
func WithTraceForGinContext(ctx context.Context, traceName string, traceID ...string) *gin.Context {
	if gctx, ok := ctx.Value(ginCtxID).(*gin.Context); ok {
		gctx.Set(tracerLogHandlerID, New(traceName, traceID...))
		return gctx
	}
	gctx := &gin.Context{}
	gctx.Set(tracerLogHandlerID, New(traceName, traceID...))
	return gctx
}

// WithTraceForContext2 will return a new context wrapped a trace handler around the original ctx
func WithTraceForContext2(ctx context.Context, tracer Trace) context.Context {
	if tracer == nil {
		return ctx
	}
	return context.WithValue(ctx, tracerLogHandlerID, tracer)
}

func GetUserInfoFromContext(ctx context.Context) (string, error) {
	userInfo, ok := ctx.Value(DefaultLoginUser).(string)
	if ok && userInfo != "" {
		return userInfo, nil
	}

	return "", fmt.Errorf("user info not exists")
}
