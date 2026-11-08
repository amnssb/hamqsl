package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

/*
部分更新（PUT）的字段白名单。

原来的写法是把请求体直接当 map 交给 GORM：

	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	h.db.Model(&item).Updates(req)

等于把整张表的所有列都开放给调用方——card_code、flow_status、
receipt_confirmed、return_received_at 这些本该只由状态机推进的字段，
一个 PUT 就能改写。

现在统一走 bindPatch：只挑白名单里出现过的键，其余一律丢弃，
并顺手挡掉 nil 值（nil 会让 GORM 把列写成 NULL）。

调用示例：

	if !applyPatch(c, h.db, &item, cardEditableFields) { ... }
	h.db.First(&item, item.ID)   // Updates(map) 不回填结构体，重新读一次
*/

// bindPatch 解析 JSON 请求体并按白名单裁剪，返回可直接交给 GORM Updates 的 map。
//
// 解析失败返回空 map（调用方按"没有可更新字段"处理）。
func bindPatch(c *gin.Context, allowed map[string]bool) map[string]interface{} {
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		// nil 交给 GORM 会把列写成 NULL；对这些字段来说 NULL 和空串等价，
		// 一律按"没传"处理，避免前端顺手带上一个 null 把备注清成 NULL。
		if v == nil || !allowed[k] {
			continue
		}
		out[k] = v
	}
	return out
}

// fieldSet 把字段名列表转成白名单集合，让调用点读起来就是一行清单。
func fieldSet(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			m[n] = true
		}
	}
	return m
}

// applyPatch 按白名单写入，写入成功后重新读一遍。
//
// 两个返回值为 (是否写入了合法字段, 错误)：
//   - (false, nil)  请求体里没有白名单内的字段 → 调用方回 400
//   - (false, err)  写库失败             → 调用方回 500
//   - (true,  nil)  成功，item 已是最新值
//
// 写入后必须重读：GORM 的 Updates(map) 不保证回填结构体字段，
// 直接返回 item 可能把旧值当成"保存结果"交给前端。
func applyPatch(c *gin.Context, db *gorm.DB, item interface{}, id string, allowed map[string]bool) (bool, error) {
	body := bindPatch(c, allowed)
	if len(body) == 0 {
		return false, nil
	}
	if err := db.Model(item).Updates(body).Error; err != nil {
		return false, err
	}
	if err := db.First(item, id).Error; err != nil {
		return true, err
	}
	return true, nil
}

// cardEditableFields 卡片记录允许通过 PUT 修改的字段。
//
// 刻意不含：
//   - card_code / card_date / card_time —— 建单时定下的身份与时间戳
//   - flow_status / card_issued / card_sent / card_received / receipt_confirmed
//     —— 由 issue/sent/received/resend/公共确认收件这些流程端点推进
//   - return_mailed_at / return_received_at / return_record_code —— 由回寄登记与收卡确认写入
//   - tracking_status / tracking_detail / tracking_updated_at —— 由快递查询写入
//   - created_mail_status / sent_mail_status / received_mail_status —— 由发信结果写入
//   - received_record_code / received_at —— 由收卡登记写入
var cardEditableFields = fieldSet(
	"call_sign", "owner_name", "card_type", "scene_type", "card_version",
	"qso_record_id", "offline_activity_id", "address_entry_id", "bureau_id",
	"business_remarks", "card_remarks", "sent_remarks", "received_remarks",
	"mail_type", "tracking_number", "tracking_carrier", "mail_target_email",
	"return_mail_enabled", "return_mail_type", "return_tracking",
)

// qsoEditableFields 通联记录允许通过 PUT 修改的字段。
//
// 不含 record_code（发号器生成）与 has_card（建卡时回写）。
var qsoEditableFields = fieldSet(
	"scene_type", "date", "time", "timezone", "freq", "band", "mode",
	"my_rig", "my_rig_mode", "my_rig_ant", "my_rig_pwr", "my_qth", "operator",
	"call_sign", "rig", "ant", "pwr", "qth", "rst_sent", "rst_rcvd", "remarks",
)

// activityEditableFields 线下换卡活动允许通过 PUT 修改的字段。
//
// 不含 activity_code（发号器生成）。
var activityEditableFields = fieldSet(
	"activity_name", "activity_location", "activity_date", "activity_time",
	"card_remarks", "workflow_status",
)

// addressBookEditableFields 地址簿允许通过 PUT 修改的字段。
var addressBookEditableFields = fieldSet(
	"call_sign", "name", "telephone", "postal_code", "destination_country",
	"address", "email", "address_remarks",
)

// bureauEditableFields 卡片局允许通过 PUT 修改的字段。
var bureauEditableFields = fieldSet(
	"bureau_name", "telephone", "postal_code", "destination_country",
	"address", "address_remarks",
)

// stationProfileEditableFields 台站档案允许通过 PUT 修改的字段。
//
// 不含 call_sign：它是唯一索引，也是公开页"回寄给谁"的依据。
var stationProfileEditableFields = fieldSet(
	"name", "name_en", "telephone", "postal_code", "address", "address_en",
	"email", "remarks",
)

// stationCardEditableFields 卡片版本允许通过 PUT 修改的字段。
//
// 不含 card_version（唯一索引，已发出的卡都引用它）与 created_at。
var stationCardEditableFields = fieldSet(
	"image_path", "available_inventory", "version_total", "sort_order",
	"qso_only", "remarks", "is_active",
)
