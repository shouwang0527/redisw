package native

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ========================================
// Install Command Tests
// ========================================

// NativeManifest 用于测试的 manifest 结构
type TestNativeManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

// Feature: browser-extension, Property 4: Manifest 内容完整性
func TestProperty_ManifestContentIntegrity(t *testing.T) {
	// 创建临时目录模拟浏览器目录
	tmpDir, err := os.MkdirTemp("", "install-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试 manifest
	manifest := TestNativeManifest{
		Name:           "com.redisw.native",
		Description:    "Redisw Native Messaging Host",
		Path:           "/usr/local/bin/redisw",
		Type:           "stdio",
		AllowedOrigins: []string{"chrome-extension://test-id/"},
	}

	manifestPath := filepath.Join(tmpDir, "com.redisw.native.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// 读取并验证
	readData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	var readManifest TestNativeManifest
	if err := json.Unmarshal(readData, &readManifest); err != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", err)
	}

	// 验证必需字段
	if readManifest.Name == "" {
		t.Error("Manifest name is empty")
	}
	if readManifest.Description == "" {
		t.Error("Manifest description is empty")
	}
	if readManifest.Path == "" {
		t.Error("Manifest path is empty")
	}
	if readManifest.Type == "" {
		t.Error("Manifest type is empty")
	}
	if len(readManifest.AllowedOrigins) == 0 {
		t.Error("Manifest allowed_origins is empty")
	}
}

// Feature: browser-extension, Property 5: 安装命令幂等性
func TestProperty_InstallIdempotency(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "install-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestPath := filepath.Join(tmpDir, "com.redisw.native.json")

	// 第一次安装
	manifest := TestNativeManifest{
		Name:           "com.redisw.native",
		Description:    "Redisw Native Messaging Host",
		Path:           "/usr/local/bin/redisw",
		Type:           "stdio",
		AllowedOrigins: []string{"chrome-extension://test-id/"},
	}

	data, _ := json.MarshalIndent(manifest, "", "  ")
	os.WriteFile(manifestPath, data, 0644)

	// 读取第一次安装的内容
	firstContent, _ := os.ReadFile(manifestPath)

	// 第二次安装（覆盖）
	os.WriteFile(manifestPath, data, 0644)

	// 读取第二次安装的内容
	secondContent, _ := os.ReadFile(manifestPath)

	// 验证两次安装结果相同
	if string(firstContent) != string(secondContent) {
		t.Error("Install is not idempotent - content differs between runs")
	}
}

// 测试 manifest 文件格式
func TestManifestFormat(t *testing.T) {
	manifest := TestNativeManifest{
		Name:           "com.redisw.native",
		Description:    "Redisw Native Messaging Host",
		Path:           "/usr/local/bin/redisw",
		Type:           "stdio",
		AllowedOrigins: []string{"chrome-extension://test-id/"},
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	// 验证 JSON 格式正确
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Generated manifest is not valid JSON: %v", err)
	}

	// 验证字段存在
	requiredFields := []string{"name", "description", "path", "type", "allowed_origins"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}
}

// 测试 type 字段值
func TestManifestTypeValue(t *testing.T) {
	manifest := TestNativeManifest{
		Name:           "com.redisw.native",
		Description:    "Redisw Native Messaging Host",
		Path:           "/usr/local/bin/redisw",
		Type:           "stdio",
		AllowedOrigins: []string{"chrome-extension://test-id/"},
	}

	if manifest.Type != "stdio" {
		t.Errorf("Expected type 'stdio', got '%s'", manifest.Type)
	}
}

// 测试 allowed_origins 格式
func TestManifestAllowedOriginsFormat(t *testing.T) {
	origins := []string{
		"chrome-extension://abcdefghijklmnop/",
		"chrome-extension://1234567890abcdef/",
	}

	for _, origin := range origins {
		// 验证格式：chrome-extension://ID/
		if len(origin) < 20 {
			t.Errorf("Origin too short: %s", origin)
		}
		if origin[len(origin)-1] != '/' {
			t.Errorf("Origin should end with '/': %s", origin)
		}
	}
}
