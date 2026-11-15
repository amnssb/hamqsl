package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/pkg/response"
)

// LoTW 只读查询。密码放在 query 里，这是 ARRL 这个接口的用法；
// 因此任何错误文本都要先把密码抹掉，再返回给调用方或写进日志。
//
// 文档：https://lotw.arrl.org/lotw-help/developer-query-qsos-qsls/
// qso_qsl=yes 只返回已确认，过滤参数是 qso_qslsince。
// qso_qsl=no 返回未确认的已上传记录，过滤参数是 qso_qsorxsince。
// 两个参数在对方为空时会落到 LoTW 自己记住的上次查询水位，
// 其他软件下载过就会让这里第一次同步变成 0 条。留空时改用一个足够早的日期，全量拉取，靠去重消化重复。

const (
	lotwFullSince = "1930-01-01"
	lotwMaxBody   = 32 << 20
)

var (
	lotwEndpoint = "https://lotw.arrl.org/lotwuser/lotwreport.adi"
	lotwTimeout  = 90 * time.Second
)

func (h *SettingsHandler) GetLotw(c *gin.Context) {
	pass := h.getSetting("lotw_password")
	response.OK(c, gin.H{
		"lotw_username":     h.getSetting("lotw_username"),
		"lotw_password":     "",
		"lotw_password_set": pass != "",
		"lotw_since":        h.getSetting("lotw_since"),
	})
}

func (h *SettingsHandler) SaveLotw(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	if raw, ok := req["lotw_username"]; ok {
		user, err := cleanLotwUser(asString(raw))
		if err != nil {
			response.Fail(c, 400, err.Error())
			return
		}
		h.setSetting("lotw_username", user)
	}
	if raw, ok := req["lotw_since"]; ok {
		since := strings.TrimSpace(asString(raw))
		if since != "" {
			if _, err := time.Parse("2006-01-02", since); err != nil {
				response.Fail(c, 400, "起始日期格式应为 YYYY-MM-DD")
				return
			}
		}
		h.setSetting("lotw_since", since)
	}
	if raw, ok := req["lotw_password"]; ok {
		pass := asString(raw)
		if pass != "" && pass != "***" {
			if err := cleanLotwPassword(pass); err != nil {
				response.Fail(c, 400, err.Error())
				return
			}
			h.setSetting("lotw_password", pass)
		}
	}
	response.OKMsg(c, "LoTW 配置已保存")
}

// SyncLotw 用已保存的账号拉取已确认和未确认两份 ADIF，再走同一套导入。
func (h *SettingsHandler) SyncLotw(c *gin.Context) {
	user := strings.TrimSpace(h.getSetting("lotw_username"))
	pass := h.getSetting("lotw_password")
	since := strings.TrimSpace(h.getSetting("lotw_since"))
	if user == "" || pass == "" {
		response.Fail(c, 422, "请先保存 LoTW 用户名和密码")
		return
	}
	result, err := syncLotw(h.db, user, pass, since)
	if err != nil {
		// 不用 401：前端会把 401 当成登录过期并清掉会话。
		response.Fail(c, 422, err.Error())
		return
	}
	response.OK(c, result)
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func cleanLotwUser(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.ContainsAny(s, "\r\n\t") {
		return "", errors.New("LoTW 用户名无效")
	}
	if len(s) > 64 {
		return "", errors.New("LoTW 用户名过长")
	}
	return s, nil
}

func cleanLotwPassword(s string) error {
	if len(s) > 200 {
		return errors.New("LoTW 密码过长")
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return errors.New("LoTW 密码无效")
		}
	}
	return nil
}

