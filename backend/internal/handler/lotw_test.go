package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"qsl-management/internal/model"
)

func useLotwServer(t *testing.T, h http.Handler, timeout time.Duration) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	oldURL, oldTimeout := lotwEndpoint, lotwTimeout
	lotwEndpoint = srv.URL
	if timeout > 0 {
		lotwTimeout = timeout
	}
	t.Cleanup(func() {
		lotwEndpoint = oldURL
		lotwTimeout = oldTimeout
	})
}

func TestSyncLotwImportsConfirmedAndPending(t *testing.T) {
	const secret = "p@ss/word?"
	db := newTestDB(t)
	useLotwServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("login") != "BA4SLT" || q.Get("password") != secret || q.Get("qso_query") != "1" {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		switch q.Get("qso_qsl") {
		case "yes":
			if q.Get("qso_qslsince") != "2024-01-01" {
				t.Errorf("已确认查询应带 qso_qslsince，实际 %q", q.Get("qso_qslsince"))
			}
			io.WriteString(w, "<PROGRAMID:4>LoTW<EOH>"+
				"<CALL:6>JA1ABC<QSO_DATE:8>20261101<TIME_ON:4>1600<BAND:3>20M<FREQ:9>14.074000<MODE:3>FT8<QSL_RCVD:1>Y<QSLRDATE:8>20261102<EOR>")
		case "no":
			if q.Get("qso_qsorxsince") != "2024-01-01" {
				t.Errorf("未确认查询应带 qso_qsorxsince，实际 %q", q.Get("qso_qsorxsince"))
			}
			// 同一条再以未确认出现，也不能把刚才写入的确认打回去。另加一条真正未确认的。
			io.WriteString(w, "<PROGRAMID:4>LoTW<EOH>"+
				"<CALL:6>JA1ABC<QSO_DATE:8>20261101<TIME_ON:4>1600<BAND:3>20M<FREQ:9>14.074000<MODE:3>FT8<QSL_RCVD:1>N<EOR>"+
				"<CALL:5>K1XYZ<QSO_DATE:8>20261103<TIME_ON:4>0100<BAND:3>40M<MODE:2>CW<FREQ:8>7.074000<QSL_RCVD:1>N<EOR>")
		default:
			t.Errorf("意外的 qso_qsl=%q", q.Get("qso_qsl"))
		}
	}), 0)

	got, err := syncLotw(db, "BA4SLT", secret, "2024-01-01")
	if err != nil {
		t.Fatalf("同步失败: %v", err)
	}
	if got.Imported != 2 || got.Updated != 0 || got.Skipped != 1 || got.Failed != 0 {
		t.Fatalf("期望新增 2 跳过 1，实际 %+v", got)
	}
	var ja model.QsoRecord
	db.Where("call_sign = ?", "JA1ABC").First(&ja)
	if !ja.LotwQslRcvd || ja.LotwQslDate != "2026-11-02" || ja.Date != "2026-11-02" || ja.Time != "00:00" {
		t.Fatalf("已确认记录被未确认副本盖掉了: %+v", ja)
	}
	var k1 model.QsoRecord
	db.Where("call_sign = ?", "K1XYZ").First(&k1)
	if k1.LotwQslRcvd || k1.Time != "09:00" {
		t.Fatalf("未确认记录不对: %+v", k1)
	}
	if strings.Contains(gotString(got), secret) {
		t.Fatal("同步结果里出现了密码")
	}
}

func gotString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestFetchLotwRejectsBadPasswordWithoutLeakingIt(t *testing.T) {
	const secret = "s3cret-lotw-pass"
	useLotwServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<html><I>Username/password incorrect</I> "+r.URL.RawQuery+"</html>")
	}), 0)

	_, err := fetchLotwReport("BA4SLT", secret, "2024-01-01", true)
	if err == nil || !strings.Contains(err.Error(), "用户名或密码不正确") {
		t.Fatalf("应指出账号密码错误，实际 %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("错误信息带回了密码: %v", err)
	}
}

func TestFetchLotwRedactsPasswordOnUnknownPage(t *testing.T) {
	const secret = "s3cret-lotw-pass"
	useLotwServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><p>maintenance "+secret+"</p></html>")
	}), 0)

	_, err := fetchLotwReport("BA4SLT", secret, "", false)
	if err == nil || !strings.Contains(err.Error(), "没有返回 ADIF") {
		t.Fatalf("应说明不是 ADIF，实际 %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("未知错误页带回了密码: %v", err)
	}
	if !strings.Contains(err.Error(), "***") {
		t.Fatalf("密码应被抹成 ***，实际 %v", err)
	}
}

func TestFetchLotwTimeout(t *testing.T) {
	useLotwServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		io.WriteString(w, "<EOH>")
	}), 200*time.Millisecond)

	_, err := fetchLotwReport("BA4SLT", "s3cret-lotw-pass", "2024-01-01", true)
	if err == nil || !strings.Contains(err.Error(), "超时") {
		t.Fatalf("应提示超时，实际 %v", err)
	}
	if strings.Contains(err.Error(), "s3cret-lotw-pass") {
		t.Fatalf("超时错误带回了密码: %v", err)
	}
}

func TestLotwSettingsHidePassword(t *testing.T) {
	db := newTestDB(t)
	invalidateSettingsCache()
	t.Cleanup(invalidateSettingsCache)
	h := NewSettingsHandler(db)

	gin.SetMode(gin.TestMode)
	save := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(save)
	body := `{"lotw_username":"BA4SLT","lotw_password":"s3cret-lotw-pass","lotw_since":"2020-01-01"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/api/settings/lotw", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.SaveLotw(c)
	if save.Code != http.StatusOK {
		t.Fatalf("保存失败 %d %s", save.Code, save.Body.String())
	}

	// 留空不覆盖
	again := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(again)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/settings/lotw", strings.NewReader(`{"lotw_username":"BA4SLT","lotw_password":"","lotw_since":"2020-01-01"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.SaveLotw(c)

	var stored model.SystemSetting
	if err := db.Where("key = ?", "lotw_password").First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Value != "s3cret-lotw-pass" {
		t.Fatalf("空密码不应覆盖，实际 %q", stored.Value)
	}

	get := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(get)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/settings/lotw", nil)
	h.GetLotw(c)
	text := get.Body.String()
	if strings.Contains(text, "s3cret-lotw-pass") {
		t.Fatalf("读取设置带回了密码: %s", text)
	}
	var parsed struct {
		Data struct {
			Username string `json:"lotw_username"`
			Password string `json:"lotw_password"`
			Set      bool   `json:"lotw_password_set"`
			Since    string `json:"lotw_since"`
		} `json:"data"`
	}
	if err := json.Unmarshal(get.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Data.Username != "BA4SLT" || parsed.Data.Password != "" || !parsed.Data.Set || parsed.Data.Since != "2020-01-01" {
		t.Fatalf("设置回显不对: %+v", parsed.Data)
	}
}

func TestCreateQsoIgnoresLotwFlag(t *testing.T) {
	db := newTestDB(t)
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/qso-records", strings.NewReader(
		`{"call_sign":"bi1abc","date":"2026-11-01","lotw_qsl_rcvd":true,"lotw_qsl_date":"2026-11-02"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	NewQsoHandler(db).Create(c)

	var got model.QsoRecord
	if err := db.Where("call_sign = ?", "BI1ABC").First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.LotwQslRcvd || got.LotwQslDate != "" {
		t.Fatalf("手工新建不能自标 LoTW 确认: %+v", got)
	}
}
