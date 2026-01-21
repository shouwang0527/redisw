package native

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"redisw/internal/history"
)

// ========================================
// History Property Tests
// ========================================

// Feature: browser-extension, Property 11: 历史记录往返一致性
func TestProperty_HistoryRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("历史记录往返一致性", prop.ForAll(
		func(serverName string) bool {
			// 创建临时目录
			tmpDir, err := os.MkdirTemp("", "history-test-*")
			if err != nil {
				return false
			}
			defer os.RemoveAll(tmpDir)

			// 创建历史管理器
			mgr, err := history.NewManager(tmpDir)
			if err != nil {
				return false
			}

			// 记录历史
			if err := mgr.Record(serverName); err != nil {
				return false
			}

			// 验证历史包含该服务器
			recent := mgr.GetRecent()
			for _, name := range recent {
				if name == serverName {
					return true
				}
			}

			return false
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t)
}

// Feature: browser-extension, Property 12: 历史记录容量限制
func TestProperty_HistoryCapacityLimit(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("历史记录容量限制", prop.ForAll(
		func(count int) bool {
			// 创建临时目录
			tmpDir, err := os.MkdirTemp("", "history-test-*")
			if err != nil {
				return false
			}
			defer os.RemoveAll(tmpDir)

			// 创建历史管理器
			mgr, err := history.NewManager(tmpDir)
			if err != nil {
				return false
			}

			// 记录多个服务器
			for i := 0; i < count; i++ {
				mgr.Record("server-" + string(rune('a'+i%26)))
			}

			// 验证历史不超过 10 条
			recent := mgr.GetRecent()
			return len(recent) <= 10
		},
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t)
}

// Feature: browser-extension, Property 13: 历史记录去重和排序
func TestProperty_HistoryDeduplication(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "history-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建历史管理器
	mgr, err := history.NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}

	// 记录同一服务器多次
	mgr.Record("server-a")
	mgr.Record("server-b")
	mgr.Record("server-a") // 重复

	recent := mgr.GetRecent()

	// 验证只有一个 server-a
	count := 0
	for _, name := range recent {
		if name == "server-a" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected 1 occurrence of server-a, got %d", count)
	}

	// 验证 server-a 在列表头部
	if len(recent) > 0 && recent[0] != "server-a" {
		t.Errorf("Expected server-a at head, got %s", recent[0])
	}
}

// ========================================
// Handler History Tests
// ========================================

func TestHandler_HistoryRoundTrip(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// 记录历史
	recordParams, _ := json.Marshal(map[string]interface{}{
		"server": "test-server",
	})
	recordMsg := &Message{ID: "1", Action: "record_history", Params: recordParams}
	recordResp := handler.Handle(recordMsg)

	if !recordResp.Success {
		t.Fatalf("record_history failed: %s", recordResp.Error)
	}

	// 获取历史
	getMsg := &Message{ID: "2", Action: "get_history"}
	getResp := handler.Handle(getMsg)

	if !getResp.Success {
		t.Fatalf("get_history failed: %s", getResp.Error)
	}

	// 验证历史包含 test-server
	historyData, ok := getResp.Data.([]string)
	if !ok {
		// 可能是 []interface{}
		if historySlice, ok := getResp.Data.([]interface{}); ok {
			for _, item := range historySlice {
				if str, ok := item.(string); ok && str == "test-server" {
					return // 测试通过
				}
			}
		}
		t.Error("test-server not found in history")
		return
	}

	found := false
	for _, name := range historyData {
		if name == "test-server" {
			found = true
			break
		}
	}

	if !found {
		t.Error("test-server not found in history")
	}
}

// ========================================
// Import Tests
// ========================================

func TestHandler_ImportServers_JSON(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	importData := `[{"name":"imported-server","host":"192.168.1.1","port":6379,"password":""}]`

	params, _ := json.Marshal(map[string]interface{}{
		"data":     importData,
		"format":   "json",
		"conflict": "skip",
	})
	msg := &Message{ID: "1", Action: "import_servers", Params: params}
	resp := handler.Handle(msg)

	if !resp.Success {
		t.Fatalf("import_servers failed: %s", resp.Error)
	}

	// 验证导入结果
	result, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Invalid response data type")
	}

	if result["imported"].(float64) != 1 {
		t.Errorf("Expected 1 imported, got %v", result["imported"])
	}
}

func TestHandler_ImportServers_Conflict_Skip(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// 导入与现有服务器同名的配置
	importData := `[{"name":"test-server","host":"192.168.1.1","port":6380,"password":""}]`

	params, _ := json.Marshal(map[string]interface{}{
		"data":     importData,
		"format":   "json",
		"conflict": "skip",
	})
	msg := &Message{ID: "1", Action: "import_servers", Params: params}
	resp := handler.Handle(msg)

	if !resp.Success {
		t.Fatalf("import_servers failed: %s", resp.Error)
	}

	result := resp.Data.(map[string]interface{})
	if result["skipped"].(float64) != 1 {
		t.Errorf("Expected 1 skipped, got %v", result["skipped"])
	}
}

func TestHandler_ImportServers_Conflict_Overwrite(t *testing.T) {
	handler, configPath, cleanup := setupTestHandler(t)
	defer cleanup()

	// 导入与现有服务器同名的配置（覆盖）
	importData := `[{"name":"test-server","host":"192.168.1.1","port":6380,"password":"newpass"}]`

	params, _ := json.Marshal(map[string]interface{}{
		"data":     importData,
		"format":   "json",
		"conflict": "overwrite",
	})
	msg := &Message{ID: "1", Action: "import_servers", Params: params}
	resp := handler.Handle(msg)

	if !resp.Success {
		t.Fatalf("import_servers failed: %s", resp.Error)
	}

	// 验证配置已更新
	servers, _ := loadConfigForTest(configPath)
	for _, s := range servers {
		if s.Name == "test-server" {
			if s.Host != "192.168.1.1" || s.Port != 6380 {
				t.Error("Server was not overwritten correctly")
			}
			return
		}
	}
	t.Error("test-server not found after overwrite")
}

func TestHandler_ImportServers_Conflict_Rename(t *testing.T) {
	handler, configPath, cleanup := setupTestHandler(t)
	defer cleanup()

	// 导入与现有服务器同名的配置（重命名）
	importData := `[{"name":"test-server","host":"192.168.1.1","port":6380,"password":""}]`

	params, _ := json.Marshal(map[string]interface{}{
		"data":     importData,
		"format":   "json",
		"conflict": "rename",
	})
	msg := &Message{ID: "1", Action: "import_servers", Params: params}
	resp := handler.Handle(msg)

	if !resp.Success {
		t.Fatalf("import_servers failed: %s", resp.Error)
	}

	// 验证新服务器被重命名
	servers, _ := loadConfigForTest(configPath)
	foundOriginal := false
	foundRenamed := false
	for _, s := range servers {
		if s.Name == "test-server" {
			foundOriginal = true
		}
		if s.Name == "test-server_1" {
			foundRenamed = true
		}
	}

	if !foundOriginal {
		t.Error("Original server should still exist")
	}
	if !foundRenamed {
		t.Error("Renamed server should exist")
	}
}

func loadConfigForTest(path string) ([]struct {
	Name     string
	Host     string
	Port     int
	Password string
}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var servers []struct {
		Name     string
		Host     string
		Port     int
		Password string
	}

	// 简单解析 YAML
	// 实际使用 config.Load
	_ = data
	return servers, nil
}
