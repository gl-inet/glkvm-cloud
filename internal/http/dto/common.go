package dto

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Meta struct {
	TraceID string `json:"traceId"`
	TS      int64  `json:"ts"`
}

type Envelope[T any] struct {
	Ok      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Meta    Meta   `json:"meta"`
}

const (
	CodeOK               = "OK"
	CodeInvalidArgument  = "INVALID_ARGUMENT"
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeAuthRequired     = "AUTH_REQUIRED"
	CodeAuthExpired      = "AUTH_EXPIRED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeInternalError    = "INTERNAL_ERROR"
)

func NowUnix() int64 { return time.Now().Unix() }

// Write always replies with HTTP 200 per spec.
func Write[T any](c *gin.Context, env Envelope[T]) {
	c.JSON(http.StatusOK, env)
}

func Ok[T any](traceID string, data T) Envelope[T] {
	return Envelope[T]{
		Ok:   true,
		Code: CodeOK,
		Data: data,
		Meta: Meta{TraceID: traceID, TS: NowUnix()},
	}
}

func Err(traceID, code, msg string, data any) Envelope[any] {
	return Envelope[any]{
		Ok:      false,
		Code:    code,
		Message: msg,
		Data:    data,
		Meta:    Meta{TraceID: traceID, TS: NowUnix()},
	}
}
