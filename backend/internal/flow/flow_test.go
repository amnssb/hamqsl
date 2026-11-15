package flow

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qsl-management/internal/model"
)

func newFlowDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
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
	for _, tbl := range []any{
		&model.CardRecord{}, &model.QsoRecord{}, &model.ExchangeRequest{}, &model.ReceiveRecord{},
	} {
		if err := db.AutoMigrate(tbl); err != nil {
			t.Fatalf("建表失败: %v", err)
		}
	}
	return db
}

func cardAt(status string) model.CardRecord {
	card := model.CardRecord{CardCode: "C1", CallSign: "BI1ABC", SceneType: SceneQSO}
	setStatus(&card, status)
	if status == StatusIssued || status == StatusPacked || status == StatusSent || status == StatusReceived || status == StatusSigned {
		card.CardIssuedAt = "2026-11-01 08:00:00"
	}
	if status == StatusSent || status == StatusReceived || status == StatusSigned {
		card.SentAt = "2026-11-02 08:00:00"
		card.TrackingNumber = "EA100"
	}
	if status == StatusReceived || status == StatusSigned {
		card.ReceivedAt = "2026-11-03"
	}
	return card
}

func TestNowUsesShanghai(t *testing.T) {
	if _, offset := Now().Zone(); offset != 8*3600 {
		t.Fatalf("Now 不在东八区，offset=%d", offset)
	}
	// UTC 11-14 16:30 在上海已是 11-15 00:30。日期不能按 UTC 取。
	utc := time.Date(2026, 11, 14, 16, 30, 0, 0, time.UTC)
	if got := Day(utc); got != "2026-11-15" {
		t.Fatalf("上海日期 = %s，想要 2026-11-15", got)
	}
	if got := Stamp(utc); got != "2026-11-15 00:30:00" {
		t.Fatalf("上海时间戳 = %s", got)
	}
}

func TestCardTransitionTable(t *testing.T) {
	at := "2026-11-15 09:00:00"
	legal := []struct {
		name  string
		start string
		run   func(*model.CardRecord) error
		want  string
	}{
		{"issue", StatusPendingIssue, func(c *model.CardRecord) error { return Issue(c, at) }, StatusIssued},
		{"pack", StatusIssued, func(c *model.CardRecord) error { return Pack(c) }, StatusPacked},
		{"send from issued", StatusIssued, func(c *model.CardRecord) error {
			return Send(c, SendInput{At: at, TrackingNumber: "EA9", MailType: "REGISTERED"})
		}, StatusSent},
		{"send from packed", StatusPacked, func(c *model.CardRecord) error {
			return Send(c, SendInput{At: at})
		}, StatusSent},
		{"receive", StatusSent, func(c *model.CardRecord) error {
			return Receive(c, ReceiveInput{At: "2026-11-15", UpdateRemarks: true, Remarks: "收到", RecordCode: "R1"})
		}, StatusReceived},
		{"confirm from sent", StatusSent, func(c *model.CardRecord) error {
			return ConfirmReceipt(c, "2026-11-15", "签收")
		}, StatusSigned},
		{"confirm from received", StatusReceived, func(c *model.CardRecord) error {
			return ConfirmReceipt(c, "2026-11-16", "")
		}, StatusSigned},
		{"resend from signed", StatusSigned, func(c *model.CardRecord) error { return Resend(c) }, StatusPendingIssue},
		{"resend from sent", StatusSent, func(c *model.CardRecord) error { return Resend(c) }, StatusPendingIssue},
	}
	for _, tc := range legal {
		t.Run(tc.name, func(t *testing.T) {
			card := cardAt(tc.start)
			if err := tc.run(&card); err != nil {
				t.Fatalf("合法迁移被拒绝: %v", err)
			}
			if card.FlowStatus != tc.want {
				t.Fatalf("状态 = %s，想要 %s", card.FlowStatus, tc.want)
			}
			if !flagsMatch(&card) {
				t.Fatalf("布尔位和 flow_status 不一致: %+v", card)
			}
		})
	}

	illegal := []struct {
		name  string
		start string
		run   func(*model.CardRecord) error
		code  string
	}{
		{"issue twice", StatusIssued, func(c *model.CardRecord) error { return Issue(c, at) }, CodeCardNotPending},
		{"send before issue", StatusPendingIssue, func(c *model.CardRecord) error {
			return Send(c, SendInput{At: at})
		}, CodeCardNotIssued},
		{"pack before issue", StatusPendingIssue, func(c *model.CardRecord) error { return Pack(c) }, CodeCardNotIssued},
		{"receive before send", StatusIssued, func(c *model.CardRecord) error {
			return Receive(c, ReceiveInput{At: "2026-11-15"})
		}, CodeCardNotSent},
		{"receive twice", StatusReceived, func(c *model.CardRecord) error {
			return Receive(c, ReceiveInput{At: "2026-11-15"})
		}, CodeCardAlreadyRecv},
		{"receive after sign", StatusSigned, func(c *model.CardRecord) error {
			return Receive(c, ReceiveInput{At: "2026-11-15"})
		}, CodeCardAlreadySigned},
		{"confirm before send", StatusPendingIssue, func(c *model.CardRecord) error {
			return ConfirmReceipt(c, "2026-11-15", "")
		}, CodeCardNotSent},
		{"confirm issued", StatusIssued, func(c *model.CardRecord) error {
			return ConfirmReceipt(c, "2026-11-15", "")
		}, CodeCardNotSent},
		{"confirm twice", StatusSigned, func(c *model.CardRecord) error {
			return ConfirmReceipt(c, "2026-11-15", "")
		}, CodeCardAlreadySigned},
	}
	for _, tc := range illegal {
		t.Run(tc.name, func(t *testing.T) {
			card := cardAt(tc.start)
			before := card.FlowStatus
			err := tc.run(&card)
			fe, ok := AsError(err)
			if !ok {
				t.Fatalf("应返回迁移错误，实际 %v", err)
			}
			if fe.Code != tc.code {
				t.Fatalf("错误码 = %s，想要 %s（%s）", fe.Code, tc.code, fe.Message)
			}
			if card.FlowStatus != before {
				t.Fatalf("拒绝后状态被改成 %s", card.FlowStatus)
			}
		})
	}
}

