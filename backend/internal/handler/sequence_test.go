package handler

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qsl-management/internal/model"
)

// 发号器是并发正确性的关键：原来"读 system_settings → +1 → 写回"没有互斥，
// 两个人同时提交换卡申请会拿到同一个编号，唯一索引挡住写入后其中一个请求 500。
// 这些用例把"不重号"钉死。

// newTestDB 每个用例一个独立的内存库。
//
// 注意不能固定写 "file::memory:?cache=shared"——共享缓存意味着所有测试
// 跑在同一个库上，序号会互相污染（一个用例 +1 完之后另一个用例就看不到 1 了）。
// 用 t.Name() 做库名即可互相隔离。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	for _, tbl := range []interface{}{
		&model.SystemSetting{}, &model.CardRecord{}, &model.ReceiveRecord{},
		&model.QsoRecord{}, &model.ExchangeRequest{},
	} {
		if err := db.AutoMigrate(tbl); err != nil {
			t.Fatalf("建表失败: %v", err)
		}
	}
	return db
}

func TestNextSequenceIncrements(t *testing.T) {
	db := newTestDB(t)

	first := nextSequence(db, "test_seq", "QSO-", 6, "")
	second := nextSequence(db, "test_seq", "QSO-", 6, "")

	if first != "QSO-000001" {
		t.Errorf("首个序号应为 QSO-000001，实际 %q", first)
	}
	if second != "QSO-000002" {
		t.Errorf("第二个序号应为 QSO-000002，实际 %q", second)
	}
}

func TestNextSequenceWithSuffix(t *testing.T) {
	db := newTestDB(t)

	got := nextSequence(db, "test_seq", "R", 4, "-20261107")

	if got != "R0001-20261107" {
		t.Errorf("带后缀的序号格式不对: %q", got)
	}
}

func TestNextSequenceZeroPadding(t *testing.T) {
	db := newTestDB(t)

	// 跨过补零边界
	var last string
	for i := 0; i < 12; i++ {
		last = nextSequence(db, "test_seq", "C", 3, "")
	}

	if last != "C012" {
		t.Errorf("3 位补零应得到 C012，实际 %q", last)
	}
}

// 核心用例：并发取号不能出现重复
func TestNextSequenceConcurrentNoDuplicates(t *testing.T) {
	db := newTestDB(t)
	const goroutines = 30
	const perGoroutine = 20

	var mu sync.Mutex
	seen := make(map[string]int, goroutines*perGoroutine)
	var wg sync.WaitGroup

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				code := nextSequence(db, "test_seq", "C", 8, "")
				mu.Lock()
				seen[code]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	total := goroutines * perGoroutine
	if len(seen) != total {
		t.Errorf("取号出现重复：期望 %d 个唯一编号，实际 %d", total, len(seen))
	}
	for code, n := range seen {
		if n > 1 {
			t.Errorf("编号 %s 被发了 %d 次", code, n)
		}
	}
}

// 一次领走一段序号后，下一次单取必须从这段的下一个开始，不能重号也不能回头
func TestReserveSequenceThenNext(t *testing.T) {
	db := newTestDB(t)

	start := reserveSequence(db, "test_seq", 5)
	got := nextSequence(db, "test_seq", "QSO-", 6, "")

	if start != 1 {
		t.Errorf("批量领号应从 1 开始，实际 %d", start)
	}
	if got != "QSO-000006" {
		t.Errorf("批量领走 5 个后，下一个应为 QSO-000006，实际 %q", got)
	}
}

// 序号必须单调递增且连续（不跳号、不回退）
func TestNextSequenceMonotonic(t *testing.T) {
	db := newTestDB(t)

	var prev int
	for i := 0; i < 50; i++ {
		code := nextSequence(db, "test_seq", "QSO-", 6, "")
		n, err := strconv.Atoi(code[len("QSO-"):])
		if err != nil {
			t.Fatalf("解析序号失败: %q", code)
		}
		if n != prev+1 {
			t.Fatalf("序号不连续：第 %d 次得到 %d，期望 %d", i, n, prev+1)
		}
		prev = n
	}
}

