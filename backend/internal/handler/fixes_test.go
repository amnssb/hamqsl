package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"qsl-management/internal/model"
)

func jsonContext(method, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	return c, w
}

func TestSanitizeMailAddrRejectsHeaderInjection(t *testing.T) {
	if _, err := sanitizeMailAddr("user@example.com\r\nBcc: evil@example.com"); err == nil {
		t.Fatal("带换行的地址必须拒绝，否则会注入 SMTP 命令")
	}
	if _, err := sanitizeMailAddr("not-an-email"); err == nil {
		t.Fatal("非法地址必须拒绝")
	}
	got, err := sanitizeMailAddr("Ham <user@example.com>")
	if err != nil {
		t.Fatal(err)
	}
	if got != "user@example.com" {
		t.Errorf("应只保留纯地址，实际 %q", got)
	}
}

func TestConfiguredSiteURLRejectsNonHTTP(t *testing.T) {
	if got := configuredSiteURL("javascript:alert(1)"); got != "" {
		t.Errorf("javascript: 不应进入邮件链接，实际 %q", got)
	}
	if got := configuredSiteURL("https://qsl.example.com/app/"); got != "https://qsl.example.com/app" {
		t.Errorf("合法站点地址 = %q", got)
	}
	if got := forwardedProto("https, http"); got != "https" {
		t.Errorf("多级代理应取第一个协议，实际 %q", got)
	}
	if got := forwardedProto("javascript"); got != "" {
		t.Errorf("未知协议应丢弃，实际 %q", got)
	}
}

func TestResetForResendClearsSignedState(t *testing.T) {
	item := model.CardRecord{
		CardIssued: true, CardSent: true, CardReceived: true, ReceiptConfirmed: true,
		FlowStatus: "SIGNED", TrackingNumber: "EA123",
	}
	resetForResend(&item)
	if item.FlowStatus != "PENDING_ISSUE" {
		t.Fatalf("重发后应为待制卡，实际 %s", item.FlowStatus)
	}
	if item.ReceiptConfirmed || item.CardReceived || item.CardSent || item.TrackingNumber != "" {
		t.Fatalf("签收和寄出标记应被清掉: %+v", item)
	}
}

