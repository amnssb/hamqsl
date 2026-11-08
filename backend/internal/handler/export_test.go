package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qsl-management/internal/model"
)

// newImportDB 跟生产一样关掉 GORM 的默认事务。
// 备份导入自己有一层事务；如果分批插入偷偷跑到事务外面，失败时删掉的旧数据就找不回来了。
func newImportDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	for _, tbl := range model.Tables {
		if err := db.AutoMigrate(tbl.Model); err != nil {
			t.Fatalf("建表 %s 失败: %v", tbl.Key, err)
		}
	}
	return db
}

func postImport(t *testing.T, db *gorm.DB, payload string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", "backup.json")
	if err != nil {
		t.Fatalf("构造上传失败: %v", err)
	}
	if _, err := fw.Write([]byte(payload)); err != nil {
		t.Fatalf("写入上传失败: %v", err)
	}
	w.Close()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/import", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	c.Request = req
	NewExportHandler(db).Import(c)
	return rec
}

func TestImportRollsBackWhenARowFails(t *testing.T) {
	db := newImportDB(t)
	db.Create(&model.QsoRecord{RecordCode: "KEEP", CallSign: "KEEP", Date: "2026-01-01"})

	// 两条记录编号相同，第二条会撞唯一索引。整次导入必须回滚，原来的 KEEP 还在。
	payload := `{"exported_at":"t","tables":{"qso_records":[
		{"record_code":"QSO-000001","call_sign":"AAA","date":"2026-11-01"},
		{"record_code":"QSO-000001","call_sign":"BBB","date":"2026-11-02"}
	]}}`
	rec := postImport(t, db, payload)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("冲突的备份应导入失败，实际 HTTP %d body=%s", rec.Code, rec.Body.String())
	}

	var rows []model.QsoRecord
	db.Find(&rows)
	if len(rows) != 1 || rows[0].CallSign != "KEEP" {
		t.Fatalf("失败后应原样保留 KEEP，实际 %+v", rows)
	}
}

func TestImportRestoresAcrossBatches(t *testing.T) {
	db := newImportDB(t)

	const n = 90 // 超过 createInChunks 的 80，第二批也必须写进去
	rows := make([]map[string]any, n)
	for i := 1; i <= n; i++ {
		rows[i-1] = map[string]any{
			"id":          i,
			"record_code": fmt.Sprintf("QSO-%06d", i),
			"call_sign":   fmt.Sprintf("C%05d", i),
			"date":        "2026-11-01",
		}
	}
	raw, err := json.Marshal(map[string]any{
		"exported_at": "t",
		"tables":      map[string]any{"qso_records": rows},
	})
	if err != nil {
		t.Fatalf("构造备份失败: %v", err)
	}
	rec := postImport(t, db, string(raw))
	if rec.Code != http.StatusOK {
		t.Fatalf("导入应成功，实际 HTTP %d body=%s", rec.Code, rec.Body.String())
	}

	var count int64
	db.Model(&model.QsoRecord{}).Count(&count)
	if count != n {
		t.Fatalf("应恢复 %d 行，实际 %d", n, count)
	}
	var last model.QsoRecord
	db.First(&last, n)
	if last.CallSign != "C00090" || last.RecordCode != "QSO-000090" {
		t.Fatalf("第 90 行没按原 ID 写回: %+v", last)
	}
}

func TestImportKeepsSecretWhenBackupBlanksIt(t *testing.T) {
	db := newImportDB(t)
	db.Create(&model.SystemSetting{Key: "smtp_password", Value: "s3cret"})
	db.Create(&model.SystemSetting{Key: "site_name", Value: "OLD"})

	payload := `{"exported_at":"t","tables":{"system_settings":[
		{"key":"smtp_password","value":""},
		{"key":"site_name","value":"NEW"}
	]}}`
	rec := postImport(t, db, payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("导入应成功，实际 HTTP %d body=%s", rec.Code, rec.Body.String())
	}

	var pass, name model.SystemSetting
	db.Where("key = ?", "smtp_password").First(&pass)
	db.Where("key = ?", "site_name").First(&name)
	if pass.Value != "s3cret" {
		t.Errorf("备份里的空密码不应覆盖现有密码，实际 %q", pass.Value)
	}
	if name.Value != "NEW" {
		t.Errorf("非敏感配置应被备份覆盖，实际 %q", name.Value)
	}
}
