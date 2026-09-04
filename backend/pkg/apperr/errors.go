package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code 业务错误码
type Code int

const (
	// 通用
	CodeOK           Code = 0
	CodeBadRequest   Code = 1001
	CodeUnauthorized Code = 1002
	CodeForbidden    Code = 1003

	// 资源
	CodeNotFound Code = 2001
	CodeConflict Code = 2002

	// 业务
	CodeBizError Code = 3001

	// 系统
	CodeInternal Code = 5000
)

// Error 统一应用错误
type Error struct {
	Code       Code   // 业务错误码
	HTTPStatus int    // HTTP 状态码
	Message    string // 面向用户的错误消息
	Cause      error  // 原始错误（不暴露给前端）
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ---- 构造函数 ----

// BadRequest 参数错误 (400)
func BadRequest(msg string) *Error {
	return &Error{Code: CodeBadRequest, HTTPStatus: http.StatusBadRequest, Message: msg}
}

// NotFound 资源不存在 (404)
func NotFound(msg string) *Error {
	return &Error{Code: CodeNotFound, HTTPStatus: http.StatusNotFound, Message: msg}
}

// Conflict 资源冲突 (409)
func Conflict(msg string) *Error {
	return &Error{Code: CodeConflict, HTTPStatus: http.StatusConflict, Message: msg}
}

// Forbidden 无权限 (403)
func Forbidden(msg string) *Error {
	return &Error{Code: CodeForbidden, HTTPStatus: http.StatusForbidden, Message: msg}
}

// Unauthorized 未授权 (401)
func Unauthorized(msg string) *Error {
	return &Error{Code: CodeUnauthorized, HTTPStatus: http.StatusUnauthorized, Message: msg}
}

// BizError 业务错误 (200 + 业务码)
func BizError(code Code, msg string) *Error {
	return &Error{Code: code, HTTPStatus: http.StatusOK, Message: msg}
}

// Internal 系统错误 (500)，不暴露内部细节
func Internal(cause error) *Error {
	return &Error{Code: CodeInternal, HTTPStatus: http.StatusInternalServerError, Message: "系统异常", Cause: cause}
}

// Wrap 包装任意错误为 AppError，如果已经是 AppError 则直接返回
func Wrap(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}

// Is 判断 err 链中是否包含目标 sentinel 错误
func Is(err, target error) bool {
	return errors.Is(err, target)
}
