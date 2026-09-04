package apperr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GinHandlerFunc 带 error 返回值的 handler 签名
type GinHandlerFunc func(c *gin.Context) error

// Handle 将 GinHandlerFunc 转换为 gin.HandlerFunc，统一处理错误响应
func Handle(fn GinHandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := fn(c); err != nil {
			respond(c, err)
		}
	}
}

// respond 根据 error 类型统一输出响应
func respond(c *gin.Context, err error) {
	var appErr *Error
	if !errors.As(err, &appErr) {
		appErr = Internal(err)
	}

	// 记录日志（仅 5xx 和含 Cause 的错误记录详细日志）
	if appErr.HTTPStatus >= http.StatusInternalServerError {
		zap.L().Error("internal error",
			zap.Int("code", int(appErr.Code)),
			zap.String("msg", appErr.Message),
			zap.Error(appErr.Cause),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)
	} else if appErr.Cause != nil {
		zap.L().Warn("business error",
			zap.Int("code", int(appErr.Code)),
			zap.String("msg", appErr.Message),
			zap.Error(appErr.Cause),
		)
	}

	type resp struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data,omitempty"`
	}

	c.JSON(appErr.HTTPStatus, resp{
		Code: int(appErr.Code),
		Msg:  appErr.Message,
	})
}