func syncLotw(db *gorm.DB, user, pass, since string) (adifImportResult, error) {
	user = strings.TrimSpace(user)
	since = strings.TrimSpace(since)
	if user == "" || pass == "" {
		return adifImportResult{}, errors.New("请先保存 LoTW 用户名和密码")
	}
	if since == "" {
		since = lotwFullSince
	}
	if _, err := time.Parse("2006-01-02", since); err != nil {
		return adifImportResult{}, errors.New("起始日期格式应为 YYYY-MM-DD")
	}
	confirmed, err := fetchLotwReport(user, pass, since, true)
	if err != nil {
		return adifImportResult{}, err
	}
	pending, err := fetchLotwReport(user, pass, since, false)
	if err != nil {
		return adifImportResult{}, err
	}
	// 先写入已确认，随后的未确认副本只会命中去重，不会把确认打回去。
	first, err := importADIF(db, confirmed)
	if err != nil {
		return adifImportResult{}, err
	}
	second, err := importADIF(db, pending)
	if err != nil {
		return first, err
	}
	return first.add(second), nil
}

func fetchLotwReport(user, pass, since string, confirmed bool) ([]byte, error) {
	q := url.Values{}
	q.Set("login", user)
	q.Set("password", pass)
	q.Set("qso_query", "1")
	q.Set("qso_qsldetail", "yes")
	q.Set("qso_withown", "yes")
	if confirmed {
		q.Set("qso_qsl", "yes")
		if since != "" {
			q.Set("qso_qslsince", since)
		}
	} else {
		q.Set("qso_qsl", "no")
		if since != "" {
			q.Set("qso_qsorxsince", since)
		}
	}
	endpoint, err := url.Parse(lotwEndpoint)
	if err != nil {
		return nil, errors.New("LoTW 地址无效")
	}
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, errors.New("无法连接 LoTW")
	}
	req.Header.Set("User-Agent", "qsl-management/lotw-import")
	req.Header.Set("Accept", "text/plain, application/octet-stream, */*")

	client := &http.Client{
		Timeout: lotwTimeout,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= 5 || next.URL.Host != via[0].URL.Host {
				return errors.New("redirect")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, classifyLotwNetErr(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, lotwMaxBody+1))
	if err != nil {
		return nil, errors.New("读取 LoTW 响应失败")
	}
	if len(body) > lotwMaxBody {
		return nil, errors.New("LoTW 返回内容超过 32MB，请把起始日期调近一些后分段同步")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("LoTW 用户名或密码不正确")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LoTW 暂时不可用（HTTP %d）", resp.StatusCode)
	}
	if !looksLikeADIF(body) {
		return nil, lotwFailure(string(body), pass)
	}
	return body, nil
}

func classifyLotwNetErr(err error) error {
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		return errors.New("LoTW 响应超时，请稍后再试")
	}
	return errors.New("无法连接 LoTW")
}

func looksLikeADIF(body []byte) bool {
	// 登录失败时 LoTW 回的是一张 HTML，HTTP 状态仍是 200，靠有没有记录头来区分。
	low := strings.ToLower(string(body))
	return strings.Contains(low, "<eoh>") || strings.Contains(low, "<eor>")
}

func lotwFailure(body, secret string) error {
	plain := strings.ToLower(stripTags(body))
	switch {
	case strings.Contains(plain, "username/password incorrect"),
		strings.Contains(plain, "password incorrect"),
		strings.Contains(plain, "invalid password"),
		strings.Contains(plain, "no such user"),
		strings.Contains(plain, "unknown user"):
		return errors.New("LoTW 用户名或密码不正确")
	}
	snippet := strings.Join(strings.Fields(redactSecret(stripTags(body), secret)), " ")
	if snippet == "" {
		return errors.New("LoTW 没有返回 ADIF（请检查账号密码，或稍后重试）")
	}
	if len([]rune(snippet)) > 160 {
		snippet = string([]rune(snippet)[:160])
	}
	return fmt.Errorf("LoTW 没有返回 ADIF：%s", snippet)
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
		case !in:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func redactSecret(msg, secret string) string {
	if secret == "" {
		return msg
	}
	msg = strings.ReplaceAll(msg, secret, "***")
	msg = strings.ReplaceAll(msg, url.QueryEscape(secret), "***")
	return msg
}