func TestResendClearsSignedFlagsAndTracking(t *testing.T) {
	card := cardAt(StatusSigned)
	card.CreatedMailStatus = "SENT"
	card.SentMailStatus = "SENT"
	card.ReceivedMailStatus = "SENT"
	card.ReceivedRecordCode = "R9"
	if err := Resend(&card); err != nil {
		t.Fatal(err)
	}
	if card.FlowStatus != StatusPendingIssue {
		t.Fatalf("状态 = %s", card.FlowStatus)
	}
	if card.ReceiptConfirmed || card.CardReceived || card.CardSent || card.CardIssued || card.EnvelopePrinted {
		t.Fatalf("进度布尔位应全部为假: %+v", card)
	}
	if card.TrackingNumber != "" || card.SentAt != "" || card.CardIssuedAt != "" || card.ReceivedAt != "" {
		t.Fatalf("步骤时间应清掉: %+v", card)
	}
	if card.CreatedMailStatus != "" || card.SentMailStatus != "" || card.ReceivedMailStatus != "" || card.ReceivedRecordCode != "" {
		t.Fatalf("邮件状态和收卡编号应清掉: %+v", card)
	}
}

func TestSendKeepsPrefilledTrackingWhenRequestOmitsIt(t *testing.T) {
	card := cardAt(StatusIssued)
	card.TrackingNumber = "DRAFT"
	if err := Send(&card, SendInput{At: "2026-11-15 10:00:00", Remarks: ""}); err != nil {
		t.Fatal(err)
	}
	if card.TrackingNumber != "DRAFT" || !card.CardSent || card.FlowStatus != StatusSent {
		t.Fatalf("发信结果不对: %+v", card)
	}
	if card.SentAt != "2026-11-15 10:00:00" || !card.CardIssued {
		t.Fatalf("已寄出应同时是已制卡: %+v", card)
	}
}

func TestPackedSendKeepsEnvelope(t *testing.T) {
	card := cardAt(StatusPacked)
	if err := Send(&card, SendInput{At: "2026-11-15 10:00:00"}); err != nil {
		t.Fatal(err)
	}
	if !card.EnvelopePrinted || card.FlowStatus != StatusSent {
		t.Fatalf("从已打包发信应保留打包标记: %+v", card)
	}
}

