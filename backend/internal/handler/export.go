package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

/*
数据备份导出与导入。

导哪些表、清哪些表，全部由 model.Tables 决定——这里不再列一遍表名，
否则加一张新表就要在三个地方同时改，漏一处就是"备份里没有它"或"导入后旧数据残留"。

两条刻意的例外：
  - users / passkeys 只导出、不导入。口令哈希本来就不进备份，恢复了也登不进去；
    passkeys 的 user_id 在导入后可能悬空。宁可让管理员重新绑定，也不要把自己锁在门外。
  - system_settings 导入时逐条跳过被剔除的敏感键，并回填当前值。

调用示例：

	GET  /api/admin/export                     → 下载 qsl-backup-时间戳.json
	POST /api/admin/import  (multipart: file)  → 单事务覆盖，失败整体回滚
*/
type ExportHandler struct{ db *gorm.DB }

func NewExportHandler(db *gorm.DB) *ExportHandler { return &ExportHandler{db: db} }

// importSecretKeys 导出时被剔除的敏感配置：导入时从当前库保留，避免恢复备份后邮件功能失效。
var importSecretKeys = []string{"smtp_password", "kuaidi100_key", "kuaidi100_secret"}

func isSecretKey(key string) bool {
	for _, k := range importSecretKeys {
		if k == key {
			return true
		}
	}
	return false
}

// createInChunks 把 rows（*[]某张表）按 batch 行一批插入。
// 一批是一条 INSERT，不再一行一次往返。
// 80 行是给字段最多的卡片表留的余量：大约 50 列 × 80 ≈ 4000 个参数，
// 低于 SQLite 单条语句 32766 的上限。调用方已在事务里，这里不再另开一层。
func createInChunks(tx *gorm.DB, rows any, batch int) error {
	rv := reflect.ValueOf(rows).Elem()
	n := rv.Len()
	if n == 0 {
		return nil
	}
	if batch < 1 {
		batch = 80
	}
	for i := 0; i < n; i += batch {
		end := i + batch
		if end > n {
			end = n
		}
		chunk := reflect.MakeSlice(rv.Type(), end-i, end-i)
		reflect.Copy(chunk, rv.Slice(i, end))
		ptr := reflect.New(chunk.Type())
		ptr.Elem().Set(chunk)
		if err := tx.Create(ptr.Interface()).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h *ExportHandler) All(c *gin.Context) {
	tables := gin.H{}
	for _, t := range model.Tables {
		if !t.Backup {
			continue
		}
		if t.Key == "system_settings" {
			// 敏感配置清空后再导出，避免明文随备份泄露
			var settings []model.SystemSetting
			h.db.Find(&settings)
			safe := make([]gin.H, 0, len(settings))
			for _, s := range settings {
				v := s.Value
				if isSecretKey(s.Key) {
					v = ""
				}
				safe = append(safe, gin.H{"key": s.Key, "value": v})
			}
			tables[t.Key] = safe
			continue
		}
		rows := model.NewRows(t)
		h.db.Find(rows)
		tables[t.Key] = rows
	}

	c.Header("Content-Disposition", "attachment; filename=qsl-backup-"+time.Now().Format("20060102-150405")+".json")
	c.JSON(http.StatusOK, gin.H{
		"exported_at": cstNow().Format("2006-01-02 15:04:05"),
		"tables":      tables,
	})
}

// Import 导入恢复：接收系统导出的备份 JSON，在单个事务内整库覆盖。
// 任一步骤失败整体回滚，当前数据不受影响。
func (h *ExportHandler) Import(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, 400, "请选择备份文件（.json）")
		return
	}
	if file.Size > 20<<20 {
		response.Fail(c, 400, "备份文件不能超过 20MB")
		return
	}
	f, err := file.Open()
	if err != nil {
		response.Fail(c, 400, "读取文件失败")
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		response.Fail(c, 400, "读取文件失败")
		return
	}

	var payload struct {
		ExportedAt string                     `json:"exported_at"`
		Tables     map[string]json.RawMessage `json:"tables"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Tables == nil {
		response.Fail(c, 400, "备份文件格式不正确（需为本系统导出的 JSON 文件）")
		return
	}

	restored := gin.H{}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// 1. 记录当前敏感配置（导出已剔除，导入不能弄丢）
		preserved := map[string]string{}
		for _, key := range importSecretKeys {
			var s model.SystemSetting
			if err := tx.Where("key = ?", key).First(&s).Error; err == nil && s.Value != "" {
				preserved[key] = s.Value
			}
		}

		// 2. 清空参与恢复的表（users / passkeys 的 Restore 为 false，登录凭证保持现状）
		for _, t := range model.Tables {
			if !t.Restore {
				continue
			}
			if err := tx.Where("id > 0").Delete(t.Model).Error; err != nil {
				return fmt.Errorf("清空 %s 失败: %v", t.Key, err)
			}
		}

		// 3. 按备份内容恢复（保留原始 ID，卡↔通联↔申请 关联不断）
		for _, t := range model.Tables {
			if !t.Restore || t.Key == "system_settings" {
				continue // system_settings 有敏感键要逐条处理，走下面单独一段
			}
			rawTable, ok := payload.Tables[t.Key]
			if !ok || string(rawTable) == "null" {
				continue
			}
			rows := model.NewRows(t)
			if err := json.Unmarshal(rawTable, rows); err != nil {
				return fmt.Errorf("解析 %s 失败: %v", t.Key, err)
			}
			rv := reflect.ValueOf(rows).Elem()
			// 一批一条 INSERT。整段仍在上面的事务里，任一批失败会连前面的批次一起回滚。
			if err := createInChunks(tx, rows, 80); err != nil {
				return fmt.Errorf("恢复 %s 失败: %v", t.Key, err)
			}
			restored[t.Key] = rv.Len()
		}

		// system_settings：跳过敏感键的空值行；缺敏感键时回填当前值
		if rawSettings, ok := payload.Tables["system_settings"]; ok && string(rawSettings) != "null" {
			var settings []model.SystemSetting
			if err := json.Unmarshal(rawSettings, &settings); err != nil {
				return fmt.Errorf("恢复 system_settings 失败: %v", err)
			}
			for _, s := range settings {
				if s.Value == "" && isSecretKey(s.Key) {
					continue
				}
				if err := tx.Create(&model.SystemSetting{Key: s.Key, Value: s.Value}).Error; err != nil {
					return fmt.Errorf("恢复 system_settings 失败: %v", err)
				}
			}
			restored["system_settings"] = len(settings)
		}
		for key, val := range preserved {
			var cnt int64
			tx.Model(&model.SystemSetting{}).Where("key = ?", key).Count(&cnt)
			if cnt == 0 {
				if err := tx.Create(&model.SystemSetting{Key: key, Value: val}).Error; err != nil {
					return fmt.Errorf("回填敏感配置失败: %v", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		response.Fail(c, 500, "导入失败，已整体回滚，当前数据未受影响："+err.Error())
		return
	}

	response.OK(c, gin.H{
		"restored":     restored,
		"exported_at":  payload.ExportedAt,
		"kept_secrets": len(importSecretKeys) > 0,
		"notice":       "通行密钥与二步验证不会随备份恢复，如为换机请重新绑定",
	})
}
