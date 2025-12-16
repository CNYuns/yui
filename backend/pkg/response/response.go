package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  "ok",
		Data: data,
	})
}

func SuccessMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Msg:  msg,
	})
}

// Error 返回业务错误（HTTP 200，业务错误码）
// 用于预期内的业务逻辑错误
func Error(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
	})
}

// ErrorWithStatus 返回带 HTTP 状态码的错误
// 用于需要明确 HTTP 状态码的场景
func ErrorWithStatus(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
	})
}

func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{
		Code: 400,
		Msg:  msg,
	})
}

func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code: 401,
		Msg:  msg,
	})
}

func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, Response{
		Code: 403,
		Msg:  msg,
	})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{
		Code: 404,
		Msg:  msg,
	})
}

func ServerError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{
		Code: 500,
		Msg:  msg,
	})
}

// TooManyRequests 请求过于频繁
func TooManyRequests(c *gin.Context, msg string) {
	c.JSON(http.StatusTooManyRequests, Response{
		Code: 429,
		Msg:  msg,
	})
}
