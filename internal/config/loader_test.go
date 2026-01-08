package config

import (
	"os"
	"path/filepath"
	"testing"
)

// ========================================
// Configuration Loading Tests
// ========================================

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("load existing config", func(t *testing.T) {
		// 创建测试配置文件
		configFile := filepath.Join(tmpDir, "test_config.yml")
		content := `- name: "test1"
  host: "localhost"
  port: 6379
  password: ""
- name: "test2"
  host: "example.com"
  port: 6380
  password: "testpass"
`
		if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		// 加载配置
		servers, err := Load(configFile)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		// 验证结果
		if len(servers) != 2 {
			t.Errorf("Expected 2 servers, got %d", len(servers))
		}

		if servers[0].Name != "test1" {
			t.Errorf("Expected name 'test1', got %s", servers[0].Name)
		}

		if servers[1].Password != "testpass" {
			t.Errorf("Expected password 'testpass', got %s", servers[1].Password)
		}
	})

	t.Run("create default config when not exists", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, "new_config.yml")

		// 加载不存在的配置（应该创建默认配置）
		servers, err := Load(configFile)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		// 验证创建了默认配置
		if len(servers) != 1 {
			t.Errorf("Expected 1 default server, got %d", len(servers))
		}

		if servers[0].Name != "localhost" {
			t.Errorf("Expected default name 'localhost', got %s", servers[0].Name)
		}

		// 验证文件已创建
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("Config file was not created")
		}
	})
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "save_test.yml")

	// 创建测试数据
	servers := []RedisServer{
		{
			Name:     "test-server",
			Host:     "127.0.0.1",
			Port:     6379,
			Password: "secret",
		},
	}

	// 保存配置
	if err := Save(configFile, servers); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 重新加载验证
	loaded, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load after save failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Errorf("Expected 1 server, got %d", len(loaded))
	}

	if loaded[0].Name != "test-server" {
		t.Errorf("Expected name 'test-server', got %s", loaded[0].Name)
	}

	if loaded[0].Password != "secret" {
		t.Errorf("Expected password 'secret', got %s", loaded[0].Password)
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "default.yml")

	// 创建默认配置
	if err := createDefaultConfig(configFile); err != nil {
		t.Fatalf("createDefaultConfig failed: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// 加载并验证内容
	servers, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load default config failed: %v", err)
	}

	if len(servers) != 1 {
		t.Errorf("Expected 1 default server, got %d", len(servers))
	}

	if servers[0].Host != "127.0.0.1" {
		t.Errorf("Expected default host '127.0.0.1', got %s", servers[0].Host)
	}

	if servers[0].Port != 6379 {
		t.Errorf("Expected default port 6379, got %d", servers[0].Port)
	}
}
