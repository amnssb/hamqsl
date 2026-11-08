package handler

import (
	"testing"
	"time"
)

// 时区口径是这次修的真实 bug：入库时间是东八区墙钟字符串，
// 用 time.Parse 会按 UTC 解释，比真实时刻晚 8 小时。
// ensureTrackingFresh 的 10 分钟窗口因此一直判成"未过期"。

func TestParseCSTReadsAsEast8(t *testing.T) {
	const s = "2026-11-07 12:00:00"

	got, err := parseCST(s)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if got.Format(cstLayout) != s {
		t.Errorf("往返不一致: %s -> %s", s, got.Format(cstLayout))
	}
	// 东八区 12:00 == UTC 04:00
	if h := got.UTC().Hour(); h != 4 {
		t.Errorf("期望 UTC 04:00，实际 UTC %02d:00", h)
	}
	if name, offset := got.Zone(); offset != 8*3600 {
		t.Errorf("期望东八区（+28800s），实际 %s %d", name, offset)
	}
}

// 这条是回归用例：旧写法 time.Parse 会把 8 小时前的记录当成"刚刚更新"。
func TestParseCSTDoesNotTreatCSTAsUTC(t *testing.T) {
	// 造一条"8 小时前"的东八区时间戳
	eightHoursAgo := cstNow().Add(-8 * time.Hour).Format(cstLayout)

	parsed, err := parseCST(eightHoursAgo)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	age := time.Since(parsed)

	// 正确解析：age 约等于 8 小时，理应超过 10 分钟缓存窗口
	if age < trackingCacheTTL {
		t.Errorf("8 小时前的时间戳被当成 %v 前，缓存窗口 %v 内 → 轨迹永远不会刷新", age, trackingCacheTTL)
	}
	// 旧写法 time.Parse 会算出 8 小时 - 8 小时 ≈ 0（甚至是负数）
	if age < 0 {
		t.Errorf("解析出未来时间（%v），说明时区方向搞反了", age)
	}
}

func TestParseCSTAcceptsFreshTimestamp(t *testing.T) {
	// 刚写入的时间戳应当落在缓存窗口内
	justNow := cstNow().Format(cstLayout)

	parsed, err := parseCST(justNow)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if age := time.Since(parsed); age >= trackingCacheTTL {
		t.Errorf("刚写入的时间戳被判成 %v 前，缓存会立刻失效（每次请求都查快递100）", age)
	}
}

func TestParseCSTRejectsGarbage(t *testing.T) {
	for _, s := range []string{"", "2026-11-07", "not-a-time", "2026/11/07 12:00:00"} {
		if _, err := parseCST(s); err == nil {
			t.Errorf("%q 应当解析失败", s)
		}
	}
}

// cstNow 必须真的在东八区，否则所有 Format 出来的字符串都会偏
func TestCstNowIsEast8(t *testing.T) {
	if _, offset := cstNow().Zone(); offset != 8*3600 {
		t.Errorf("cstNow 不在东八区，offset=%d", offset)
	}
}
