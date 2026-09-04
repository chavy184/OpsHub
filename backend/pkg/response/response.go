package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// R 统一响应结构
type R struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// PageData 分页数据
type PageData struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"page_size"`
}

// OK 成功响应
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, R{Code: 0, Msg: "success", Data: data})
}

// OKPage 分页成功响应
func OKPage(c *gin.Context, list interface{}, total int64, page, size int) {
	c.JSON(http.StatusOK, R{
		Code: 0,
		Msg:  "success",
		Data: PageData{List: list, Total: total, Page: page, Size: size},
	})
}

// Created 创建成功
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, R{Code: 0, Msg: "success", Data: data})
}

// Fail 业务失败
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, R{Code: code, Msg: msg})
}

// ErrBadRequest 参数错误
func ErrBadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, R{Code: 1001, Msg: msg})
}

// ErrBadRequestCode 参数或业务错误，允许调用方保留稳定业务码
func ErrBadRequestCode(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusBadRequest, R{Code: code, Msg: msg})
}

// ErrUnauthorized 未授权
func ErrUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, R{Code: 1002, Msg: "未授权"})
}

// ErrForbidden 无权限
func ErrForbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, R{Code: 1003, Msg: "无权限"})
}

// ErrNotFound 资源不存在
func ErrNotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, R{Code: 2001, Msg: msg})
}

// ErrConflict 冲突
func ErrConflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, R{Code: 3002, Msg: msg})
}

// ErrInternal 系统异常
func ErrInternal(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, R{Code: 5000, Msg: "系统异常"})
}

// NoContent 204 无内容
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
