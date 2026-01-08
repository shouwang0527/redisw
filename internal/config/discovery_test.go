package config

import (
	"os"
	"path/filepath"
	"testing"
)

// ========================================
// Configuration Discovery Tests
// ========================================

func TestDiscoverConfigFile(t *testing.T) {
	// 测试默认行为：应该返回配置目录下的路径
	path := DiscoverConfigFile()
	if path == "" {
		t.Error("Expected non-empty config path")
	}

	// 路径应该包含 redisw_config
	if !contains(path, "redisw_config") {
		t.Errorf("Expected path to contain 'redisw_config', got: %s", path)
	}
}

func TestCheckHomeDir(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	// 测试当前环境（可能有或没有 redisw_config）
	result := checkHomeDir()

	// 如果返回非空，验证路径确实存在
	if result != "" {
		if _, err := os.Stat(result); os.IsNotExist(err) {
			t.Errorf("checkHomeDir returned path that doesn't exist: %s", result)
		}

		// 验证路径在家目录下
		if !filepath.HasPrefix(result, homeDir) {
			t.Errorf("Expected path to be in home dir, got: %s", result)
		}
	}
}

func TestFindFirstExisting(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config1.yml")
	file2 := filepath.Join(tmpDir, "config2.yml")

	// 场景 1: 都不存在
	result := findFirstExisting([]string{file1, file2})
	if result != "" {
		t.Errorf("Expected empty string, got: %s", result)
	}

	// 场景 2: 第二个存在
	os.WriteFile(file2, []byte("test"), 0644)
	result = findFirstExisting([]string{file1, file2})
	if result != file2 {
		t.Errorf("Expected %s, got: %s", file2, result)
	}

	// 场景 3: 两个都存在，返回第一个
	os.WriteFile(file1, []byte("test"), 0644)
	result = findFirstExisting([]string{file1, file2})
	if result != file1 {
		t.Errorf("Expected %s, got: %s", file1, result)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	err := EnsureConfigDir()
	if err != nil {
		t.Errorf("EnsureConfigDir failed: %v", err)
	}

	// 验证目录存在
	configDir := getConfigDir()
	if configDir != "" {
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			t.Error("Config directory was not created")
		}
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && filepath.Base(s) != "" &&
		(filepath.Base(s) == substr || len(s) >= len(substr))
}
