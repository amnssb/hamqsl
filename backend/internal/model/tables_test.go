package model

import (
	"reflect"
	"testing"
)

// 表清单是建表 / 备份 / 导入的唯一数据源。漏一张表就是"备份里没有它"，
// 或者"导入后旧数据残留"。这些用例把清单的约定钉死。

func TestTablesHaveUniqueKeys(t *testing.T) {
	seen := make(map[string]int, len(Tables))
	for _, tbl := range Tables {
		seen[tbl.Key]++
	}
	for key, n := range seen {
		if n > 1 {
			t.Errorf("表清单里 %q 出现 %d 次（备份 JSON 的键会互相覆盖）", key, n)
		}
	}
}

func TestTablesHaveModels(t *testing.T) {
	for _, tbl := range Tables {
		if tbl.Model == nil {
			t.Errorf("%s 没有 Model，AutoMigrate 会 panic", tbl.Key)
			continue
		}
		// 必须是结构体指针，AutoMigrate 与 NewRows 都依赖这一点
		rt := reflect.TypeOf(tbl.Model)
		if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Struct {
			t.Errorf("%s 的 Model 应为结构体指针，实际 %s", tbl.Key, rt)
		}
	}
}

// Restore 为真的表必须也参与备份，否则"导入时清空、但没有数据写回"= 直接清库
func TestRestorableTablesAreAlsoBackedUp(t *testing.T) {
	for _, tbl := range Tables {
		if tbl.Restore && !tbl.Backup {
			t.Errorf("%s 参与导入恢复却不进备份：导入时会先清空它，再把空数据写回", tbl.Key)
		}
	}
}

// 这两张表刻意只导出不导入（见文件头注释），改动它们要显式改这条用例
func TestUsersAndPasskeysAreNotRestored(t *testing.T) {
	want := map[string]struct {
		backup  bool
		restore bool
	}{
		"users":    {backup: true, restore: false},
		"passkeys": {backup: true, restore: false},
	}
	for _, tbl := range Tables {
		w, ok := want[tbl.Key]
		if !ok {
			continue
		}
		if tbl.Backup != w.backup {
			t.Errorf("%s 的 Backup 应为 %v（口令哈希/公钥不随备份恢复，避免把自己锁在门外）", tbl.Key, w.backup)
		}
		if tbl.Restore != w.restore {
			t.Errorf("%s 的 Restore 应为 %v", tbl.Key, w.restore)
		}
	}
}

// 二步验证密钥与半登录票据绝不能进备份
func TestSecretsAreNeverBackedUp(t *testing.T) {
	for _, tbl := range Tables {
		switch tbl.Key {
		case "totp_secrets", "login_tickets":
			if tbl.Backup {
				t.Errorf("%s 是密钥/临时票据，绝不能进备份", tbl.Key)
			}
			if tbl.Restore {
				t.Errorf("%s 不应参与导入恢复", tbl.Key)
			}
		}
	}
}

func TestTablesCoverRequiredKeys(t *testing.T) {
	// 业务主表一张都不能少
	required := []string{
		"users", "station_profiles", "station_cards", "qso_records",
		"card_records", "address_book", "bureau_entries", "exchange_requests",
		"offline_activities", "receive_records", "audit_logs", "system_settings",
		"passkeys", "totp_secrets", "login_tickets",
	}
	have := make(map[string]bool, len(Tables))
	for _, tbl := range Tables {
		have[tbl.Key] = true
	}
	for _, key := range required {
		if !have[key] {
			t.Errorf("表清单缺少 %q", key)
		}
	}
}

// NewRows 必须返回"指向切片的指针"，否则 Find / json.Unmarshal 填不进去
func TestNewRowsReturnsSlicePointer(t *testing.T) {
	for _, tbl := range Tables {
		rows := NewRows(tbl)
		rt := reflect.TypeOf(rows)
		if rt.Kind() != reflect.Ptr {
			t.Errorf("%s: NewRows 应返回指针，实际 %s", tbl.Key, rt)
			continue
		}
		if rt.Elem().Kind() != reflect.Slice {
			t.Errorf("%s: NewRows 应返回切片指针，实际 %s", tbl.Key, rt)
		}
		// 元素类型要跟 Model 一致
		wantElem := reflect.TypeOf(tbl.Model).Elem()
		if gotElem := rt.Elem().Elem(); gotElem != wantElem {
			t.Errorf("%s: 切片元素类型 %s 与 Model %s 不一致", tbl.Key, gotElem, wantElem)
		}
	}
}

// 导入顺序依赖"先主数据、后引用它的记录"，靠 users 在最前面表达
func TestUsersComesFirst(t *testing.T) {
	if len(Tables) == 0 {
		t.Fatal("表清单为空")
	}
	if Tables[0].Key != "users" {
		t.Errorf("users 应排在首位（导入顺序先主数据），实际首位是 %q", Tables[0].Key)
	}
}
