package handler

import (
	"log"

	"gorm.io/gorm"
	"qsl-management/internal/model"
)

// NormalizeCallSigns 把已入库的呼号统一成大写。
//
// 公开申请入库时会 ToUpper，通联日志和手工建卡以前不会。
// 同一呼号一边是 BI1ABC、一边是 bi1abc，换卡申请就关联不上通联记录，
// 看起来像"建了卡但申请上的创建按钮还在"。只在确实有小写行时才写库。
func NormalizeCallSigns(db *gorm.DB) {
	targets := []struct {
		label string
		model any
	}{
		{"qso_records", &model.QsoRecord{}},
		{"card_records", &model.CardRecord{}},
		{"exchange_requests", &model.ExchangeRequest{}},
		{"receive_records", &model.ReceiveRecord{}},
		{"address_book", &model.AddressBookEntry{}},
		{"station_profiles", &model.StationProfile{}},
	}
	for _, t := range targets {
		var hit int64
		err := db.Model(t.model).
			Where("call_sign <> '' AND call_sign != UPPER(call_sign)").
			Limit(1).Count(&hit).Error
		if err != nil || hit == 0 {
			continue
		}
		res := db.Model(t.model).
			Where("call_sign != UPPER(call_sign)").
			Update("call_sign", gorm.Expr("UPPER(call_sign)"))
		if res.Error != nil {
			log.Printf("[migrate] %s 呼号大写失败: %v", t.label, res.Error)
			continue
		}
		if res.RowsAffected > 0 {
			log.Printf("[migrate] %s: %d 行呼号已转为大写", t.label, res.RowsAffected)
		}
	}
}