func TestAlignRepairsBooleansWithoutClearingDraftTracking(t *testing.T) {
	card := model.CardRecord{
		FlowStatus: StatusPendingIssue, CardSent: true, ReceiptConfirmed: true, TrackingNumber: "KEEP",
	}
	if !Align(&card) {
		t.Fatal("脏布尔位应该被对齐")
	}
	if card.CardSent || card.ReceiptConfirmed || card.FlowStatus != StatusPendingIssue {
		t.Fatalf("对齐结果不对: %+v", card)
	}
	if card.TrackingNumber != "KEEP" {
		t.Fatalf("预先填写的单号被清掉了: %q", card.TrackingNumber)
	}
	if Align(&card) {
		t.Fatal("已经对齐的卡片不应再报有改动")
	}

	sent := model.CardRecord{FlowStatus: StatusSent}
	if !Align(&sent) || !sent.CardSent || !sent.CardIssued || sent.ReceiptConfirmed {
		t.Fatalf("SENT 应推出已制卡且已寄出: %+v", sent)
	}
}

func TestReturnMailRules(t *testing.T) {
	card := cardAt(StatusSent)
	if err := RegisterCardReturn(&card, "ORDINARY", "", "2026-11-15 10:00:00"); err == nil || AsCode(err) != CodeReturnNeedSign {
		t.Fatalf("未签收不应登记回寄: %v", err)
	}
	card = cardAt(StatusSigned)
	if err := RegisterCardReturn(&card, "ORDINARY", "", "2026-11-15 10:00:00"); err == nil || AsCode(err) != CodeReturnDisabled {
		t.Fatalf("未开通不应登记回寄: %v", err)
	}
	SetReturnEnabled(&card, true)
	if err := RegisterCardReturn(&card, "REGISTERED", "EA8", "2026-11-15 11:00:00"); err != nil {
		t.Fatal(err)
	}
	if card.ReturnTracking != "EA8" || card.FlowStatus != StatusSigned {
		t.Fatalf("登记回寄不应改进度: %+v", card)
	}
	fresh := cardAt(StatusSigned)
	if err := PrepareCardReturn(&fresh); err == nil || AsCode(err) != CodeReturnNotMailed {
		t.Fatalf("没登记不能确认收到: %v", err)
	}
	if err := PrepareCardReturn(&card); err != nil {
		t.Fatal(err)
	}
	card.ReturnReceivedAt = "2026-11-16 08:00:00"
	card.ReturnRecordCode = "R2"
	err := PrepareCardReturn(&card)
	if AsCode(err) != CodeReturnAlready || !strings.Contains(err.Error(), "R2") {
		t.Fatalf("重复确认收卡: %v", err)
	}
}