func TestDeleteCardReleasesExchangeAndQso(t *testing.T) {
	db := newTestDB(t)
	qso := model.QsoRecord{RecordCode: "QSO-000001", CallSign: "BI1ABC", Date: "2026-11-01", HasCard: true}
	if err := db.Create(&qso).Error; err != nil {
		t.Fatal(err)
	}
	card := model.CardRecord{CardCode: "C1", CallSign: "BI1ABC", QsoRecordID: &qso.ID, FlowStatus: "SENT", CardSent: true}
	if err := db.Create(&card).Error; err != nil {
		t.Fatal(err)
	}
	ex := model.ExchangeRequest{
		RequestCode: "EX1", CallSign: "BI1ABC", SceneType: "QSO", ReviewStatus: "APPROVED",
		CardCreated: true, CreatedCardID: &card.ID,
	}
	if err := db.Create(&ex).Error; err != nil {
		t.Fatal(err)
	}

	c, w := jsonContext(http.MethodDelete, "", gin.Params{{Key: "id", Value: fmt.Sprint(card.ID)}})
	NewCardHandler(db).Delete(c)
	if w.Code != 200 {
		t.Fatalf("删除失败 %d %s", w.Code, w.Body.String())
	}

	var gotQ model.QsoRecord
	if err := db.First(&gotQ, qso.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotQ.HasCard {
		t.Fatal("删卡后通联仍被标成已建卡，通联换卡里再也选不到")
	}
	var gotEx model.ExchangeRequest
	if err := db.First(&gotEx, ex.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotEx.CardCreated || gotEx.CreatedCardID != nil {
		t.Fatalf("删卡后申请仍占着已建卡: created=%v id=%v", gotEx.CardCreated, gotEx.CreatedCardID)
	}
}

func TestCreateCardMarksQsoOnlyAfterInsert(t *testing.T) {
	db := newTestDB(t)
	qso := model.QsoRecord{RecordCode: "QSO-000001", CallSign: "BI1ABC", Date: "2026-11-01", SceneType: "QSO"}
	if err := db.Create(&qso).Error; err != nil {
		t.Fatal(err)
	}
	ex := model.ExchangeRequest{
		RequestCode: "EX1", CallSign: "BI1ABC", SceneType: "QSO", ReviewStatus: "APPROVED",
		QsoDate: "2026-11-01",
	}
	if err := db.Create(&ex).Error; err != nil {
		t.Fatal(err)
	}
	h := NewExchangeHandler(db)
	card := h.buildCardForExchange(&ex)
	var before model.QsoRecord
	db.First(&before, qso.ID)
	if before.HasCard {
		t.Fatal("卡片还没落库，通联不应被标成已建卡")
	}
	if card.QsoRecordID == nil || *card.QsoRecordID != qso.ID {
		t.Fatalf("应关联通联 %d，实际 %v", qso.ID, card.QsoRecordID)
	}

	c, w := jsonContext(http.MethodPost, "", gin.Params{{Key: "id", Value: fmt.Sprint(ex.ID)}})
	h.CreateCard(c)
	if w.Code != 200 {
		t.Fatalf("建卡失败 %d %s", w.Code, w.Body.String())
	}
	var after model.QsoRecord
	db.First(&after, qso.ID)
	if !after.HasCard {
		t.Fatal("建卡成功后应回写 has_card")
	}
}

func TestConfirmReceiptRejectsUnsentAndRepeat(t *testing.T) {
	db := newTestDB(t)
	card := model.CardRecord{CardCode: "CABC", CallSign: "BI1ABC", FlowStatus: "PENDING_ISSUE"}
	if err := db.Create(&card).Error; err != nil {
		t.Fatal(err)
	}
	c, w := jsonContext(http.MethodPost, `{"card_code":"CABC","received_date":"2026-11-15"}`, nil)
	NewPublicHandler(db).ConfirmReceipt(c)
	if w.Code != 422 {
		t.Fatalf("未寄出应拒绝，实际 %d %s", w.Code, w.Body.String())
	}

	db.Model(&card).Updates(map[string]any{"card_sent": true, "receipt_confirmed": true, "flow_status": "SIGNED"})
	c, w = jsonContext(http.MethodPost, `{"card_code":"CABC","received_date":"2026-11-15"}`, nil)
	NewPublicHandler(db).ConfirmReceipt(c)
	if w.Code != 422 {
		t.Fatalf("重复签收应拒绝，实际 %d %s", w.Code, w.Body.String())
	}
}

func TestSubmitReturnMailRequiresApprovedSWL(t *testing.T) {
	db := newTestDB(t)
	ex := model.ExchangeRequest{RequestCode: "EXQSO", CallSign: "BI1ABC", SceneType: "QSO", ReviewStatus: "APPROVED"}
	if err := db.Create(&ex).Error; err != nil {
		t.Fatal(err)
	}
	c, w := jsonContext(http.MethodPost, `{"request_code":"EXQSO","mail_type":"ORDINARY"}`, nil)
	NewPublicHandler(db).SubmitReturnMail(c)
	if w.Code != 422 {
		t.Fatalf("QSO 申请不应登记 SWL 寄出，实际 %d %s", w.Code, w.Body.String())
	}
}

func TestReceiveCreateRefreshesFlowStatus(t *testing.T) {
	db := newTestDB(t)
	card := model.CardRecord{CardCode: "C1", CallSign: "BI1ABC", CardSent: true, FlowStatus: "SENT"}
	if err := db.Create(&card).Error; err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"call_sign":"bi1abc","received_date":"2026-11-15","outbound_card_id":%d}`, card.ID)
	c, w := jsonContext(http.MethodPost, body, nil)
	NewReceiveHandler(db).Create(c)
	if w.Code != 200 {
		t.Fatalf("收卡失败 %d %s", w.Code, w.Body.String())
	}
	var got model.CardRecord
	db.First(&got, card.ID)
	if got.FlowStatus != "RECEIVED" {
		t.Fatalf("关联收卡后状态应是 RECEIVED，实际 %s", got.FlowStatus)
	}
	var rec model.ReceiveRecord
	if err := db.First(&rec).Error; err != nil {
		t.Fatal(err)
	}
	if rec.CallSign != "BI1ABC" || rec.MatchStatus != "MATCHED" {
		t.Fatalf("收卡记录 = %+v", rec)
	}
}

func TestNormalizeCallSigns(t *testing.T) {
	db := newImportDB(t)
	if err := db.Create(&model.QsoRecord{RecordCode: "QSO-1", CallSign: "bi1abc", Date: "2026-11-01"}).Error; err != nil {
		t.Fatal(err)
	}
	NormalizeCallSigns(db)
	var got model.QsoRecord
	db.First(&got)
	if got.CallSign != "BI1ABC" {
		t.Fatalf("呼号应转成大写，实际 %s", got.CallSign)
	}
}

func TestSaveSmtpIgnoresUnknownKeys(t *testing.T) {
	db := newTestDB(t)
	t.Cleanup(invalidateSettingsCache)
	invalidateSettingsCache()

	c, w := jsonContext(http.MethodPost, `{"smtp_host":"smtp.example.com","plugin_enabled_showcase":"true","site_notice":"hacked"}`, nil)
	NewSettingsHandler(db).SaveSmtp(c)
	if w.Code != 200 {
		t.Fatalf("保存失败 %d %s", w.Code, w.Body.String())
	}
	invalidateSettingsCache()
	if got := getSettingValue(db, "smtp_host"); got != "smtp.example.com" {
		t.Fatalf("smtp_host = %q", got)
	}
	if got := getSettingValue(db, "plugin_enabled_showcase"); got != "" {
		t.Fatalf("未知键被写进了系统设置: %q", got)
	}
	if got := getSettingValue(db, "site_notice"); got != "" {
		t.Fatalf("SMTP 保存不应改站点公告: %q", got)
	}
}

func TestAddressListSizeZeroDoesNotPanic(t *testing.T) {
	db := newImportDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=0&size=0", nil)
	NewAddressHandler(db).ListBook(c)
	if w.Code != 200 {
		t.Fatalf("size=0 不应失败，实际 %d %s", w.Code, w.Body.String())
	}
}
