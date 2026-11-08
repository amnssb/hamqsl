package statsdaily

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qsl-management/internal/model"
)

func TestAggregateCardDaysBucketsByCST(t *testing.T) {
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
	if err := db.AutoMigrate(&model.CardRecord{}); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	now := time.Date(2026, 11, 8, 15, 0, 0, 0, cstZone)
	today := time.Date(2026, 11, 8, 9, 30, 0, 0, cstZone)
	yesterday := time.Date(2026, 11, 7, 23, 0, 0, 0, cstZone)
	older := time.Date(2026, 11, 1, 12, 0, 0, 0, cstZone)

	db.Create(&model.CardRecord{CardCode: "C1", CallSign: "A", CreatedAt: today, CardIssued: true, CardSent: true})
	db.Create(&model.CardRecord{CardCode: "C2", CallSign: "B", CreatedAt: yesterday, ReceiptConfirmed: true})
	db.Create(&model.CardRecord{CardCode: "C3", CallSign: "C", CreatedAt: older, CardIssued: true})

	items := aggregateCardDays(db, 2, now)
	if len(items) != 2 {
		t.Fatalf("应返回 2 天，实际 %d", len(items))
	}
	if items[0].Day != "2026-11-07" || items[1].Day != "2026-11-08" {
		t.Fatalf("日期顺序不对: %+v", items)
	}
	if items[0].Created != 1 || items[0].Signed != 1 || items[0].Issued != 0 {
		t.Errorf("11-07 应为建卡 1、签收 1，实际 %+v", items[0])
	}
	if items[1].Created != 1 || items[1].Issued != 1 || items[1].Sent != 1 || items[1].Signed != 0 {
		t.Errorf("11-08 应为建卡 1、制卡 1、寄出 1，实际 %+v", items[1])
	}
}
