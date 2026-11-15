package flow

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"qsl-management/internal/model"
)

// 建卡、删卡、SWL 收卡会同时改卡片、通联 has_card、申请 card_created。
// 这三列是缓存：卡片写成功之后，在同一事务里更新；任何一步失败都一起回滚。
// handler 不要自己 Update 这几个字段。

func initCard(card *model.CardRecord) {
	card.CallSign = strings.ToUpper(strings.TrimSpace(card.CallSign))
	card.SceneType = NormScene(card.SceneType)
	setStatus(card, StatusPendingIssue)
}

// CreateManual 卡片记录页直接建卡。QSO 场景如果带了通联 ID，回写 has_card。
// 同呼号、同场景、已通过且还没建卡的申请会被绑上，避免线上换卡再弹出「创建卡片」。
func CreateManual(db *gorm.DB, card *model.CardRecord, at string) error {
	initCard(card)
	if card.SceneType != SceneQSO {
		card.QsoRecordID = nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(card).Error; err != nil {
			return err
		}
		if card.QsoRecordID != nil {
			if err := markQSOHasCard(tx, *card.QsoRecordID); err != nil {
				return err
			}
		}
		return attachExchange(tx, card, at)
	})
}

// CreateFromQSO 从一条还没建卡的通联记录建卡，成功后才把 has_card 置真。
// 这个入口不自动绑定换卡申请，和原来的「通联换卡」按钮一致。
func CreateFromQSO(db *gorm.DB, qso *model.QsoRecord, card *model.CardRecord) error {
	scene := NormScene(qso.SceneType)
	if scene != "" && scene != SceneQSO {
		return reject(CodeQsoScene, "只有通联记录可以这样建卡")
	}
	if qso.HasCard {
		return reject(CodeQsoHasCard, "该通联记录已建卡")
	}
	initCard(card)
	card.SceneType = SceneQSO
	card.CardType = "QSO"
	card.CallSign = strings.ToUpper(strings.TrimSpace(qso.CallSign))
	card.QsoRecordID = &qso.ID
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(card).Error; err != nil {
			return err
		}
		res := tx.Model(&model.QsoRecord{}).Where("id = ? AND has_card = ?", qso.ID, false).Update("has_card", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return reject(CodeQsoHasCard, "该通联记录已建卡")
		}
		qso.HasCard = true
		return nil
	})
}

// CreateFromExchange 从已通过的申请建卡。SWL 必须已经登记过收卡。
// QSO 才会沿用卡片上的通联 ID；其它场景即使调用方填了，也会清掉，避免误标 has_card。
func CreateFromExchange(db *gorm.DB, ex *model.ExchangeRequest, card *model.CardRecord, at string) error {
	if err := canCreateFromExchange(ex); err != nil {
		return err
	}
	initCard(card)
	card.SceneType = NormScene(ex.SceneType)
	if card.SceneType == "" {
		card.SceneType = SceneQSO
	}
	if card.SceneType != SceneQSO {
		card.QsoRecordID = nil
	}
	card.CallSign = strings.ToUpper(strings.TrimSpace(ex.CallSign))
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(card).Error; err != nil {
			return err
		}
		if card.QsoRecordID != nil {
			if err := markQSOHasCard(tx, *card.QsoRecordID); err != nil {
				return err
			}
		}
		if err := markExchangeCard(tx, ex, card.ID, at); err != nil {
			return err
		}
		return nil
	})
}

