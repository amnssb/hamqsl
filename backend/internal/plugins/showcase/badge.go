package showcase

import (
	"strconv"
	"strings"
)

// badgeSVG 画一张 480×132 的徽章。颜色写死：它显示在别人的网页上，读不到本站主题变量。
// 文字只用 XML 转义后的呼号和数字，不插入任何标记。
func badgeSVG(s snapshot) string {
	call := s.CallSign
	if call == "" {
		call = "QSL"
	}
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="480" height="132" viewBox="0 0 480 132" role="img">`)
	b.WriteString(`<title>` + xmlEscape(call) + ` QSL</title>`)
	b.WriteString(`<rect width="480" height="132" rx="16" fill="#1a2d3d"/>`)
	b.WriteString(`<rect width="8" height="132" fill="#f5a623"/>`)
	b.WriteString(`<text x="28" y="52" fill="#ffffff" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif" font-size="28" font-weight="700">`)
	b.WriteString(xmlEscape(call))
	b.WriteString(`</text>`)
	b.WriteString(`<text x="452" y="46" fill="#f5a623" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif" font-size="13" font-weight="700" text-anchor="end">QSL</text>`)
	writeStat(&b, 28, "QSO", s.QSO)
	writeStat(&b, 140, "SENT", s.Sent)
	writeStat(&b, 252, "SIGNED", s.Signed)
	writeStat(&b, 364, "RX", s.Received)
	b.WriteString(`</svg>`)
	return b.String()
}

func writeStat(b *strings.Builder, x int, label string, n int64) {
	b.WriteString(`<text x="` + strconv.Itoa(x) + `" y="84" fill="#9c978d" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif" font-size="11">`)
	b.WriteString(label)
	b.WriteString(`</text>`)
	b.WriteString(`<text x="` + strconv.Itoa(x) + `" y="110" fill="#ffffff" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif" font-size="20" font-weight="700">`)
	b.WriteString(strconv.FormatInt(n, 10))
	b.WriteString(`</text>`)
}

func xmlEscape(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&#34;",
		"'", "&#39;",
	).Replace(s)
}