func TestExchangeTransitionTable(t *testing.T) {
	at := "2026-11-15 09:00:00"
	pending := func() model.ExchangeRequest {
		return model.ExchangeRequest{SceneType: SceneQSO, ReviewStatus: ReviewPending, CallSign: "bi1abc"}
	}

	t.Run("approve and reject", func(t *testing.T) {
		ex := pending()
		if err := Approve(&ex, "op", at); err != nil {
			t.Fatal(err)
		}
		if ex.ReviewStatus != ReviewApproved || ex.ReviewedBy != "op" || ex.ReviewedAt != at {
			t.Fatalf("审批结果不对: %+v", ex)
		}
		if err := Approve(&ex, "op", at); AsCode(err) != CodeExNotPending {
			t.Fatalf("重复通过: %v", err)
		}
		if err := Reject(&ex, "不", at); AsCode(err) != CodeExNotPending {
			t.Fatalf("已通过不能再拒绝: %v", err)
		}
		ex = pending()
		if err := Reject(&ex, "资料不符", at); err != nil {
			t.Fatal(err)
		}
		if ex.ReviewStatus != ReviewRejected || ex.ReviewReason != "资料不符" {
			t.Fatalf("拒绝结果不对: %+v", ex)
		}
		if err := Reject(&ex, "再拒", at); AsCode(err) != CodeExAlreadyRejected {
			t.Fatalf("重复拒绝: %v", err)
		}
	})

	t.Run("scene gates", func(t *testing.T) {
		qso := model.ExchangeRequest{SceneType: SceneQSO, ReviewStatus: ReviewApproved}
		if err := SendAddress(&qso, "地址", at); AsCode(err) != CodeNotSWL {
			t.Fatalf("QSO 不能发地址: %v", err)
		}
		if err := MarkCounterpartMailed(&qso, "ORDINARY", "", at); AsCode(err) != CodeNotSWL {
			t.Fatalf("QSO 不能登记寄出: %v", err)
		}
		if AsCode(PrepareSWLReceive(&qso)) != CodeNotSWL {
			t.Fatal("QSO 不能登记收卡")
		}
		if err := canCreateFromExchange(&qso); err != nil {
			t.Fatalf("已通过的 QSO 应能建卡: %v", err)
		}

		swl := model.ExchangeRequest{SceneType: SceneSWL, ReviewStatus: ReviewPending}
		if AsCode(SendAddress(&swl, "地址", at)) != CodeExNotApproved {
			t.Fatal("未通过不能发地址")
		}
		swl.ReviewStatus = ReviewApproved
		if err := SendAddress(&swl, "  某地址  ", at); err != nil {
			t.Fatal(err)
		}
		if swl.ReturnAddressText != "某地址" || swl.AddressSentAt != at {
			t.Fatalf("地址没写上: %+v", swl)
		}
		// 已通过就可以登记，不强制先有地址（公开页在没地址时也给出登记表）。
		if err := MarkCounterpartMailed(&swl, "ORDINARY", "", at); err != nil {
			t.Fatal(err)
		}
		if err := MarkCounterpartMailed(&swl, "REGISTERED", "EA7", at); err != nil {
			t.Fatal(err)
		}
		if AsCode(SendAddress(&swl, "新地址", at)) != CodeSWLAddressClosed {
			t.Fatal("对方已寄出不能再改地址")
		}
		if AsCode(canCreateFromExchange(&swl)) != CodeExSWLWaiting {
			t.Fatal("SWL 没收卡不能建卡")
		}
		if err := PrepareSWLReceive(&swl); err != nil {
			t.Fatal(err)
		}
		swl.ReturnReceivedAt = at
		if err := canCreateFromExchange(&swl); err != nil {
			t.Fatalf("收到对方的卡之后应能建卡: %v", err)
		}
		if AsCode(MarkCounterpartMailed(&swl, "ORDINARY", "", at)) != CodeSWLAlreadyRecv {
			t.Fatal("收卡之后不能再改寄出信息")
		}
		if AsCode(PrepareSWLReceive(&swl)) != CodeSWLAlreadyRecv {
			t.Fatal("不能重复收卡")
		}

		eyeball := model.ExchangeRequest{SceneType: SceneEyeball, ReviewStatus: ReviewApproved}
		if err := canCreateFromExchange(&eyeball); err != nil {
			t.Fatalf("EYEBALL 已通过应能建卡: %v", err)
		}
		if AsCode(SendAddress(&eyeball, "地址", at)) != CodeNotSWL {
			t.Fatal("EYEBALL 不能走 SWL 发地址")
		}
	})

	t.Run("init clears smuggled progress", func(t *testing.T) {
		id := uint(9)
		ex := model.ExchangeRequest{
			ID: 4, CallSign: " bi1abc ", SceneType: "swl",
			ReviewStatus: ReviewApproved, CardCreated: true, CreatedCardID: &id,
			ReturnMailedAt: "2026-11-01", QsoDate: "2026-11-01",
		}
		InitExchange(&ex)
		if ex.ID != 0 || ex.CallSign != "BI1ABC" || ex.SceneType != SceneSWL || ex.ReviewStatus != ReviewPending {
			t.Fatalf("初始化不对: %+v", ex)
		}
		if ex.CardCreated || ex.CreatedCardID != nil || ex.ReturnMailedAt != "" {
			t.Fatalf("进度应被清空: %+v", ex)
		}
		if ex.QsoDate != "2026-11-01" {
			t.Fatal("业务字段不应被初始化清掉")
		}
		qso := model.QsoRecord{CallSign: " bi1abc ", HasCard: true}
		InitQSO(&qso)
		if qso.HasCard || qso.CallSign != "BI1ABC" {
			t.Fatalf("通联初始化不对: %+v", qso)
		}
	})
}

