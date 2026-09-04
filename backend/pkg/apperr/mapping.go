package apperr

import (
	"errors"
)

// DomainErrorMapping domain 哨兵错误到 AppError 的映射
type DomainErrorMapping struct {
	Target error
	AppErr *Error
}

// MapDomainError 根据映射表将 domain 错误转换为 AppError
// 如果未命中任何映射，返回 Internal
func MapDomainError(err error, mappings []DomainErrorMapping) *Error {
	if err == nil {
		return nil
	}
	// 如果已经是 AppError，直接返回
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	// 遍历映射表
	for _, m := range mappings {
		if errors.Is(err, m.Target) {
			result := *m.AppErr // 复制一份，避免共享状态
			result.Cause = err
			return &result
		}
	}
	// 兜底
	return Internal(err)
}