// 独立 key 之间互不影响
func TestNextSequenceIndependentKeys(t *testing.T) {
	db := newTestDB(t)

	a1 := nextSequence(db, "seq_a", "A", 3, "")
	b1 := nextSequence(db, "seq_b", "B", 3, "")
	a2 := nextSequence(db, "seq_a", "A", 3, "")

	if a1 != "A001" || b1 != "B001" || a2 != "A002" {
		t.Errorf("不同 key 应各自计数，实际 a1=%q b1=%q a2=%q", a1, b1, a2)
	}
}

// 已存在的 key 不能被"新建"覆盖（历史坑：Save 对 ID=0 走 INSERT 撞唯一索引静默失败）
func TestNextSequenceUpdatesExistingRow(t *testing.T) {
	db := newTestDB(t)

	nextSequence(db, "test_seq", "X", 2, "")
	nextSequence(db, "test_seq", "X", 2, "")

	var rows []model.SystemSetting
	db.Where("key = ?", "test_seq").Find(&rows)
	if len(rows) != 1 {
		t.Fatalf("同一个 key 只应有一行，实际 %d 行", len(rows))
	}
	if rows[0].Value != "2" {
		t.Errorf("序号应写回 2，实际 %q", rows[0].Value)
	}
}

// 库里存了脏值（非数字）时应从 0 重新开始，而不是 panic
func TestNextSequenceRecoversFromGarbageValue(t *testing.T) {
	db := newTestDB(t)
	db.Create(&model.SystemSetting{Key: "test_seq", Value: "not-a-number"})

	got := nextSequence(db, "test_seq", "C", 3, "")

	if got != "C001" {
		t.Errorf("脏值应回退到 1，实际 %q", got)
	}
}

// —— nextReceiveCode / nextQsoRecordCode ——

func TestNextReceiveCodeFormatsDate(t *testing.T) {
	db := newTestDB(t)

	got := nextReceiveCode(db, "2026-11-07")

	if got != "R0001-20261107" {
		t.Errorf("收卡编号格式不对: %q", got)
	}
}

// 历史 bug：receivedDate 长度不足 10 时切片越界 panic
func TestNextReceiveCodeShortDateDoesNotPanic(t *testing.T) {
	db := newTestDB(t)

	for _, d := range []string{"", "2026", "2026-1", "abc"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("nextReceiveCode(%q) panic: %v", d, r)
				}
			}()
			code := nextReceiveCode(db, d)
			if code == "" {
				t.Errorf("nextReceiveCode(%q) 返回空串", d)
			}
		}()
	}
}

func TestNextQsoRecordCodeFormat(t *testing.T) {
	db := newTestDB(t)

	got := nextQsoRecordCode(db)

	if len(got) != len("QSO-000001") {
		t.Errorf("QSO 编号应补零到 6 位，实际 %q", got)
	}
	if got != "QSO-000001" {
		t.Errorf("首个 QSO 编号应为 QSO-000001，实际 %q", got)
	}
}

// 并发建卡 + 建收卡记录混跑，两条序列不能互相干扰
func TestSequencesDoNotInterfere(t *testing.T) {
	db := newTestDB(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); nextQsoRecordCode(db) }()
		go func() { defer wg.Done(); nextReceiveCode(db, "2026-11-07") }()
	}
	wg.Wait()

	var qsoRows, recRows []model.SystemSetting
	db.Where("key = ?", "qso_record_sequence").Find(&qsoRows)
	db.Where("key = ?", "receive_record_sequence").Find(&recRows)

	if len(qsoRows) != 1 || qsoRows[0].Value != "10" {
		t.Errorf("QSO 序列应为 10，实际 %v", qsoRows)
	}
	if len(recRows) != 1 || recRows[0].Value != "10" {
		t.Errorf("收卡序列应为 10，实际 %v", recRows)
	}
}

// 覆盖 format 里 %0*d 的宽度行为：宽度大于实际位数时补零，小于时原样展开
func TestNextSequenceWidthEdgeCases(t *testing.T) {
	db := newTestDB(t)

	cases := []struct {
		prefix string
		width  int
		want   string
	}{
		{"A", 1, "A1"},
		{"B", 5, "B00001"},
		{"C", 0, "C1"},
	}
	for _, tc := range cases {
		got := nextSequence(db, fmt.Sprintf("seq_%s_%d", tc.prefix, tc.width), tc.prefix, tc.width, "")
		if got != tc.want {
			t.Errorf("width=%d 期望 %q，实际 %q", tc.width, tc.want, got)
		}
	}
}