func TestCreateFromExchangeRollsBackWhenInsertFails(t *testing.T) {
	db := newFlowDB(t)
	qso := model.QsoRecord{RecordCode: "QSO-1", CallSign: "BI1ABC", Date: "2026-11-01", SceneType: SceneQSO}
	if err := db.Create(&qso).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CardRecord{CardCode: "C-DUP", CallSign: "X", FlowStatus: StatusPendingIssue}).Error; err != nil {
		t.Fatal(err)
	}
	ex := model.ExchangeRequest{RequestCode: "EX1", CallSign: "BI1ABC", SceneType: SceneQSO, ReviewStatus: ReviewApproved}
	if err := db.Create(&ex).Error; err != nil {
		t.Fatal(err)
	}
	card := model.CardRecord{CardCode: "C-DUP", QsoRecordID: &qso.ID}
	err := CreateFromExchange(db, &ex, &card, "2026-11-15 12:00:00")
	if err == nil {
		t.Fatal("卡号冲突应该失败")
	}
	if _, ok := AsError(err); ok {
		t.Fatalf("卡号冲突是数据库错误，不应伪装成迁移错误: %v", err)
	}
	var gotQ model.QsoRecord
	db.First(&gotQ, qso.ID)
	if gotQ.HasCard {
		t.Fatal("卡片没写成功，has_card 不应被置真")
	}
	var gotEx model.ExchangeRequest
	db.First(&gotEx, ex.ID)
	if gotEx.CardCreated || gotEx.CreatedCardID != nil {
		t.Fatalf("卡片没写成功，申请不应标成已建卡: %+v", gotEx)
	}
}

func TestCreateFromExchangeLinksQSOOnlyAfterInsert(t *testing.T) {
	db := newFlowDB(t)
	qso := model.QsoRecord{RecordCode: "QSO-1", CallSign: "BI1ABC", Date: "2026-11-01", SceneType: SceneQSO}
	if err := db.Create(&qso).Error; err != nil {
		t.Fatal(err)
	}
	ex := model.ExchangeRequest{RequestCode: "EX1", CallSign: "BI1ABC", SceneType: SceneQSO, ReviewStatus: ReviewApproved}
	if err := db.Create(&ex).Error; err != nil {
		t.Fatal(err)
	}
	card := model.CardRecord{CardCode: "C-OK", QsoRecordID: &qso.ID, CardType: SceneQSO}
	if err := CreateFromExchange(db, &ex, &card, "2026-11-15 12:00:00"); err != nil {
		t.Fatal(err)
	}
	if card.FlowStatus != StatusPendingIssue || card.CardIssued || card.CardSent {
		t.Fatalf("新卡应停在待制卡: %+v", card)
	}
	var gotQ model.QsoRecord
	db.First(&gotQ, qso.ID)
	if !gotQ.HasCard {
		t.Fatal("建卡成功后应回写 has_card")
	}
	var gotEx model.ExchangeRequest
	db.First(&gotEx, ex.ID)
	if !gotEx.CardCreated || gotEx.CreatedCardID == nil || *gotEx.CreatedCardID != card.ID {
		t.Fatalf("申请应记下这张卡: %+v", gotEx)
	}
	again := model.CardRecord{CardCode: "C-2", QsoRecordID: &qso.ID}
	err := CreateFromExchange(db, &ex, &again, "2026-11-15 13:00:00")
	if AsCode(err) != CodeExCardExists {
		t.Fatalf("重复建卡: %v", err)
	}
}

func TestSWLCardWaitsForReturnAndDoesNotTouchQSO(t *testing.T) {
	db := newFlowDB(t)
	qso := model.QsoRecord{RecordCode: "QSO-1", CallSign: "BI1ABC", Date: "2026-11-01", SceneType: SceneQSO}
	db.Create(&qso)
	ex := model.ExchangeRequest{
		RequestCode: "EX1", CallSign: "BI1ABC", SceneType: SceneSWL, ReviewStatus: ReviewApproved,
		ReturnMailType: "ORDINARY",
	}
	db.Create(&ex)
	card := model.CardRecord{CardCode: "C-SWL", QsoRecordID: &qso.ID}
	err := CreateFromExchange(db, &ex, &card, "2026-11-15 12:00:00")
	if AsCode(err) != CodeExSWLWaiting {
		t.Fatalf("没收卡就建卡: %v", err)
	}
	var n int64
	db.Model(&model.CardRecord{}).Where("card_code = ?", "C-SWL").Count(&n)
	if n != 0 {
		t.Fatal("被拒绝的建卡不应落库")
	}
	var gotQ model.QsoRecord
	db.First(&gotQ, qso.ID)
	if gotQ.HasCard {
		t.Fatal("SWL 建卡不应碰通联")
	}

	ex.ReturnReceivedAt = "2026-11-15 11:00:00"
	db.Save(&ex)
	if err := CreateFromExchange(db, &ex, &card, "2026-11-15 12:00:00"); err != nil {
		t.Fatal(err)
	}
	if card.QsoRecordID != nil || card.SceneType != SceneSWL {
		t.Fatalf("SWL 卡片不应关联通联: %+v", card)
	}
	db.First(&gotQ, qso.ID)
	if gotQ.HasCard {
		t.Fatal("SWL 建卡成功后也不应把通联标成已建卡")
	}
}

