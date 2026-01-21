package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCreatesMissingDirectory(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 临时修改 HOME，使 EnsureConfigDir() 使用我们的测试目录
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// 配置文件路径（目录不存在）
	configPath := filepath.Join(tmpDir, ".config", "redisw", "test_config.yml")

	// 确保目录不存在
	if _, err := os.Stat(filepath.Dir(configPath)); err == nil {
		t.Fatal("Test directory should not exist initially")
	}

	// 创建测试数据
	servers := []RedisServer{
		{Name: "test-server", Host: "localhost", Port: 6379, Password: "secret"},
	}

	// 尝试保存（目录不存在，但 Save() 应该自动创建）
	if err := Save(configPath, servers); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Config file was not created: %v", err)
	}

	// 验证内容正确
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if len(loaded) != 1 || loaded[0].Name != "test-server" {
		t.Fatalf("Loaded config doesn't match: %+v", loaded)
	}

	t.Log("✓ Save() successfully creates missing directories and saves config")
}
