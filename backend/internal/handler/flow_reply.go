package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qsl-management/internal/flow"
	"qsl-management/pkg/response"
)

// writeFlowErr 把状态机拒绝的迁移写成 422。普通错误返回 false，由调用方按 500 处理。
func writeFlowErr(c *gin.Context, err error) bool {
	fe, ok := flow.AsError(err)
	if !ok {
		return false
	}
	response.FailCode(c, http.StatusUnprocessableEntity, fe.Code, fe.Message)
	return true
}
