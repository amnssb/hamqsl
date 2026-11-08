package handler

import (
	"fmt"
	"strconv"
	"sync"

	"gorm.io/gorm"
	"qsl-management/internal/model"
)

// 序列号发号器。
//
// 原来的实现散在三处（qso.go / receive.go / exchange.go），都是
// "读 system_settings → +1 → 写回"，既没有事务也没有互斥：
// 两个并发请求（比如两个人同时提交换卡申请）可能读到同一个值，
// 生成同一个编号。唯一索引会挡住重复写入，但结果是其中一个请求 500。
//
// 这里统一收敛成一个入口，并用进程内互斥把"读-改-写"串起来。
// 部署模型是单进程（docker 单容器），进程内锁足够。
// 如果将来要多副本，需要换成数据库层原子自增
// （PG: UPDATE ... RETURNING；SQLite: 单写者 + busy_timeout 排队）。
var seqMu sync.Mutex

// reserveSequence 一次领走 n 个连续序号，返回其中的第一个（含）。
// n <= 0 时不写库，返回 0。
//
// 和 nextSequence 共用一把锁：导入一整批和同时新建一条卡不会领到同一个号。
// 号码在返回前已经落库。调用方若随后插入失败，这些号就空着——和以前
// 「先取号再插入、插入失败号也不回收」是同一行为，唯一索引仍会挡住重复。
func reserveSequence(db *gorm.DB, key string, n int64) int64 {
	if n <= 0 {
		return 0
	}
	seqMu.Lock()
	var setting model.SystemSetting
	if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
		setting = model.SystemSetting{Key: key, Value: "0"}
	}
	seq, _ := strconv.ParseInt(setting.Value, 10, 64) // 脏值（非数字）从 0 重新数，不 panic
	start := seq + 1
	setting.Value = strconv.FormatInt(seq+n, 10)
	// 不能一律 db.Save：ID=0 的新 struct，Save 走 INSERT，
	// 撞上已有同名 key 会静默失败（历史踩过的坑）。
	if setting.ID == 0 {
		db.Create(&setting)
	} else {
		db.Model(&setting).Update("value", setting.Value)
	}
	seqMu.Unlock()
	invalidateSettingsCache() // 序号存在 system_settings 里，快照要跟着失效
	return start
}

// nextSequence 取下一个序号并格式化。
//
//	prefix 前缀，如 "QSO-" / "R" / "EX"
//	width  数字补零宽度，如 6 → 000123
//	suffix 追加在数字后面的固定串，如 "-20260131"；为空表示不加
func nextSequence(db *gorm.DB, key, prefix string, width int, suffix string) string {
	start := reserveSequence(db, key, 1)
	return fmt.Sprintf("%s%0*d%s", prefix, width, start, suffix)
}
