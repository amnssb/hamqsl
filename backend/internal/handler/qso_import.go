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
// QSO 大约 28 列，100 行约 2800 个参数，低于 SQLite 单条语句 32766 的上限。
const adifInsertBatch = 100

type adifRecord map[string]string

type adifDoc struct {
	ProgramID string
	Records   []adifRecord
}

// adifImportResult 区分四种结局，调用方直接作为 JSON 返回。
type adifImportResult struct {
	Imported int `json:"imported"`
	Updated  int `json:"updated"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

func (r adifImportResult) add(o adifImportResult) adifImportResult {
	r.Imported += o.Imported
	r.Updated += o.Updated
	r.Skipped += o.Skipped
	r.Failed += o.Failed
	return r
}

// parseADIF 解析 ADIF 文本为字段记录列表；<NAME:LEN>value ... <EOR>
func parseADIF(data string) []adifRecord {
	return parseADIFDoc(data).Records
}

// parseADIFDoc 在记录之外记住文件头的 PROGRAMID。
// <EOH> 之前的字段是文件头，不能并进第一条通联，否则 PROGRAMID 会变成一条假 QSO。
func parseADIFDoc(data string) adifDoc {
	doc := adifDoc{Records: []adifRecord{}}
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
		name, length := splitADIFTag(tag)
		switch name {
		case "EOR":
			if len(rec) > 0 {
				doc.Records = append(doc.Records, rec)
				rec = adifRecord{}
			}
			continue
		case "EOH":
			if doc.ProgramID == "" {
				doc.ProgramID = rec["PROGRAMID"]
			}
			rec = adifRecord{}
			continue
		}
		value := readADIFValue(data, &i, length)
		if name != "" && value != "" {
			rec[name] = value
		}
	}
	if len(rec) > 0 {
		doc.Records = append(doc.Records, rec)
	}
	return doc
}

func splitADIFTag(tag string) (name string, length int) {
	name = tag
	if p := strings.IndexByte(tag, ':'); p >= 0 {
		name = tag[:p]
		lenPart := tag[p+1:]
		// ADIF 可以在长度后面再跟一个类型：<QSO_DATE:8:D>、<COMMENT:13:S>。
		// Atoi("8:D") 得到 0，长度被丢掉后只能读到下一个 <。
		// 备注里再出现 <，后面的字段就会被切碎。
		if c := strings.IndexByte(lenPart, ':'); c >= 0 {
			lenPart = lenPart[:c]
		}
		length, _ = strconv.Atoi(strings.TrimSpace(lenPart))
	}
	return strings.ToUpper(strings.TrimSpace(name)), length
}

func readADIFValue(data string, i *int, length int) string {
	if length > 0 && *i+length <= len(data) {
		value := strings.TrimSpace(data[*i : *i+length])
		*i += length
		return value
	}
	rest := data[*i:]
	if nx := strings.IndexByte(rest, '<'); nx >= 0 {
		*i += nx
		return strings.TrimSpace(rest[:nx])
	}
	*i = len(data)
	return strings.TrimSpace(rest)
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

// parseADIFDate 接受 YYYYMMDD 或 YYYY-MM-DD，非法日期返回 false。
func parseADIFDate(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if len(v) >= 10 && v[4] == '-' && v[7] == '-' {
		v = v[:10]
	}
	iso := adifDate(v)
	if len(iso) != 10 {
		return "", false
	}
	if _, err := time.Parse("2006-01-02", iso); err != nil {
		return "", false
	}
	return iso, true
}

func adifTime(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 4 {
		return v[:2] + ":" + v[2:4]
	}
	return v
}

func adifDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseADIFClock(v string) (h, m int, ok bool) {
	v = strings.TrimSpace(v)
	switch {
	case len(v) >= 5 && v[2] == ':' && adifDigits(v[:2]) && adifDigits(v[3:5]):
		h, _ = strconv.Atoi(v[:2])
		m, _ = strconv.Atoi(v[3:5])
	case len(v) >= 4 && adifDigits(v[:4]):
		h, _ = strconv.Atoi(v[:2])
		m, _ = strconv.Atoi(v[2:4])
	default:
		return 0, 0, false
	}
	if h > 23 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

// wallFromADIF 把 QSO_DATE / TIME_ON 收成入库用的日期和 HH:MM。
// asUTC 为真时按 ADIF 的 UTC 口径换算到东八区墙钟；普通 ADIF 保持原文，
// 这样已经导过的日志再次导入仍然对得上原来的去重键。
func wallFromADIF(dateRaw, timeRaw string, asUTC bool) (date, clock string, ok bool) {
	date, ok = parseADIFDate(dateRaw)
	if !ok {
		return "", "", false
	}
	h, m, hasClock := parseADIFClock(timeRaw)
	if !hasClock {
		if asUTC {
			return date, "", true
		}
		return date, adifTime(timeRaw), true
	}
	if !asUTC {
		return date, fmt.Sprintf("%02d:%02d", h, m), true
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %02d:%02d", date, h, m), time.UTC)
	if err != nil {
		return date, fmt.Sprintf("%02d:%02d", h, m), true
	}
	local := t.In(cstZone)
	return local.Format("2006-01-02"), local.Format("15:04"), true
}

func splitLotwTimestamp(ts string) (date, clock string, ok bool) {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return "", "", false
	}
	ts = strings.TrimSuffix(ts, "Z")
	if i := strings.IndexByte(ts, 'T'); i >= 0 {
		ts = ts[:i] + " " + ts[i+1:]
	}
	if len(ts) < 10 {
		return "", "", false
	}
	date = ts[:10]
	if len(ts) >= 16 && ts[10] == ' ' {
		clock = ts[11:16]
	}
	return date, clock, true
}

// canonicalFreq 去掉小数尾部的 0。LoTW 的 FREQ 固定 5 位小数（14.074000），
// 和手填的 14.074 是同一频率。非数字原样返回。
func canonicalFreq(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	neg := false
	t := s
	switch {
	case strings.HasPrefix(t, "+"):
		t = t[1:]
	case strings.HasPrefix(t, "-"):
		neg = true
		t = t[1:]
	}
	if t == "" || strings.Count(t, ".") > 1 {
		return s
	}
	for _, r := range t {
		if (r < '0' || r > '9') && r != '.' {
			return s
		}
	}
	if strings.Contains(t, ".") {
		t = strings.TrimRight(t, "0")
		t = strings.TrimRight(t, ".")
	}
	t = strings.TrimLeft(t, "0")
	if t == "" || strings.HasPrefix(t, ".") {
		t = "0" + t
	}
	if neg && t != "0" {
		return "-" + t
	}
	return t
}

func canonicalMode(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// canonicalBand 只把 20M / 70CM 收成 20m / 70cm。SAT 这种没有数字前缀的保持原样。
func canonicalBand(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	i := 0
	for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || s[i] == '.') {
		i++
	}
	if i == 0 || i == len(s) {
		return s
	}
	return s[:i] + strings.ToLower(s[i:])
}

func adifMode(rec adifRecord) string {
	for _, k := range []string{"MODE", "APP_LOTW_MODE", "SUBMODE", "APP_LOTW_MODEGROUP"} {
		if v := canonicalMode(rec[k]); v != "" {
			return v
		}
	}
	return ""
}

func firstADIF(rec adifRecord, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(rec[k]); v != "" {
			return v
		}
	}
	return ""
}

func programIsLotw(id string) bool {
	id = strings.ToUpper(strings.TrimSpace(id))
	return strings.Contains(id, "LOTW")
}

// adifClockIsUTC 整份文件按 UTC 换算。LoTW 下载的 PROGRAMID 是 LoTW；
// 日志软件导出的文件则靠 LOTW_* / APP_LoTW_* 字段认出来。
// 认出来之后整份一起换算，避免同一份文件里有的记录平移 8 小时、有的不平移。
func adifClockIsUTC(doc adifDoc) bool {
	if programIsLotw(doc.ProgramID) {
		return true
	}
	for _, rec := range doc.Records {
		if programIsLotw(rec["PROGRAMID"]) {
			return true
		}
		for k := range rec {
			if strings.HasPrefix(k, "LOTW_") || strings.HasPrefix(k, "APP_LOTW_") {
				return true
			}
		}
	}
	return false
}

func lotwProgramFile(doc adifDoc) bool {
	if programIsLotw(doc.ProgramID) {
		return true
	}
	for _, rec := range doc.Records {
		if programIsLotw(rec["PROGRAMID"]) {
			return true
		}
	}
	return false
}

// lotwConfirm 判断这条是不是已被 LoTW 确认，并给出确认日期（YYYY-MM-DD，可空）。
// 日志软件用 ADIF 标准字段 LOTW_QSL_RCVD。LoTW 网站自己的 lotwreport.adi 不写这个字段，
// 它用 QSL_RCVD=Y 表示 LoTW 确认；这个含义只在 PROGRAMID 为 LoTW 时成立。
// 普通 ADIF 里的 QSL_RCVD 是纸质/卡片局回卡，不能当成 LoTW。
func lotwConfirm(rec adifRecord, lotwProgram bool) (bool, string) {
	switch strings.ToUpper(strings.TrimSpace(rec["LOTW_QSL_RCVD"])) {
	case "Y", "V", "1":
		return true, lotwQslDate(rec)
	case "N", "I", "R":
		return false, ""
	}
	if strings.TrimSpace(rec["LOTW_QSLRDATE"]) != "" {
		return true, lotwQslDate(rec)
	}
	if lotwProgram && (strings.EqualFold(strings.TrimSpace(rec["QSL_RCVD"]), "Y") ||
		strings.EqualFold(strings.TrimSpace(rec["APP_LOTW_2XQSL"]), "Y")) {
		return true, lotwQslDate(rec)
	}
	return false, ""
}

func lotwQslDate(rec adifRecord) string {
	for _, k := range []string{"LOTW_QSLRDATE", "QSLRDATE"} {
		if d, ok := parseADIFDate(rec[k]); ok {
			return d
		}
	}
	v := strings.TrimSpace(rec["APP_LOTW_RXQSL"])
	if len(v) >= 10 {
		if d, ok := parseADIFDate(v[:10]); ok {
			return d
		}
	}
	return ""
}

// isIgnorableADIF 文件尾的 APP_LoTW_EOF 之类没有呼号和日期，不能算解析失败。
func isIgnorableADIF(rec adifRecord) bool {
	if strings.TrimSpace(rec["CALL"]) != "" || strings.TrimSpace(rec["QSO_DATE"]) != "" || strings.TrimSpace(rec["APP_LOTW_QSO_TIMESTAMP"]) != "" {
		return false
	}
	if len(rec) == 0 {
		return true
	}
	for k := range rec {
		switch k {
		case "APP_LOTW_EOF", "APP_LOTW_NUMREC", "APP_LOTW_LASTQSL", "APP_LOTW_LASTQSORX", "PROGRAMID", "ADIF_VER", "PROGRAMVERSION":
			continue
		default:
			return false
		}
	}
	return true
}

func lotwExtraRemarks(rec adifRecord, qth, mode string) string {
	var parts []string
	add := func(label, val string) {
		val = strings.TrimSpace(val)
		if val == "" {
			return
		}
		parts = append(parts, label+" "+val)
	}
	if g := strings.TrimSpace(rec["GRIDSQUARE"]); g != "" && !strings.EqualFold(g, qth) {
		add("GRID", g)
	}
	if v := strings.TrimSpace(rec["VUCC_GRIDS"]); v != "" && !strings.EqualFold(v, qth) {
		add("VUCC", v)
	}
	dxcc := strings.TrimSpace(rec["DXCC"])
	country := strings.TrimSpace(rec["COUNTRY"])
	if dxcc != "" || country != "" {
		add("DXCC", strings.TrimSpace(dxcc+" "+country))
	}
	add("CNTY", rec["CNTY"])
	add("IOTA", rec["IOTA"])
	add("SAT", rec["SAT_NAME"])
	add("PROP", rec["PROP_MODE"])
	add("CREDIT", firstADIF(rec, "CREDIT_GRANTED", "APP_LOTW_CREDIT_GRANTED"))
	if qslMode := canonicalMode(rec["APP_LOTW_QSLMODE"]); qslMode != "" && qslMode != mode {
		add("对方模式", qslMode)
	}
	if own := strings.ToUpper(firstADIF(rec, "STATION_CALLSIGN", "APP_LOTW_OWNCALL")); own != "" {
		add("本台", own)
	}
	return strings.Join(parts, " · ")
}

// qsoKey 是 ADIF 去重用的四元组：呼号 + 日期 + 频率 + 模式。
// 空频率/空模式与数据库里的 NULL 算同一条。时间、频段参与展示，不参与去重，
// 和原来的导入规则一致，这样手填日志和 LoTW 记录对得上。
type qsoKey struct {
	call string
	date string
	freq string
	mode string
}

func makeQsoKey(call, date, freq, mode string) qsoKey {
	return qsoKey{
		call: strings.ToUpper(strings.TrimSpace(call)),
		date: date,
		freq: canonicalFreq(freq),
		mode: canonicalMode(mode),
	}
}

type qsoHit struct {
	ID          uint
	Confirmed   bool
	ConfirmDate string
}

func keepQsoHit(prev, next qsoHit) qsoHit {
	if prev.ID == 0 {
		return next
	}
	if next.Confirmed != prev.Confirmed {
		if next.Confirmed {
			return next
		}
		return prev
	}
	if next.ConfirmDate != "" && prev.ConfirmDate == "" {
		return next
	}
	if prev.ConfirmDate != "" && next.ConfirmDate == "" {
		return prev
	}
	if next.ID < prev.ID {
		return next
	}
	return prev
}

// ImportADIF 导入 ADIF 通联日志（multipart 字段 file）。
// 按 呼号+日期+频率+模式 去重；缺少呼号或日期的记录算解析失败。
// LoTW 导出的 .adi 走同一入口：认得出 LoTW 时把 UTC 换算成 UTC+8，并写入确认状态。
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

	result, err := importADIF(h.db, raw)
	if err != nil {
		response.Fail(c, 500, "导入失败："+err.Error())
		return
	}
	response.OK(c, result)
}

type adifPending struct {
	item      model.QsoRecord
	confirmed bool
}

// importADIF 把一份 ADIF 写进通联日志。
//
// 以前每条记录都是「查重 + 取一个号 + 插入」三次往返，一份几千条的竞赛日志
// 要打几万次库，而且取号把全局锁占在每一条上。现在：
//  1. 文件内先按四元组去重；同一条里后出现的 LoTW 确认会留下
//  2. 用呼号和日期一次查出库里已有的键
//  3. 已存在且这次显示已确认的，只更新确认状态
//  4. 整批领号，再按 100 行一条 INSERT 写入
func importADIF(db *gorm.DB, raw []byte) (adifImportResult, error) {
	var result adifImportResult
	doc := parseADIFDoc(string(raw))
	useUTC := adifClockIsUTC(doc)
	lotwFile := lotwProgramFile(doc)

	kept := map[qsoKey]*adifPending{}
	var order []qsoKey
	calls := map[string]struct{}{}
	dates := map[string]struct{}{}

	for _, rec := range doc.Records {
		if isIgnorableADIF(rec) {
			continue
		}
		dateRaw, timeRaw := rec["QSO_DATE"], rec["TIME_ON"]
		if strings.TrimSpace(dateRaw) == "" {
			if d, tm, ok := splitLotwTimestamp(rec["APP_LOTW_QSO_TIMESTAMP"]); ok {
				dateRaw, timeRaw = d, tm
			}
		}
		call := strings.ToUpper(strings.TrimSpace(rec["CALL"]))
		date, clock, ok := wallFromADIF(dateRaw, timeRaw, useUTC)
		if call == "" || !ok {
			result.Failed++
			continue
		}
		mode := adifMode(rec)
		freq := canonicalFreq(rec["FREQ"])
		confirmed, qslDate := lotwConfirm(rec, lotwFile)
		qth := firstADIF(rec, "QTH", "GRIDSQUARE", "VUCC_GRIDS")
		remarks := strings.TrimSpace(rec["COMMENT"])
		if useUTC {
			if extra := lotwExtraRemarks(rec, qth, mode); extra != "" {
				if remarks != "" {
					remarks += "\n"
				}
				remarks += extra
			}
		}
		item := model.QsoRecord{
			SceneType:   "QSO",
			CallSign:    call,
			Date:        date,
			Time:        clock,
			Timezone:    "UTC+8",
			Freq:        freq,
			Band:        canonicalBand(rec["BAND"]),
			Mode:        mode,
			RstSent:     rec["RST_SENT"],
			RstRcvd:     rec["RST_RCVD"],
			QTH:         qth,
			MyQTH:       strings.TrimSpace(rec["MY_GRIDSQUARE"]),
			Remarks:     remarks,
			LotwQslRcvd: confirmed,
			LotwQslDate: qslDate,
		}
		// 对方操作员姓名：ADIF 标准是 NAME（对方姓名）；OPERATOR 在 ADIF 语义里
		// 是本台操作员，日志软件常留空——NAME 优先，OPERATOR 回退
		item.Operator = rec["NAME"]
		if item.Operator == "" {
			item.Operator = rec["OPERATOR"]
		}
		key := makeQsoKey(call, date, freq, mode)
		if prev, exists := kept[key]; exists {
			result.Skipped++
			if confirmed && !prev.confirmed {
				prev.item = item
				prev.confirmed = true
			} else if confirmed && prev.confirmed && prev.item.LotwQslDate == "" && qslDate != "" {
				prev.item.LotwQslDate = qslDate
			}
			continue
		}
		kept[key] = &adifPending{item: item, confirmed: confirmed}
		order = append(order, key)
		calls[call] = struct{}{}
		dates[date] = struct{}{}
	}
	if len(order) == 0 {
		return result, nil
	}

	existing, err := existingQsoKeys(db, mapKeys(calls), mapKeys(dates))
	if err != nil {
		return result, err
	}

	var fresh []model.QsoRecord
	var updates []lotwUpdate
	for _, key := range order {
		p := kept[key]
		hit, exists := existing[key]
		if !exists || hit.ID == 0 {
			fresh = append(fresh, p.item)
			existing[key] = qsoHit{}
			continue
		}
		switch {
		case p.confirmed && !hit.Confirmed:
			updates = append(updates, lotwUpdate{ID: hit.ID, Date: p.item.LotwQslDate})
		case p.confirmed && hit.Confirmed && hit.ConfirmDate == "" && p.item.LotwQslDate != "":
			updates = append(updates, lotwUpdate{ID: hit.ID, Date: p.item.LotwQslDate, DateOnly: true})
		default:
			result.Skipped++
		}
	}

	n, err := applyLotwUpdates(db, updates)
	if err != nil {
		return result, err
	}
	result.Updated += n
	result.Skipped += len(updates) - n

	if len(fresh) == 0 {
		if result.Imported > 0 || result.Updated > 0 {
			invalidatePublicStatsCache()
		}
		return result, nil
	}

	// 一次领走这一批的号。插入若失败，空号不回收（和逐条取号时一样）。
	start := reserveSequence(db, "qso_record_sequence", int64(len(fresh)))
	for i := range fresh {
		fresh[i].RecordCode = fmt.Sprintf("QSO-%06d", start+int64(i))
	}

	for i := 0; i < len(fresh); i += adifInsertBatch {
		end := i + adifInsertBatch
		if end > len(fresh) {
			end = len(fresh)
		}
		chunk := append([]model.QsoRecord(nil), fresh[i:end]...)
		if err := db.Create(&chunk).Error; err != nil {
			// 整批失败时退回逐条，让唯一索引冲突只丢掉那一条
			for j := range chunk {
				if err := db.Create(&chunk[j]).Error; err != nil {
					result.Skipped++
					continue
				}
				result.Imported++
			}
			continue
		}
		result.Imported += len(chunk)
	}
	if result.Imported > 0 || result.Updated > 0 {
		invalidatePublicStatsCache()
	}
	return result, nil
}

type lotwUpdate struct {
	ID       uint
	Date     string
	DateOnly bool
}

// applyLotwUpdates 只改确认标记和确认日期。
// WHERE 里带上「尚未确认」，并发或重复导入不会把已确认写回未确认。
func applyLotwUpdates(db *gorm.DB, updates []lotwUpdate) (int, error) {
	if len(updates) == 0 {
		return 0, nil
	}
	changed := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, u := range updates {
			q := tx.Model(&model.QsoRecord{}).Where("id = ?", u.ID)
			var res *gorm.DB
			if u.DateOnly {
				res = q.Where("lotw_qsl_rcvd = ? AND (lotw_qsl_date = '' OR lotw_qsl_date IS NULL)", true).
					Update("lotw_qsl_date", u.Date)
			} else {
				res = q.Where("lotw_qsl_rcvd = ?", false).
					Updates(map[string]any{
						"lotw_qsl_rcvd": true,
						"lotw_qsl_date": u.Date,
					})
			}
			if res.Error != nil {
				return res.Error
			}
			changed += int(res.RowsAffected)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return changed, nil
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
func existingQsoKeys(db *gorm.DB, calls, dates []string) (map[qsoKey]qsoHit, error) {
	out := map[qsoKey]qsoHit{}
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
				ID          uint
				CallSign    string
				Date        string
				Freq        string
				Mode        string
				LotwQslRcvd bool
				LotwQslDate string
			}
			// COALESCE：老数据里频率/模式可能是 NULL，和空字符串算同一条（和以前的去重条件一致）。
			err := db.Model(&model.QsoRecord{}).
				Select("id, call_sign, date, COALESCE(freq, '') AS freq, COALESCE(mode, '') AS mode, lotw_qsl_rcvd, lotw_qsl_date").
				Where("call_sign IN ? AND date IN ?", calls[i:ce], dates[j:de]).
				Find(&rows).Error
			if err != nil {
				return nil, err
			}
			for _, r := range rows {
				key := makeQsoKey(r.CallSign, r.Date, r.Freq, r.Mode)
				hit := qsoHit{ID: r.ID, Confirmed: r.LotwQslRcvd, ConfirmDate: r.LotwQslDate}
				if prev, ok := out[key]; ok {
					out[key] = keepQsoHit(prev, hit)
					continue
				}
				out[key] = hit
			}
		}
	}
	return out, nil
}
