package handler

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

// adifInsertBatch 一次 INSERT 的行数。
// QSO 大约 26 列，100 行约 2600 个参数，低于 SQLite 单条语句 32766 的上限。
const adifInsertBatch = 100

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

// qsoKey 是 ADIF 去重用的四元组。空频率/空模式与数据库里的 NULL 算同一条。
type qsoKey struct {
	call string
	date string
	freq string
	mode string
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

	imported, skipped, err := importADIF(h.db, raw)
	if err != nil {
		response.Fail(c, 500, "导入失败："+err.Error())
		return
	}
	response.OK(c, gin.H{"imported": imported, "skipped": skipped})
}

// importADIF 把一份 ADIF 写进通联日志。
//
// 以前每条记录都是「查重 + 取一个号 + 插入」三次往返，一份几千条的竞赛日志
// 要打几万次库，而且取号把全局锁占在每一条上。现在：
//  1. 文件内先按四元组去重
//  2. 用呼号和日期一次查出库里已有的键
//  3. 整批领号，再按 100 行一条 INSERT 写入
func importADIF(db *gorm.DB, raw []byte) (imported, skipped int, err error) {
	type pending struct {
		key  qsoKey
		item model.QsoRecord
	}
	var fresh []pending
	seen := map[qsoKey]struct{}{}
	calls := map[string]struct{}{}
	dates := map[string]struct{}{}

	for _, rec := range parseADIF(string(raw)) {
		call := strings.ToUpper(strings.TrimSpace(rec["CALL"]))
		date := adifDate(rec["QSO_DATE"])
		if call == "" || date == "" {
			skipped++
			continue
		}
		key := qsoKey{call: call, date: date, freq: rec["FREQ"], mode: rec["MODE"]}
		if _, ok := seen[key]; ok {
			skipped++ // 同一份文件里的重复行
			continue
		}
		seen[key] = struct{}{}
		calls[call] = struct{}{}
		dates[date] = struct{}{}

		item := model.QsoRecord{
			SceneType: "QSO",
			CallSign:  call,
			Date:      date,
			Time:      adifTime(rec["TIME_ON"]),
			Freq:      rec["FREQ"],
			Band:      rec["BAND"],
			Mode:      rec["MODE"],
			RstSent:   rec["RST_SENT"],
			RstRcvd:   rec["RST_RCVD"],
			QTH:       rec["QTH"],
			Remarks:   rec["COMMENT"],
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
		fresh = append(fresh, pending{key: key, item: item})
	}
	if len(fresh) == 0 {
		return 0, skipped, nil
	}

	existing, err := existingQsoKeys(db, mapKeys(calls), mapKeys(dates))
	if err != nil {
		return 0, skipped, err
	}
	rows := make([]model.QsoRecord, 0, len(fresh))
	for _, p := range fresh {
		if _, ok := existing[p.key]; ok {
			skipped++
			continue
		}
		existing[p.key] = struct{}{}
		rows = append(rows, p.item)
	}
	if len(rows) == 0 {
		return 0, skipped, nil
	}

	// 一次领走这一批的号。插入若失败，空号不回收（和逐条取号时一样）。
	start := reserveSequence(db, "qso_record_sequence", int64(len(rows)))
	for i := range rows {
		rows[i].RecordCode = fmt.Sprintf("QSO-%06d", start+int64(i))
	}

	for i := 0; i < len(rows); i += adifInsertBatch {
		end := i + adifInsertBatch
		if end > len(rows) {
			end = len(rows)
		}
		chunk := append([]model.QsoRecord(nil), rows[i:end]...)
		if err := db.Create(&chunk).Error; err != nil {
			// 整批失败时退回逐条，让唯一索引冲突只丢掉那一条
			for j := range chunk {
				if err := db.Create(&chunk[j]).Error; err != nil {
					skipped++
					continue
				}
				imported++
			}
			continue
		}
		imported += len(chunk)
	}
	return imported, skipped, nil
}

func mapKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// existingQsoKeys 查出这批呼号+日期里已经在库中的去重键。
// 日期不在文件里的旧日志不可能撞上，不必读出来。
func existingQsoKeys(db *gorm.DB, calls, dates []string) (map[qsoKey]struct{}, error) {
	out := map[qsoKey]struct{}{}
	if len(calls) == 0 || len(dates) == 0 {
		return out, nil
	}
	const chunk = 200 // IN 列表分块，避开老版本 SQLite 999 个参数的上限
	for i := 0; i < len(calls); i += chunk {
		ce := i + chunk
		if ce > len(calls) {
			ce = len(calls)
		}
		for j := 0; j < len(dates); j += chunk {
			de := j + chunk
			if de > len(dates) {
				de = len(dates)
			}
			var rows []struct {
				CallSign string
				Date     string
				Freq     string
				Mode     string
			}
			// COALESCE：老数据里频率/模式可能是 NULL，和空字符串算同一条（和以前的去重条件一致）。
			err := db.Model(&model.QsoRecord{}).
				Select("call_sign, date, COALESCE(freq, '') AS freq, COALESCE(mode, '') AS mode").
				Where("call_sign IN ? AND date IN ?", calls[i:ce], dates[j:de]).
				Find(&rows).Error
			if err != nil {
				return nil, err
			}
			for _, r := range rows {
				out[qsoKey{call: r.CallSign, date: r.Date, freq: r.Freq, mode: r.Mode}] = struct{}{}
			}
		}
	}
	return out, nil
}
