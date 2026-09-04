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

// ExportHandler 数据备份导出：全部业务表打包为 JSON 下载
type ExportHandler struct{ db *gorm.DB }

func NewExportHandler(db *gorm.DB) *ExportHandler { return &ExportHandler{db: db} }

func (h *ExportHandler) All(c *gin.Context) {
	var users []model.User
	var profiles []model.StationProfile
	var cards []model.StationCard
	var equipments []model.StationEquipment
	var qsoRecords []model.QsoRecord
	var cardRecords []model.CardRecord
	var addresses []model.AddressBookEntry
	var bureaus []model.BureauEntry
	var exchanges []model.ExchangeRequest
	var offlineActivities []model.OfflineActivity
	var receives []model.ReceiveRecord
	var settings []model.SystemSetting
	var audits []model.AuditLog

	h.db.Find(&users)
	h.db.Find(&profiles)
	h.db.Find(&cards)
	h.db.Find(&equipments)
	h.db.Find(&qsoRecords)
	h.db.Find(&cardRecords)
	h.db.Find(&addresses)
	h.db.Find(&bureaus)
	h.db.Find(&exchanges)
	h.db.Find(&offlineActivities)
	h.db.Find(&receives)
	h.db.Find(&settings)
	h.db.Find(&audits)

	// 剔除敏感配置（SMTP 密码等），避免明文随备份泄露
	safeSettings := make([]gin.H, 0, len(settings))
	for _, s := range settings {
		v := s.Value
		switch s.Key {
		case "smtp_password", "kuaidi100_key", "kuaidi100_secret":
			v = ""
		}
		safeSettings = append(safeSettings, gin.H{"key": s.Key, "value": v})
	}

	c.Header("Content-Disposition", "attachment; filename=qsl-backup-"+time.Now().Format("20060102-150405")+".json")
	c.JSON(http.StatusOK, gin.H{
		"exported_at": cstNow().Format("2006-01-02 15:04:05"),
		"tables": gin.H{
			"users":              users,
			"station_profiles":   profiles,
			"station_cards":      cards,
			"station_equipments": equipments,
			"qso_records":        qsoRecords,
			"card_records":       cardRecords,
			"address_book":       addresses,
			"bureau_entries":     bureaus,
			"exchange_requests":  exchanges,
			"offline_activities": offlineActivities,
			"receive_records":    receives,
			"system_settings":    safeSettings,
			"audit_logs":         audits,
		},
	})
}

// importSecretKeys 导出时被剔除的敏感配置：导入时从当前库保留，避免恢复备份后邮件功能失效
var importSecretKeys = []string{"smtp_password", "kuaidi100_key", "kuaidi100_secret"}

// Import 导入恢复：接收系统导出的备份 JSON，在单个事务内整库覆盖。
// 登录账号（users）保持现状不覆盖；SMTP 密码等敏感配置保留当前值（备份里本就剔除）。
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

		// 2. 清空业务表（users 除外，登录账号保持现状）
		wipe := []interface{}{
			&model.ReceiveRecord{}, &model.CardRecord{}, &model.QsoRecord{},
			&model.ExchangeRequest{}, &model.OfflineActivity{},
			&model.AddressBookEntry{}, &model.BureauEntry{},
			&model.StationEquipment{}, &model.StationCard{}, &model.StationProfile{},
			&model.SystemSetting{}, &model.AuditLog{},
		}
		for _, m := range wipe {
			if err := tx.Where("id > 0").Delete(m).Error; err != nil {
				return fmt.Errorf("清空数据失败: %v", err)
			}
		}

		// 3. 按备份内容恢复（保留原始 ID，卡↔通联↔申请 关联不断）
		loadRows := func(name string, dst interface{}) (bool, error) {
			rawTable, ok := payload.Tables[name]
			if !ok || string(rawTable) == "null" {
				return false, nil
			}
			if err := json.Unmarshal(rawTable, dst); err != nil {
				return false, fmt.Errorf("解析 %s 失败: %v", name, err)
			}
			return true, nil
		}
		insertAll := func(name string, rows interface{}) error {
			rv := reflectValueSlice(rows)
			for i := 0; i < rv.Len(); i++ {
				if err := tx.Create(rv.Index(i).Addr().Interface()).Error; err != nil {
					return fmt.Errorf("恢复 %s 失败: %v", name, err)
				}
			}
			restored[name] = rv.Len()
			return nil
		}

		var profiles []model.StationProfile
		var cards []model.StationCard
		var equipments []model.StationEquipment
		var qsoRecords []model.QsoRecord
		var exchanges []model.ExchangeRequest
		var cardRecords []model.CardRecord
		var addresses []model.AddressBookEntry
		var bureaus []model.BureauEntry
		var offlineActivities []model.OfflineActivity
		var receives []model.ReceiveRecord
		var audits []model.AuditLog

		steps := []struct {
			name string
			dst  interface{}
		}{
			{"station_profiles", &profiles}, {"station_cards", &cards},
			{"station_equipments", &equipments}, {"qso_records", &qsoRecords},
			{"exchange_requests", &exchanges}, {"card_records", &cardRecords},
			{"address_book", &addresses}, {"bureau_entries", &bureaus},
			{"offline_activities", &offlineActivities}, {"receive_records", &receives},
			{"audit_logs", &audits},
		}
		for _, st := range steps {
			ok, lerr := loadRows(st.name, st.dst)
			if lerr != nil {
				return lerr
			}
			if ok {
				if err := insertAll(st.name, st.dst); err != nil {
					return err
				}
			}
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
	})
}

func isSecretKey(key string) bool {
	for _, k := range importSecretKeys {
		if k == key {
			return true
		}
	}
	return false
}

// reflectValueSlice 取出指向切片的底层值（rows 为 *[]T）
func reflectValueSlice(rows interface{}) reflect.Value {
	return reflect.ValueOf(rows).Elem()
}