// DeleteCard 删卡，并在同一事务里解开申请上的「已建卡」和通联上的 has_card。
// 同一条通联还有别的卡时，has_card 保持为真。
func DeleteCard(db *gorm.DB, id uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var card model.CardRecord
		if err := tx.First(&card, id).Error; err != nil {
			return err
		}
		if err := tx.Where("outbound_card_id = ?", card.ID).Delete(&model.ReceiveRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.ExchangeRequest{}).Where("created_card_id = ?", card.ID).
			Updates(map[string]any{
				"card_created":    false,
				"card_created_at": "",
				"created_card_id": nil,
			}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.CardRecord{}, card.ID).Error; err != nil {
			return err
		}
		if card.QsoRecordID == nil {
			return nil
		}
		var left int64
		if err := tx.Model(&model.CardRecord{}).Where("qso_record_id = ?", *card.QsoRecordID).Count(&left).Error; err != nil {
			return err
		}
		if left == 0 {
			if err := tx.Model(&model.QsoRecord{}).Where("id = ?", *card.QsoRecordID).Update("has_card", false).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CommitCardReturn 写入收卡记录，并标记这张发出的卡「回寄已收」。两步同一事务。
func CommitCardReturn(db *gorm.DB, card *model.CardRecord, record *model.ReceiveRecord, at string) error {
	if err := PrepareCardReturn(card); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		res := tx.Model(&model.CardRecord{}).
			Where("id = ? AND return_mailed_at <> '' AND (return_received_at = '' OR return_received_at IS NULL)", card.ID).
			Updates(map[string]any{
				"return_received_at": at,
				"return_record_code": record.ReceiveCode,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return reject(CodeReturnAlready, "该卡回寄已确认过收卡（记录 "+card.ReturnRecordCode+"）")
		}
		card.ReturnReceivedAt = at
		card.ReturnRecordCode = record.ReceiveCode
		return nil
	})
}

// ReceiveSWLReturn 登记 SWL 反寄收卡。收卡记录、申请上的收到时间、回寄卡片和
// card_created / has_card 在同一事务里完成。对方的卡已经建过回寄卡时不再建第二张。
// 返回值表示这次有没有新写入卡片。
func ReceiveSWLReturn(db *gorm.DB, ex *model.ExchangeRequest, record *model.ReceiveRecord, card *model.CardRecord, at string) (bool, error) {
	if err := PrepareSWLReceive(ex); err != nil {
		return false, err
	}
	initCard(card)
	card.QsoRecordID = nil
	card.SceneType = SceneSWL
	wrote := false
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		res := tx.Model(&model.ExchangeRequest{}).
			Where("id = ? AND (return_received_at = '' OR return_received_at IS NULL)", ex.ID).
			Update("return_received_at", at)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return reject(CodeSWLAlreadyRecv, "该申请已登记过收卡")
		}
		ex.ReturnReceivedAt = at
		if ex.CardCreated {
			return nil
		}
		if err := tx.Create(card).Error; err != nil {
			return err
		}
		if err := markExchangeCard(tx, ex, card.ID, at); err != nil {
			return err
		}
		wrote = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return wrote, nil
}

func markQSOHasCard(tx *gorm.DB, id uint) error {
	return tx.Model(&model.QsoRecord{}).Where("id = ?", id).Update("has_card", true).Error
}

func markExchangeCard(tx *gorm.DB, ex *model.ExchangeRequest, cardID uint, at string) error {
	q := tx.Model(&model.ExchangeRequest{}).
		Where("id = ? AND review_status = ? AND card_created = ?", ex.ID, ReviewApproved, false)
	if NormScene(ex.SceneType) == SceneSWL {
		q = q.Where("return_received_at <> ''")
	}
	res := q.Updates(map[string]any{
		"card_created":    true,
		"card_created_at": at,
		"created_card_id": cardID,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return reject(CodeExCardExists, "该申请已创建卡片")
	}
	ex.CardCreated = true
	ex.CardCreatedAt = at
	ex.CreatedCardID = &cardID
	return nil
}

func attachExchange(tx *gorm.DB, card *model.CardRecord, at string) error {
	if card.CallSign == "" || card.SceneType == "" {
		return nil
	}
	var ex model.ExchangeRequest
	err := tx.Where(
		"call_sign = ? AND scene_type = ? AND review_status = ? AND card_created = ?",
		card.CallSign, card.SceneType, ReviewApproved, false,
	).Order("id DESC").First(&ex).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	res := tx.Model(&model.ExchangeRequest{}).
		Where("id = ? AND card_created = ?", ex.ID, false).
		Updates(map[string]any{
			"card_created":    true,
			"card_created_at": at,
			"created_card_id": card.ID,
		})
	if res.Error != nil {
		return res.Error
	}
	// 并发下被别人绑走时，这张卡仍然成立，不把建卡一起回滚。
	return nil
}