func TestCreateFromQSORejectsSecondCard(t *testing.T) {
	db := newFlowDB(t)
	qso := model.QsoRecord{RecordCode: "QSO-1", CallSign: "bi1abc", Date: "2026-11-01", SceneType: SceneQSO}
	db.Create(&qso)
	card := model.CardRecord{CardCode: "C1", OwnerName: "Op"}
	if err := CreateFromQSO(db, &qso, &card); err != nil {
		t.Fatal(err)
	}
	if !qso.HasCard || card.CallSign != "BI1ABC" || card.SceneType != SceneQSO || card.FlowStatus != StatusPendingIssue {
		t.Fatalf("通联建卡结果不对: card=%+v qso=%+v", card, qso)
	}
	second := model.CardRecord{CardCode: "C2"}
	if AsCode(CreateFromQSO(db, &qso, &second)) != CodeQsoHasCard {
		t.Fatal("已建卡的通联应拒绝")
	}
	var n int64
	db.Model(&model.CardRecord{}).Count(&n)
	if n != 1 {
		t.Fatalf("拒绝后不应多出卡片，实际 %d", n)
	}

	swl := model.QsoRecord{RecordCode: "QSO-2", CallSign: "BI1XYZ", Date: "2026-11-02", SceneType: SceneSWL}
	db.Create(&swl)
	if AsCode(CreateFromQSO(db, &swl, &model.CardRecord{CardCode: "C3"})) != CodeQsoScene {
		t.Fatal("非通联记录不能走这个入口")
	}
}

func TestManualCreateLinksExchangeInSameTransaction(t *testing.T) {
	db := newFlowDB(t)
	qso := model.QsoRecord{RecordCode: "QSO-1", CallSign: "BI1ABC", Date: "2026-11-01"}
	db.Create(&qso)
	ex := model.ExchangeRequest{
		RequestCode: "EX1", CallSign: "BI1ABC", SceneType: SceneQSO, ReviewStatus: ReviewApproved,
	}
	db.Create(&ex)
	card := model.CardRecord{
		CardCode: "C1", CallSign: "bi1abc", SceneType: "qso", QsoRecordID: &qso.ID,
	}
	if err := CreateManual(db, &card, "2026-11-15 12:00:00"); err != nil {
		t.Fatal(err)
	}
	var gotQ model.QsoRecord
	db.First(&gotQ, qso.ID)
	if !gotQ.HasCard {
		t.Fatal("手动建卡应回写 has_card")
	}
	var gotEx model.ExchangeRequest
	db.First(&gotEx, ex.ID)
	if !gotEx.CardCreated || gotEx.CreatedCardID == nil || *gotEx.CreatedCardID != card.ID {
		t.Fatalf("应绑上已通过的申请: %+v", gotEx)
	}
	if card.CallSign != "BI1ABC" || card.SceneType != SceneQSO || card.FlowStatus != StatusPendingIssue {
		t.Fatalf("卡片身份应被规范化: %+v", card)
	}
}

func TestManualCreateRollsBackLinkWhenInsertFails(t *testing.T) {
	db := newFlowDB(t)
	db.Create(&model.CardRecord{CardCode: "C-DUP", CallSign: "X", FlowStatus: StatusPendingIssue})
	qso := model.QsoRecord{RecordCode: "QSO-1", CallSign: "BI1ABC", Date: "2026-11-01"}
	db.Create(&qso)
	ex := model.ExchangeRequest{RequestCode: "EX1", CallSign: "BI1ABC", SceneType: SceneQSO, ReviewStatus: ReviewApproved}
	db.Create(&ex)
	err := CreateManual(db, &model.CardRecord{CardCode: "C-DUP", CallSign: "BI1ABC", SceneType: SceneQSO, QsoRecordID: &qso.ID}, "2026-11-15 12:00:00")
	if err == nil {
		t.Fatal("卡号冲突应该失败")
	}
	var gotQ model.QsoRecord
	db.First(&gotQ, qso.ID)
	if gotQ.HasCard {
		t.Fatal("插入失败时 has_card 必须回滚")
	}
	var gotEx model.ExchangeRequest
	db.First(&gotEx, ex.ID)
	if gotEx.CardCreated {
		t.Fatal("插入失败时不应绑申请")
	}
}

