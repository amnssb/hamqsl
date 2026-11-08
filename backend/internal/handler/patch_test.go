package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 白名单裁剪是这次安全修复的核心：PUT 不再能把任意列传给 GORM。
// 这些用例锁定行为，避免以后有人"顺手"改回 map 直传。

func newJSONContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req, err := http.NewRequest(http.MethodPut, "/", strings.NewReader(body))
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c
}

func TestBindPatchDropsFieldsOutsideWhitelist(t *testing.T) {
	allowed := fieldSet("mail_type", "card_remarks")
	c := newJSONContext(t, `{"mail_type":"ORDINARY","card_remarks":"备注","flow_status":"SIGNED","card_code":"HACKED"}`)

	got := bindPatch(c, allowed)

	if len(got) != 2 {
		t.Fatalf("期望只保留 2 个白名单字段，实际 %d: %v", len(got), got)
	}
	if got["flow_status"] != nil {
		t.Error("flow_status 不在白名单，必须被丢弃——否则一个 PUT 就能把卡片直接改成已签收")
	}
	if got["card_code"] != nil {
		t.Error("card_code 不在白名单，必须被丢弃")
	}
	if got["mail_type"] != "ORDINARY" {
		t.Errorf("mail_type 应保留，实际 %v", got["mail_type"])
	}
}

func TestBindPatchDropsNilValues(t *testing.T) {
	// nil 交给 GORM 会把列写成 NULL，对这些字段来说等价于"没传"
	c := newJSONContext(t, `{"card_remarks":null,"mail_type":"REGISTERED"}`)

	got := bindPatch(c, fieldSet("card_remarks", "mail_type"))

	if _, ok := got["card_remarks"]; ok {
		t.Error("null 值不应进入更新集")
	}
	if len(got) != 1 {
		t.Errorf("期望只剩 1 个字段，实际 %v", got)
	}
}

func TestBindPatchInvalidJSONReturnsEmpty(t *testing.T) {
	c := newJSONContext(t, `{not json`)

	got := bindPatch(c, fieldSet("mail_type"))

	if len(got) != 0 {
		t.Errorf("非法 JSON 应返回空 map，实际 %v", got)
	}
}

func TestBindPatchEmptyBodyReturnsEmpty(t *testing.T) {
	c := newJSONContext(t, `{}`)

	got := bindPatch(c, fieldSet("mail_type"))

	if len(got) != 0 {
		t.Errorf("空对象应返回空 map，实际 %v", got)
	}
}

func TestCardEditableFieldsExcludesFlowColumns(t *testing.T) {
	// 这些列一旦能被 PUT 改写，"卡片流程状态机"就形同虚设
	mustNotContain := []string{
		"card_code", "card_date", "card_time", "flow_status",
		"card_issued", "card_sent", "card_received", "receipt_confirmed",
		"return_mailed_at", "return_received_at", "return_record_code",
		"tracking_status", "tracking_detail", "tracking_updated_at",
		"received_at", "received_record_code",
	}
	for _, col := range mustNotContain {
		if cardEditableFields[col] {
			t.Errorf("cardEditableFields 不应包含 %s", col)
		}
	}

	mustContain := []string{"call_sign", "card_remarks", "mail_type", "mail_target_email"}
	for _, col := range mustContain {
		if !cardEditableFields[col] {
			t.Errorf("cardEditableFields 应包含 %s", col)
		}
	}
}

func TestQsoEditableFieldsExcludesGenerated(t *testing.T) {
	if qsoEditableFields["record_code"] {
		t.Error("record_code 由发号器生成，不能通过 PUT 改写")
	}
	if qsoEditableFields["has_card"] {
		t.Error("has_card 由建卡流程回写，不能通过 PUT 改写")
	}
	if !qsoEditableFields["call_sign"] {
		t.Error("call_sign 应当可编辑")
	}
}

func TestFieldSetIgnoresBlankNames(t *testing.T) {
	got := fieldSet("a", "", "  ", "b")

	if len(got) != 2 || !got["a"] || !got["b"] {
		t.Errorf("空字段名应被忽略，实际 %v", got)
	}
}

// 防一次真实的回归：字段名必须与 GORM 的列名（蛇形）一致，
// 写成驼峰会被 GORM 当成不存在的列。
func TestWhitelistsUseSnakeCase(t *testing.T) {
	sets := map[string]map[string]bool{
		"card":     cardEditableFields,
		"qso":      qsoEditableFields,
		"activity": activityEditableFields,
		"address":  addressBookEditableFields,
		"bureau":   bureauEditableFields,
		"station":  stationProfileEditableFields,
		"cardver":  stationCardEditableFields,
	}
	for name, set := range sets {
		for field := range set {
			if strings.ToLower(field) != field {
				t.Errorf("%s 白名单字段 %q 应为蛇形小写（GORM 列名）", name, field)
			}
		}
	}
}

// 保证白名单本身能被 JSON 序列化使用（字段都是 string 键），
// 顺带验证 map 不是 nil——nil map 上读永远返回 false，会让所有 PUT 都 400。
func TestWhitelistsNotEmpty(t *testing.T) {
	sets := map[string]map[string]bool{
		"card":     cardEditableFields,
		"qso":      qsoEditableFields,
		"activity": activityEditableFields,
		"address":  addressBookEditableFields,
		"bureau":   bureauEditableFields,
		"station":  stationProfileEditableFields,
		"cardver":  stationCardEditableFields,
	}
	for name, set := range sets {
		if len(set) == 0 {
			t.Errorf("%s 白名单为空，所有 PUT 都会返回 400", name)
		}
	}
}

// 白名单裁剪后的 map 必须仍是合法 JSON payload（值类型不被改成奇怪的东西）
func TestBindPatchKeepsValueTypes(t *testing.T) {
	c := newJSONContext(t, `{"return_mail_enabled":true,"available_inventory":12}`)

	got := bindPatch(c, fieldSet("return_mail_enabled", "available_inventory"))

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	if !strings.Contains(string(raw), `"return_mail_enabled":true`) {
		t.Errorf("布尔值被改动了: %s", raw)
	}
	if !strings.Contains(string(raw), `"available_inventory":12`) {
		t.Errorf("数字被改动了: %s", raw)
	}
}
