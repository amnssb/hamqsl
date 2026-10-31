package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// 分页参数的统一入口。
//
// 之前每个列表接口各自 strconv.Atoi(c.DefaultQuery("size","20"))，都不设上界：
// 一个 ?size=1000000 就能让服务端把整张表读进内存再序列化成 JSON，
// 手滑或恶意都够用。这里统一夹到 1..maxPageSize。
const (
	defaultPageSize = 20
	maxPageSize     = 200
)

func pageParams(c *gin.Context) (page, size int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ = strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return page, size
}

// callSignPattern 呼号过滤。
//
// 呼号是前缀语义（BI1KBU / BI1KBU/P），用 "CS%" 而不是 "%CS%"：
// 前置通配符会让 call_sign 上的索引彻底失效，退化成全表扫。
// 呼号里只有字母数字和斜杠，不存在 LIKE 通配符，无需转义。
// 不在这里改大小写：SQLite 的 LIKE 对 ASCII 本就不区分大小写，
// 而 PG 上入库时已经统一成大写，两种驱动都按原样匹配即可。
func callSignPattern(v string) string {
	return strings.TrimSpace(v) + "%"
}

// containsPattern 用于姓名 / 地址 / 单号这类必须中缀匹配的字段。
// 这类查询吃不到索引，只在数据量小、且是人工点查询的场景使用。
func containsPattern(v string) string {
	return "%" + strings.TrimSpace(v) + "%"
}