func TestDeleteCardRestoresQSOAndExchange(t *testing.T) {
	db := newFlowDB(t)
	qso := model.QsoRecord{RecordCode: "QSO-1", CallSign: "BI1ABC", Date: "2026-11-01", HasCard: true}
	db.Create(&qso)
	card := model.CardRecord{CardCode: "C1", CallSign: "BI1ABC", QsoRecordID: &qso.ID, FlowStatus: StatusSent, CardSent: true}
	db.Create(&card)
	other := model.CardRecord{CardCode: "C2", CallSign: "BI1ABC", QsoRecordID: &qso.ID, FlowStatus: StatusPendingIssue}
	db.Create(&other)
	ex := model.ExchangeRequest{RequestCode: "EX1", CallSign: "BI1ABC", ReviewStatus: ReviewApproved, CardCreated: true, CreatedCardID: &card.ID}
	db.Create(&ex)
	recv := model.ReceiveRecord{ReceiveCode: "R1", CallSign: "BI1ABC", ReceivedDate: "2026-11-15", OutboundCardID: &card.ID}
	db.Create(&recv)

	if err := DeleteCard(db, card.ID); err != nil {
		t.Fatal(err)
	}
	var gotQ model.QsoRecord
	db.First(&gotQ, qso.ID)
	if !gotQ.HasCard {
		t.Fatal("还有另一张卡时不应清掉 has_card")
	}
	var gotEx model.ExchangeRequest
	db.First(&gotEx, ex.ID)
	if gotEx.CardCreated || gotEx.CreatedCardID != nil {
		t.Fatalf("删卡应解开申请: %+v", gotEx)
	}
	var n int64
	db.Model(&model.ReceiveRecord{}).Count(&n)
	if n != 0 {
		t.Fatal("指向该卡的收卡记录应一并删除")
	}

	if err := DeleteCard(db, other.ID); err != nil {
		t.Fatal(err)
	}
	db.First(&gotQ, qso.ID)
	if gotQ.HasCard {
		t.Fatal("最后一张卡删掉后应恢复 has_card")
	}
}

func TestReceiveSWLReturnWritesCardAndRollsBackTogether(t *testing.T) {
	db := newFlowDB(t)
	ex := model.ExchangeRequest{
		RequestCode: "EX1", CallSign: "BI1ABC", SceneType: SceneSWL, ReviewStatus: ReviewApproved,
		ReturnMailType: "REGISTERED", ReturnTracking: "EA1", ReturnMailedAt: "2026-11-14 10:00:00",
		Email: "a@example.com", Name: "Ham",
	}
	db.Create(&ex)
	db.Create(&model.CardRecord{CardCode: "C-DUP", CallSign: "X", FlowStatus: StatusPendingIssue})

	record := model.ReceiveRecord{ReceiveCode: "R1", CallSign: "BI1ABC", ReceivedDate: "2026-11-15", CardType: "SWL", BusinessType: "SWL"}
	card := model.CardRecord{CardCode: "C-DUP", CallSign: "BI1ABC", CardType: SceneSWL, SceneType: SceneSWL}
	if _, err := ReceiveSWLReturn(db, &ex, &record, &card, "2026-11-15 12:00:00"); err == nil {
		t.Fatal("卡号冲突时应整单回滚")
	}
	var got model.ExchangeRequest
	db.First(&got, ex.ID)
	if got.ReturnReceivedAt != "" || got.CardCreated {
		t.Fatalf("失败时申请进度不应前进: %+v", got)
	}
	var n int64
	db.Model(&model.ReceiveRecord{}).Count(&n)
	if n != 0 {
		t.Fatal("失败时不应留下收卡记录")
	}

	// 重新读一份干净的申请，卡号换掉后再收。
	db.First(&ex, ex.ID)
	record = model.ReceiveRecord{ReceiveCode: "R2", CallSign: "BI1ABC", ReceivedDate: "2026-11-15", CardType: "SWL", BusinessType: "SWL"}
	card = model.CardRecord{CardCode: "C-OK", CallSign: "BI1ABC", CardType: SceneSWL, SceneType: SceneSWL, MailTargetEmail: ex.Email}
	wrote, err := ReceiveSWLReturn(db, &ex, &record, &card, "2026-11-15 12:00:00")
	if err != nil || !wrote {
		t.Fatalf("收卡建卡失败 wrote=%v err=%v", wrote, err)
	}
	if card.FlowStatus != StatusPendingIssue || card.SceneType != SceneSWL {
		t.Fatalf("回寄卡应待制卡: %+v", card)
	}
	db.First(&got, ex.ID)
	if got.ReturnReceivedAt == "" || !got.CardCreated || got.CreatedCardID == nil || *got.CreatedCardID != card.ID {
		t.Fatalf("申请应记下收卡和卡片: %+v", got)
	}

	// 已经建过卡时，再次登记应收拒，并且不再生出第二张卡。
	db.First(&ex, ex.ID)
	record = model.ReceiveRecord{ReceiveCode: "R3", CallSign: "BI1ABC", ReceivedDate: "2026-11-16"}
	if _, err := ReceiveSWLReturn(db, &ex, &record, &model.CardRecord{CardCode: "C-2"}, "2026-11-16 12:00:00"); AsCode(err) != CodeSWLAlreadyRecv {
		t.Fatalf("重复收卡: %v", err)
	}
	db.Model(&model.CardRecord{}).Where("card_code = ?", "C-2").Count(&n)
	if n != 0 {
		t.Fatal("重复收卡不应再建卡")
	}
}

