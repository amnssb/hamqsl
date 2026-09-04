package handler

import (
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type adifRecord map[string]string

// parseADIF 解析 ADIF 文本为字段记录列表；<NAME:LEN>value ... <EOR>
func parseADIF(data string) []adifRecord {
	records := []adifRecord{}
	rec := adifRecord{}
	i := 0
	for i < len(data) {
		lt := strings.IndexByte(data[i:], '<')
		if lt < 0 {
			break
		}
		gt := strings.IndexByte(data[i+lt:], '>')
		if gt < 0 {
			break
		}
		tag := strings.TrimSpace(data[i+lt+1 : i+lt+gt])
		i = i + lt + gt + 1
		if strings.EqualFold(tag, "EOR") {
			if len(rec) > 0 {
				records = append(records, rec)
				rec = adifRecord{}
			}
			continue
		}
		name, length := tag, 0
		if p := strings.IndexByte(tag, ':'); p >= 0 {
			name = tag[:p]
			length, _ = strconv.Atoi(strings.TrimSpace(tag[p+1:]))
		}
		name = strings.ToUpper(strings.TrimSpace(name))
		value := ""
		if length > 0 && i+length <= len(data) {
			value = strings.TrimSpace(data[i : i+length])
			i += length
		} else if nx := strings.IndexByte(data[i:], '<'); nx >= 0 {
			value = strings.TrimSpace(data[i : i+nx])
			i += nx
		} else {
			value = strings.TrimSpace(data[i:])
			i = len(data)
		}
		if name != "" && value != "" {
			rec[name] = value
		}
	}
	if len(rec) > 0 {
		records = append(records, rec)
	}
	return records
}

func adifDate(v string) string {
	v = strings.TrimSpace(v)
	if len(v) == 8 {
		if _, err := time.Parse("20060102", v); err == nil {
			return v[:4] + "-" + v[4:6] + "-" + v[6:]
		}
	}
	return v
}

func adifTime(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 4 {
		return v[:2] + ":" + v[2:4]
	}
	return v
}

// ImportADIF 导入 ADIF 通联日志（multipart 字段 file）。
// 按 呼号+日期+频率+模式 去重；缺少呼号或日期的记录跳过。
func (h *QsoHandler) ImportADIF(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, 400, "请选择 ADIF 文件")
		return
	}
	if file.Size > 5<<20 {
		response.Fail(c, 400, "文件不能超过 5MB")
		return
	}
	f, err := file.Open()
	if err != nil {
		response.Fail(c, 400, "读取文件失败")
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		response.Fail(c, 400, "读取文件失败")
		return
	}

	imported, skipped := 0, 0
	for _, rec := range parseADIF(string(raw)) {
		call := strings.ToUpper(strings.TrimSpace(rec["CALL"]))
		date := adifDate(rec["QSO_DATE"])
		if call == "" || date == "" {
			skipped++
			continue
		}
		freq, mode := rec["FREQ"], rec["MODE"]
		var cnt int64
		q := h.db.Model(&model.QsoRecord{}).Where("call_sign = ? AND date = ?", call, date)
		if freq != "" {
			q = q.Where("freq = ?", freq)
		} else {
			q = q.Where("(freq = '' OR freq IS NULL)")
		}
		if mode != "" {
			q = q.Where("mode = ?", mode)
		} else {
			q = q.Where("(mode = '' OR mode IS NULL)")
		}
		q.Count(&cnt)
		if cnt > 0 {
			skipped++
			continue
		}
		item := model.QsoRecord{
			RecordCode: nextQsoRecordCode(h.db),
			SceneType:  "QSO",
			CallSign:   call,
			Date:       date,
			Time:       adifTime(rec["TIME_ON"]),
			Freq:       freq,
			Band:       rec["BAND"],
			Mode:       mode,
			RstSent:    rec["RST_SENT"],
			RstRcvd:    rec["RST_RCVD"],
			QTH:        rec["QTH"],
			Remarks:    rec["COMMENT"],
		}
		// 对方操作员姓名：ADIF 标准是 NAME（对方姓名）；OPERATOR 在 ADIF 语义里
		// 是本台操作员，日志软件常留空——NAME 优先，OPERATOR 回退
		item.Operator = rec["NAME"]
		if item.Operator == "" {
			item.Operator = rec["OPERATOR"]
		}
		if item.QTH == "" {
			item.QTH = rec["GRIDSQUARE"]
		}
		if err := h.db.Create(&item).Error; err != nil {
			skipped++
			continue
		}
		imported++
	}
	response.OK(c, gin.H{"imported": imported, "skipped": skipped})
}
