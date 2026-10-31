package model

import "reflect"

/*
表清单：建表、备份导出、备份导入都只读这一份。

加一张新表时只改这个文件，不要在 main.go 或 export.go 里再列一遍——
以前同一批表在三个地方各写一次，漏一处就是"备份里没有它"或"导入后旧数据残留"。

调用示例：

	for _, t := range model.Tables { db.AutoMigrate(t.Model) }   // 建表
	rows := model.NewRows(t)                                     // 拿到 *[]T 交给 GORM
*/
type Table struct {
	Key     string // 备份 JSON 里的键名，也是人读这张表时的叫法
	Model   any    // GORM 模型指针，用于建表与按 id 清空
	Backup  bool   // 是否写进 /api/admin/export
	Restore bool   // 导入时是否先清空、再按备份写入
}

// Tables 的顺序就是导入时的写入顺序（先主数据、后引用它的记录），不要随意调换。
//
// 两个刻意的例外：
//   - users / passkeys：导出可以，导入不覆盖。users 的口令哈希本来就不进备份，
//     恢复了也登不进去；passkeys 的 user_id 在导入后可能悬空。宁可保留现状，
//     由管理员重新绑定，也不要把自己锁在门外。
//   - system_settings：导入时要逐条跳过被剔除的敏感键（SMTP 密码等），
//     由 export.go 单独处理，所以它在通用循环里会被跳过。
var Tables = []Table{
	{Key: "users", Model: &User{}, Backup: true},
	{Key: "station_profiles", Model: &StationProfile{}, Backup: true, Restore: true},
	{Key: "station_cards", Model: &StationCard{}, Backup: true, Restore: true},
	{Key: "station_equipments", Model: &StationEquipment{}, Backup: true, Restore: true},
	{Key: "qso_records", Model: &QsoRecord{}, Backup: true, Restore: true},
	{Key: "card_records", Model: &CardRecord{}, Backup: true, Restore: true},
	{Key: "address_book", Model: &AddressBookEntry{}, Backup: true, Restore: true},
	{Key: "bureau_entries", Model: &BureauEntry{}, Backup: true, Restore: true},
	{Key: "exchange_requests", Model: &ExchangeRequest{}, Backup: true, Restore: true},
	{Key: "offline_activities", Model: &OfflineActivity{}, Backup: true, Restore: true},
	{Key: "receive_records", Model: &ReceiveRecord{}, Backup: true, Restore: true},
	{Key: "audit_logs", Model: &AuditLog{}, Backup: true, Restore: true},

	// 通行密钥：进备份便于留痕与迁移参考，但不参与导入（见上面的说明）
	{Key: "passkeys", Model: &Passkey{}, Backup: true},

	{Key: "system_settings", Model: &SystemSetting{}, Backup: true, Restore: true},

	// 二步验证密钥与半登录票据：绝不进备份。前者是密钥材料，后者是 5 分钟就过期的垃圾。
	{Key: "totp_secrets", Model: &TotpSecret{}},
	{Key: "login_tickets", Model: &LoginTicket{}},
}

// NewRows 返回指向切片零值的指针，供 GORM Find 与 json.Unmarshal 填充。
// 走反射是因为模型类型在清单里已经是变量了；为十几张表各写一个构造函数不值得。
func NewRows(t Table) any {
	elem := reflect.TypeOf(t.Model).Elem()
	return reflect.New(reflect.SliceOf(elem)).Interface()
}