func TestCommitCardReturnIsAtomic(t *testing.T) {
	db := newFlowDB(t)
	card := model.CardRecord{
		CardCode: "C1", CallSign: "BI1ABC", FlowStatus: StatusSigned,
		CardSent: true, ReceiptConfirmed: true, ReturnMailEnabled: true,
		ReturnMailType: "ORDINARY", ReturnMailedAt: "2026-11-15 09:00:00",
	}
	db.Create(&card)
	record := model.ReceiveRecord{ReceiveCode: "R1", CallSign: "BI1ABC", ReceivedDate: "2026-11-15"}
	if err := CommitCardReturn(db, &card, &record, "2026-11-15 18:00:00"); err != nil {
		t.Fatal(err)
	}
	var got model.CardRecord
	db.First(&got, card.ID)
	if got.ReturnReceivedAt == "" || got.ReturnRecordCode != "R1" {
		t.Fatalf("回寄应收下: %+v", got)
	}
	if AsCode(CommitCardReturn(db, &got, &model.ReceiveRecord{ReceiveCode: "R2", CallSign: "BI1ABC", ReceivedDate: "2026-11-16"}, "2026-11-16 18:00:00")) != CodeReturnAlready {
		t.Fatal("不能重复确认回寄")
	}
	var n int64
	db.Model(&model.ReceiveRecord{}).Count(&n)
	if n != 1 {
		t.Fatalf("重复确认不应再写收卡记录，实际 %d", n)
	}
}

func TestPendingCreateAndUnapprovedExchange(t *testing.T) {
	db := newFlowDB(t)
	ex := model.ExchangeRequest{RequestCode: "EX1", CallSign: "BI1ABC", SceneType: SceneQSO, ReviewStatus: ReviewPending}
	db.Create(&ex)
	err := CreateFromExchange(db, &ex, &model.CardRecord{CardCode: "C1"}, "2026-11-15 12:00:00")
	if AsCode(err) != CodeExNotApproved {
		t.Fatalf("未通过不能建卡: %v", err)
	}
	var n int64
	db.Model(&model.CardRecord{}).Count(&n)
	if n != 0 {
		t.Fatal("拒绝后不应有卡片")
	}
}

func flagsMatch(card *model.CardRecord) bool {
	switch card.FlowStatus {
	case StatusPendingIssue:
		return !card.CardIssued && !card.EnvelopePrinted && !card.CardSent && !card.CardReceived && !card.ReceiptConfirmed
	case StatusIssued:
		return card.CardIssued && !card.EnvelopePrinted && !card.CardSent && !card.CardReceived && !card.ReceiptConfirmed
	case StatusPacked:
		return card.CardIssued && card.EnvelopePrinted && !card.CardSent && !card.CardReceived && !card.ReceiptConfirmed
	case StatusSent:
		return card.CardIssued && card.CardSent && !card.CardReceived && !card.ReceiptConfirmed
	case StatusReceived:
		return card.CardIssued && card.CardSent && card.CardReceived && !card.ReceiptConfirmed
	case StatusSigned:
		return card.CardIssued && card.CardSent && card.CardReceived && card.ReceiptConfirmed
	default:
		return false
	}
}

func AsCode(err error) string {
	fe, ok := AsError(err)
	if !ok {
		return ""
	}
	return fe.Code
}
