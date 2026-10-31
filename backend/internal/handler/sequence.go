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

// nextSequence 取下一个序号并格式化。
//
//	prefix 前缀，如 "QSO-" / "R" / "EX"
//	width  数字补零宽度，如 6 → 000123
//	suffix 追加在数字后面的固定串，如 "-20260131"；为空表示不加
func nextSequence(db *gorm.DB, key, prefix string, width int, suffix string) string {
	seqMu.Lock()
	defer seqMu.Unlock()

	var setting model.SystemSetting
	loadErr := db.Where("key = ?", key).First(&setting).Error
	if loadErr != nil {
		setting = model.SystemSetting{Key: key, Value: "0"}
	}
	seq, _ := strconv.ParseInt(setting.Value, 10, 64)
	seq++
	setting.Value = strconv.FormatInt(seq, 10)

	// 注意：不能一律 db.Save —— 对 ID=0 的新 struct，Save 走 INSERT，
	// 遇到已有同名 key 会撞唯一索引并静默失败（历史踩过的坑）。
	if setting.ID == 0 {
		db.Create(&setting)
	} else {
		db.Model(&setting).Update("value", setting.Value)
	}

	return fmt.Sprintf("%s%0*d%s", prefix, width, seq, suffix)
}
