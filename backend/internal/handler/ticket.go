package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"gorm.io/gorm"

	"qsl-management/internal/model"
)

/*
登录票据：口令已验证通过、第二因子还没过的那几分钟的凭据。

它不是令牌，访问不了任何业务接口，只用来证明"这个人确实知道口令"。
一次性消费、5 分钟过期、可随时作废，所以没做成 JWT——
JWT 一旦签出去就收不回来，而这种半登录状态必须能被立刻掐掉。

调用示例：

	id, err := issueTicket(db, user.ID, "totp", "", ip)
	tk, err := consumeTicket(db, id, "totp")
*/
const ticketTTL = 5 * time.Minute

var errTicketInvalid = errors.New("ticket invalid")

// issueTicket 生成并落库一张票据。sessionData 存 WebAuthn begin 的中间态（TOTP 用不到，传空）。
func issueTicket(db *gorm.DB, userID uint, purpose, sessionData, ip string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(raw)

	t := model.LoginTicket{
		ID:          id,
		UserID:      userID,
		Purpose:     purpose,
		SessionData: sessionData,
		ExpiresAt:   time.Now().Add(ticketTTL),
		IP:          ip,
	}
	if err := db.Create(&t).Error; err != nil {
		return "", err
	}

	// 顺手清掉一小时前就该作废的旧票据，免得表无限涨
	db.Where("expires_at < ?", time.Now().Add(-time.Hour)).Delete(&model.LoginTicket{})
	return id, nil
}

// consumeTicket 取出并用掉一张票据。目的是否匹配、是否过期、是否已被用过，全在这里判。
func consumeTicket(db *gorm.DB, id, purpose string) (*model.LoginTicket, error) {
	if id == "" {
		return nil, errTicketInvalid
	}
	var t model.LoginTicket
	if err := db.Where("id = ?", id).First(&t).Error; err != nil {
		return nil, errTicketInvalid
	}
	if t.Purpose != purpose || t.ConsumedAt != nil || time.Now().After(t.ExpiresAt) {
		return nil, errTicketInvalid
	}

	// 条件更新抢占：并发下同一张票据只能被用掉一次
	res := db.Model(&model.LoginTicket{}).Where("id = ? AND consumed_at IS NULL", id).Update("consumed_at", time.Now())
	if res.Error != nil || res.RowsAffected != 1 {
		return nil, errTicketInvalid
	}
	return &t, nil
}

// peekTicket 只读地取一张票据，用于 WebAuthn begin（那一步还没消费，finish 时才核销）。
func peekTicket(db *gorm.DB, id, purpose string) (*model.LoginTicket, error) {
	if id == "" {
		return nil, errTicketInvalid
	}
	var t model.LoginTicket
	if err := db.Where("id = ?", id).First(&t).Error; err != nil {
		return nil, errTicketInvalid
	}
	if t.Purpose != purpose || t.ConsumedAt != nil || time.Now().After(t.ExpiresAt) {
		return nil, errTicketInvalid
	}
	return &t, nil
}
