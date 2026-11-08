package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newQueryContext(rawQuery string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	c.Request = req
	return c
}

func TestPageParamsDefaults(t *testing.T) {
	page, size := pageParams(newQueryContext(""))

	if page != 1 || size != defaultPageSize {
		t.Errorf("默认应为 page=1 size=%d，实际 page=%d size=%d", defaultPageSize, page, size)
	}
}

func TestPageParamsClampsLargeSize(t *testing.T) {
	// 这条是防 ?size=1000000 把整张表拉进内存的
	_, size := pageParams(newQueryContext("size=1000000"))

	if size != maxPageSize {
		t.Errorf("size 应被夹到 %d，实际 %d", maxPageSize, size)
	}
}

func TestPageParamsRejectsNegativeAndZero(t *testing.T) {
	cases := []struct {
		query    string
		wantPage int
		wantSize int
	}{
		{"page=0&size=0", 1, defaultPageSize},
		{"page=-5&size=-1", 1, defaultPageSize},
		{"page=abc&size=xyz", 1, defaultPageSize},
	}
	for _, tc := range cases {
		page, size := pageParams(newQueryContext(tc.query))
		if page != tc.wantPage || size != tc.wantSize {
			t.Errorf("%s → page=%d size=%d，期望 page=%d size=%d",
				tc.query, page, size, tc.wantPage, tc.wantSize)
		}
	}
}

func TestPageParamsKeepsNormalValues(t *testing.T) {
	page, size := pageParams(newQueryContext("page=3&size=50"))

	if page != 3 || size != 50 {
		t.Errorf("正常值应原样保留，实际 page=%d size=%d", page, size)
	}
}

func TestPageParamsSizeAtLimit(t *testing.T) {
	_, size := pageParams(newQueryContext("size=200"))
	if size != maxPageSize {
		t.Errorf("边界值 200 应原样保留，实际 %d", size)
	}

	_, size = pageParams(newQueryContext("size=201"))
	if size != maxPageSize {
		t.Errorf("201 应被夹到 %d，实际 %d", maxPageSize, size)
	}
}

// —— callSignPattern / containsPattern ——

func TestCallSignPatternIsPrefix(t *testing.T) {
	got := callSignPattern("  bi1kbu ")

	if got != "bi1kbu%" {
		t.Errorf("呼号必须前缀匹配才能吃索引，实际 %q", got)
	}
	if strings.HasPrefix(got, "%") {
		t.Errorf("前缀通配符会让 call_sign 索引失效，实际 %q", got)
	}
}

func TestContainsPatternIsInfix(t *testing.T) {
	if got := containsPattern(" zhang "); got != "%zhang%" {
		t.Errorf("中缀匹配需要两侧通配符，实际 %q", got)
	}
}

func TestPatternsHandleEmptyInput(t *testing.T) {
	if got := callSignPattern(""); got != "%" {
		t.Errorf(`空呼号应退化为 "%%"（匹配全部），实际 %q`, got)
	}
	if got := containsPattern(""); got != "%%" {
		t.Errorf(`空关键字应退化为 "%%%%"，实际 %q`, got)
	}
}
