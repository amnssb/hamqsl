package handler

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"crypto/tls"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// sendMail 通过后台配置的 SMTP 发送邮件；未配置 host 时返回错误（调用方自行决定是否容错）
func sendMail(db *gorm.DB, to, subject, body string) error {
	host := getSettingValue(db, "smtp_host")
	if host == "" {
		return fmt.Errorf("SMTP 未配置")
	}
	port := getSettingValue(db, "smtp_port")
	if port == "" {
		port = "465"
	}
	user := getSettingValue(db, "smtp_user")
	pass := getSettingValue(db, "smtp_password")
	from := getSettingValue(db, "smtp_from")
	if from == "" {
		from = user
	}
	if from == "" {
		return fmt.Errorf("SMTP 发件人未配置")
	}
	fromName := getSettingValue(db, "smtp_from_name")
	useSSL := getSettingValue(db, "smtp_use_ssl") != "false"

	addr := host + ":" + port
	var conn net.Conn
	var err error
	if useSSL {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("连接 SMTP 失败: %v", err)
	}
	cl, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP 会话失败: %v", err)
	}
	defer cl.Close()
	if !useSSL {
		if ok, _ := cl.Extension("STARTTLS"); ok {
			if err = cl.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return fmt.Errorf("STARTTLS 失败: %v", err)
			}
		}
	}
	if user != "" {
		if ok, _ := cl.Extension("AUTH"); ok {
			if err = cl.Auth(smtp.PlainAuth("", user, pass, host)); err != nil {
				return fmt.Errorf("SMTP 认证失败: %v", err)
			}
		}
	}
	if err = cl.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL 失败: %v", err)
	}
	if err = cl.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT 失败: %v", err)
	}
	w, err := cl.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA 失败: %v", err)
	}
	fromHeader := from
	if fromName != "" {
		fromHeader = fmt.Sprintf("=?UTF-8?B?%s?= <%s>", base64.StdEncoding.EncodeToString([]byte(fromName)), from)
	}
	subj := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		fromHeader, to, subj, mailWrap(subject, body))
	if _, err = w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("写入邮件失败: %v", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("结束邮件失败: %v", err)
	}
	return cl.Quit()
}

// cstZone 东八区（UTC+8）：所有入库与展示时间统一使用
var cstZone = time.FixedZone("UTC+8", 8*3600)

// cstNow 当前东八区时间
func cstNow() time.Time { return time.Now().In(cstZone) }

// htmlEscape 转义插入 HTML 的动态文本
func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&#34;").Replace(s)
}

// mailWrap 统一的 HTML 邮件外壳：标题头 + 正文卡片 + 页脚
func mailWrap(title, bodyHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;padding:24px;background:#f6f4ee;font-family:'Segoe UI','PingFang SC','Microsoft YaHei',sans-serif;">
<div style="max-width:560px;margin:0 auto;background:#ffffff;border:1px solid #e8e4dc;border-radius:8px;overflow:hidden;">
  <div style="background:#1a2d3d;color:#ffffff;padding:18px 24px;font-size:16px;font-weight:bold;letter-spacing:.05em;">%s</div>
  <div style="padding:24px;color:#333333;font-size:14px;line-height:1.9;">%s</div>
  <div style="padding:14px 24px;background:#f8f7f2;color:#9c978d;font-size:12px;border-top:1px solid #e8e4dc;">QSL 卡片管理系统 · 业余无线电</div>
</div>
</body></html>`, htmlEscape(title), bodyHTML)
}

// siteURL 优先使用设置中的站点地址，否则退回请求 Host（用于邮件中的进度链接）
func siteURL(db *gorm.DB, c *gin.Context) string {
	if u := strings.TrimRight(getSettingValue(db, "site_url"), "/"); u != "" {
		return u
	}
	if c == nil || c.Request == nil {
		return ""
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if fp := c.GetHeader("X-Forwarded-Proto"); fp != "" {
		scheme = fp
	}
	return scheme + "://" + c.Request.Host
}
