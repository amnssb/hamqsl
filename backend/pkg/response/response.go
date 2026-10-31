package response

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PageData struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Pages int         `json:"pages"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: "QSL-0000", Message: "成功", Data: data})
}

func OKMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Response{Code: "QSL-0000", Message: msg})
}

func Page(c *gin.Context, items interface{}, total int64, page, size int) {
	pages := int(total) / size
	if int(total)%size > 0 {
		pages++
	}
	c.JSON(http.StatusOK, Response{
		Code: "QSL-0000", Message: "成功",
		Data: PageData{Items: items, Total: total, Page: page, Size: size, Pages: pages},
	})
}

// Fail 按 HTTP 状态返回错误。响应体里的 code 跟着状态走，
// 这样 429/410/423 这些新状态在日志和前端里也能一眼认出来，不会都显示成 QSL-400-0001。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(code, Response{Code: codeForStatus(code), Message: msg})
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "QSL-401-0001"
	case http.StatusForbidden:
		return "QSL-403-0001"
	case http.StatusNotFound:
		return "QSL-404-0001"
	case http.StatusConflict:
		return "QSL-409-0001"
	case http.StatusGone:
		return "QSL-410"
	case http.StatusLocked:
		return "QSL-423"
	case http.StatusTooManyRequests:
		return "QSL-429"
	default:
		return "QSL-400-0001"
	}
}

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{Code: "QSL-401-0001", Message: "认证失败"})
}

func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, Response{Code: "QSL-403-0001", Message: "无权限"})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{Code: "QSL-404-0001", Message: msg})
}

func Error(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{Code: "QSL-500-0001", Message: msg})
}
