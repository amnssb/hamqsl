package flow

import (
	"time"
	_ "time/tzdata"
)

// 全站「今天」和入库时间戳都用上海时区。中国无夏令时，加载失败时退回固定东八区。
var shanghai = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*3600)
	}
	return loc
}()

const stampLayout = "2006-01-02 15:04:05"

// Now 当前上海时间。
func Now() time.Time { return time.Now().In(shanghai) }

// Stamp 写成入库用的墙钟字符串（不带时区后缀）。
func Stamp(t time.Time) string { return t.In(shanghai).Format(stampLayout) }

// Day 上海时区的日期，形如 2006-01-02。
func Day(t time.Time) string { return t.In(shanghai).Format("2006-01-02") }
