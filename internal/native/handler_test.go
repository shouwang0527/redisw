package native

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"redisw/internal/config"
)

// ========================================
// Handler Tests
// ========================================

func setupTestHandler(t *testing.T) (*Handler, string, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "redisw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	configPath := filepath.Join(tmpDir, "config.yml")

	// 创建初始配置
	initialServers := []config.RedisServer{
		{Name: "test-server", Host: "127.0.0.1", Port: 6379, Password: ""},
	}
	if err := config.Save(configPath, initialServers); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create initial config: %v", err)
	}

	handler, err := NewHandler(configPath, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create handler: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return handler, configPath, cleanup
}

// Feature: browser-extension, Property 6: 服务器配置往返一致性
func TestProperty_ServerConfigRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("服务器配置往返一致性", prop.ForAll(
		func(name, host string, port int) bool {
			handler, configPath, cleanup := setupTestHandler(t)
			defer cleanup()

			// 确保名称唯一
			uniqueName := name + "_unique"

			// 添加服务器
			addParams, _ := json.Marshal(map[string]interface{}{
				"name":     uniqueName,
				"host":     host,
				"port":     port,
				"password": "",
			})
			addMsg := &Message{ID: "1", Action: "add_server", Params: addParams}
			addResp := handler.Handle(addMsg)

			if !addResp.Success {
				return false
			}

			// 查询列表
			listMsg := &Message{ID: "2", Action: "list_servers"}
			listResp := handler.Handle(listMsg)

			if !listResp.Success {
				return false
			}

			// 验证服务器存在
			servers, err := config.Load(configPath)
			if err != nil {
				return false
			}

			for _, s := range servers {
				if s.Name == uniqueName && s.Host == host && s.Port == port {
					return true
				}
			}

			return false
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.IntRange(1, 65535),
	))

	properties.TestingRun(t)
}

// Feature: browser-extension, Property 7: 服务器更新持久化
func TestProperty_ServerUpdatePersistence(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("服务器更新持久化", prop.ForAll(
		func(newHost string, newPort int) bool {
			handler, configPath, cleanup := setupTestHandler(t)
			defer cleanup()

			// 更新现有服务器
			updateParams, _ := json.Marshal(map[string]interface{}{
				"name": "test-server",
				"server": map[string]interface{}{
					"name":     "test-server",
					"host":     newHost,
					"port":     newPort,
					"password": "",
				},
			})
			updateMsg := &Message{ID: "1", Action: "update_server", Params: updateParams}
			updateResp := handler.Handle(updateMsg)

			if !updateResp.Success {
				return false
			}

			// 验证更新
			servers, err := config.Load(configPath)
			if err != nil {
				return false
			}

			for _, s := range servers {
				if s.Name == "test-server" {
					return s.Host == newHost && s.Port == newPort
				}
			}

			return false
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.IntRange(1, 65535),
	))

	properties.TestingRun(t)
}

// Feature: browser-extension, Property 8: 服务器删除持久化
func TestProperty_ServerDeletePersistence(t *testing.T) {
	handler, configPath, cleanup := setupTestHandler(t)
	defer cleanup()

	// 删除服务器
	deleteParams, _ := json.Marshal(map[string]interface{}{
		"name": "test-server",
	})
	deleteMsg := &Message{ID: "1", Action: "delete_server", Params: deleteParams}
	deleteResp := handler.Handle(deleteMsg)

	if !deleteResp.Success {
		t.Fatalf("Delete failed: %s", deleteResp.Error)
	}

	// 验证删除
	servers, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	for _, s := range servers {
		if s.Name == "test-server" {
			t.Error("Server should have been deleted")
		}
	}
}

// Feature: browser-extension, Property 10: 危险操作确认验证
func TestProperty_DangerousOperationConfirmation(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// 测试 flush_db 无确认
	flushParams, _ := json.Marshal(map[string]interface{}{
		"server":  "test-server",
		"confirm": false,
	})
	flushMsg := &Message{ID: "1", Action: "flush_db", Params: flushParams}
	flushResp := handler.Handle(flushMsg)

	if flushResp.Success {
		t.Error("flush_db should fail without confirmation")
	}

	if flushResp.Error != "confirmation required" {
		t.Errorf("Expected 'confirmation required' error, got: %s", flushResp.Error)
	}

	// 测试 flush_all 无确认
	flushAllMsg := &Message{ID: "2", Action: "flush_all", Params: flushParams}
	flushAllResp := handler.Handle(flushAllMsg)

	if flushAllResp.Success {
		t.Error("flush_all should fail without confirmation")
	}
}

// ========================================
// Unit Tests
// ========================================

func TestHandler_ListServers(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	msg := &Message{ID: "test-1", Action: "list_servers"}
	resp := handler.Handle(msg)

	if !resp.Success {
		t.Fatalf("list_servers failed: %s", resp.Error)
	}

	if resp.ID != "test-1" {
		t.Errorf("Response ID mismatch: got %s, want test-1", resp.ID)
	}
}

func TestHandler_AddServer_Duplicate(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	// 尝试添加重复名称的服务器
	params, _ := json.Marshal(map[string]interface{}{
		"name":     "test-server",
		"host":     "localhost",
		"port":     6380,
		"password": "",
	})
	msg := &Message{ID: "test-1", Action: "add_server", Params: params}
	resp := handler.Handle(msg)

	if resp.Success {
		t.Error("Should fail when adding duplicate server")
	}

	if resp.Error == "" {
		t.Error("Error message should not be empty")
	}
}

func TestHandler_UpdateServer_NotFound(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	params, _ := json.Marshal(map[string]interface{}{
		"name": "non-existent",
		"server": map[string]interface{}{
			"name":     "non-existent",
			"host":     "localhost",
			"port":     6379,
			"password": "",
		},
	})
	msg := &Message{ID: "test-1", Action: "update_server", Params: params}
	resp := handler.Handle(msg)

	if resp.Success {
		t.Error("Should fail when updating non-existent server")
	}
}

func TestHandler_DeleteServer_NotFound(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	params, _ := json.Marshal(map[string]interface{}{
		"name": "non-existent",
	})
	msg := &Message{ID: "test-1", Action: "delete_server", Params: params}
	resp := handler.Handle(msg)

	if resp.Success {
		t.Error("Should fail when deleting non-existent server")
	}
}

func TestHandler_UnknownAction(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	msg := &Message{ID: "test-1", Action: "unknown_action"}
	resp := handler.Handle(msg)

	if resp.Success {
		t.Error("Should fail for unknown action")
	}

	if resp.Error == "" {
		t.Error("Error message should not be empty")
	}
}

func TestHandler_GetHistory(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	msg := &Message{ID: "test-1", Action: "get_history"}
	resp := handler.Handle(msg)

	if !resp.Success {
		t.Fatalf("get_history failed: %s", resp.Error)
	}
}

func TestHandler_RecordHistory(t *testing.T) {
	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	params, _ := json.Marshal(map[string]interface{}{
		"server": "test-server",
	})
	msg := &Message{ID: "test-1", Action: "record_history", Params: params}
	resp := handler.Handle(msg)

	if !resp.Success {
		t.Fatalf("record_history failed: %s", resp.Error)
	}
}
