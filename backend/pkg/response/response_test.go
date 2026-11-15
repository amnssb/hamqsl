package response

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFailUsesStatusCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, 422, "卡片尚未寄出，暂不能确认收件")

	if w.Code != 422 {
		t.Fatalf("HTTP 状态 = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"code":"QSL-422"`) {
		t.Fatalf("422 不应再伪装成 QSL-400，实际 %s", w.Body.String())
	}
}

func TestPageZeroSizeDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Page(c, []int{}, 3, 0, 0)

	if w.Code != 200 {
		t.Fatalf("size=0 不应把分页打崩，实际 %d %s", w.Code, w.Body.String())
	}
}
