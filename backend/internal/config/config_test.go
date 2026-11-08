package config

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// 密钥文件的加载/生成是"丢了就全套失守"的逻辑，值得锁住。

func TestLoadOrCreateFileGeneratesOnFirstCall(t *testing.T) {
	dir := t.TempDir()

	first := loadOrCreateFile(dir, ".test_secret")

	if len(first) != 64 {
		t.Fatalf("应生成 64 个 hex 字符（32 字节），实际 %d", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Errorf("生成的内容不是合法 hex: %v", err)
	}

	// 权限必须是 0600：密钥文件对同机其他用户可读就等于泄漏
	info, err := os.Stat(filepath.Join(dir, ".test_secret"))
	if err != nil {
		t.Fatalf("文件未创建: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("密钥文件权限应为 0600，实际 %#o", mode)
	}
}

func TestLoadOrCreateFileIsStable(t *testing.T) {
	dir := t.TempDir()

	first := loadOrCreateFile(dir, ".test_secret")
	second := loadOrCreateFile(dir, ".test_secret")

	if first != second {
		t.Error("第二次调用必须读到同一个密钥，否则每次重启令牌全失效")
	}
}

func TestLoadOrCreateFileRegeneratesWhenCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".test_secret")
	if err := os.WriteFile(path, []byte("short"), 0o600); err != nil {
		t.Fatalf("写测试文件失败: %v", err)
	}

	got := loadOrCreateFile(dir, ".test_secret")

	// 太短的内容会被当成"没有"，重新生成 64 位
	if len(got) != 64 {
		t.Errorf("内容过短应重新生成，实际长度 %d", len(got))
	}
}

func TestLoadOrCreateFileCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "data")

	got := loadOrCreateFile(dir, ".test_secret")

	if len(got) != 64 {
		t.Errorf("应自动建目录并生成密钥，实际长度 %d", len(got))
	}
	if _, err := os.Stat(filepath.Join(dir, ".test_secret")); err != nil {
		t.Errorf("文件未落到新建目录: %v", err)
	}
}

func TestDeriveMFAKeyIs32Bytes(t *testing.T) {
	dir := t.TempDir()

	key := deriveMFAKey(dir)

	if len(key) != 32 {
		t.Fatalf("MFA 主密钥必须 32 字节（AES-256），实际 %d", len(key))
	}
}

func TestDeriveMFAKeyIsStable(t *testing.T) {
	dir := t.TempDir()

	if string(deriveMFAKey(dir)) != string(deriveMFAKey(dir)) {
		t.Error("同一目录两次派生必须得到同一个密钥，否则已绑定的 TOTP 全部解不开")
	}
}

// 文件被改坏时要降级而不是崩：用它自身的摘要兜底，保证服务起得来
func TestDeriveMFAKeySurvivesCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".mfa_key"), []byte("0123456789abcdef0123456789abcdef-correct-length-but-not-hex-32"), 0o600); err != nil {
		t.Fatalf("写测试文件失败: %v", err)
	}

	key := deriveMFAKey(dir)

	if len(key) != 32 {
		t.Errorf("坏文件也要产出 32 字节兜底密钥，实际 %d", len(key))
	}
}

// —— splitList ——

func TestSplitListTrimsAndDropsEmpty(t *testing.T) {
	got := splitList(" https://a.com/ , ,https://b.com ")

	if len(got) != 2 {
		t.Fatalf("期望 2 项，实际 %v", got)
	}
	if got[0] != "https://a.com" || got[1] != "https://b.com" {
		t.Errorf("应去掉空格与尾部斜杠，实际 %v", got)
	}
}

func TestSplitListEmptyInput(t *testing.T) {
	if got := splitList(""); len(got) != 0 {
		t.Errorf("空输入应返回空列表，实际 %v", got)
	}
	if got := splitList("   "); len(got) != 0 {
		t.Errorf("全空白应返回空列表，实际 %v", got)
	}
}

func TestSplitListSingleValue(t *testing.T) {
	got := splitList("qsl.example.com")
	if len(got) != 1 || got[0] != "qsl.example.com" {
		t.Errorf("单值应原样返回，实际 %v", got)
	}
}

// —— getEnv ——

func TestGetEnvFallsBackToDefault(t *testing.T) {
	const key = "QSL_TEST_GETENV_UNSET"
	os.Unsetenv(key)

	if got := getEnv(key, "fallback"); got != "fallback" {
		t.Errorf("未设置时应返回默认值，实际 %q", got)
	}
}

func TestGetEnvTreatsEmptyAsUnset(t *testing.T) {
	const key = "QSL_TEST_GETENV_EMPTY"
	t.Setenv(key, "")

	if got := getEnv(key, "fallback"); got != "fallback" {
		t.Errorf("空串应视为未设置，实际 %q", got)
	}
}

func TestGetEnvUsesProvidedValue(t *testing.T) {
	const key = "QSL_TEST_GETENV_SET"
	t.Setenv(key, "value")

	if got := getEnv(key, "fallback"); got != "value" {
		t.Errorf("应返回显式设置的值，实际 %q", got)
	}
}
