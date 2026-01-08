package history

import (
	"os"
	"path/filepath"
	"testing"
)

// ========================================
// History Manager Tests
// ========================================

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("Expected non-nil manager")
	}

	if len(mgr.GetRecent()) != 0 {
		t.Error("Expected empty history for new manager")
	}
}

func TestRecord(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	t.Run("record single server", func(t *testing.T) {
		err := mgr.Record("server1")
		if err != nil {
			t.Fatalf("Record failed: %v", err)
		}

		recent := mgr.GetRecent()
		if len(recent) != 1 {
			t.Errorf("Expected 1 entry, got %d", len(recent))
		}

		if recent[0] != "server1" {
			t.Errorf("Expected 'server1', got %s", recent[0])
		}
	})

	t.Run("record multiple servers", func(t *testing.T) {
		mgr.Record("server2")
		mgr.Record("server3")

		recent := mgr.GetRecent()
		if len(recent) != 3 {
			t.Errorf("Expected 3 entries, got %d", len(recent))
		}

		// 最新的应该在前面
		if recent[0] != "server3" {
			t.Errorf("Expected 'server3' first, got %s", recent[0])
		}
	})

	t.Run("record duplicate moves to front", func(t *testing.T) {
		mgr.Record("server1") // 重复记录 server1

		recent := mgr.GetRecent()

		// server1 应该移到最前面
		if recent[0] != "server1" {
			t.Errorf("Expected 'server1' first, got %s", recent[0])
		}

		// 总数应该保持不变（去重）
		if len(recent) != 3 {
			t.Errorf("Expected 3 entries after dedup, got %d", len(recent))
		}
	})
}

func TestIsRecent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	mgr.Record("server1")
	mgr.Record("server2")

	if !mgr.IsRecent("server1") {
		t.Error("Expected server1 to be recent")
	}

	if !mgr.IsRecent("server2") {
		t.Error("Expected server2 to be recent")
	}

	if mgr.IsRecent("server3") {
		t.Error("Expected server3 to not be recent")
	}
}

func TestHistoryPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建管理器并记录历史
	mgr1, _ := NewManager(tmpDir)
	mgr1.Record("server1")
	mgr1.Record("server2")
	mgr1.Record("server3")

	// 创建新的管理器（应该加载之前保存的历史）
	mgr2, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load history: %v", err)
	}

	recent := mgr2.GetRecent()
	if len(recent) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(recent))
	}

	if recent[0] != "server3" {
		t.Errorf("Expected 'server3' first, got %s", recent[0])
	}
}

func TestMaxHistorySize(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	// 记录超过最大数量的服务器
	for i := 1; i <= 15; i++ {
		mgr.Record("server" + string(rune('0'+i)))
	}

	recent := mgr.GetRecent()

	// 应该只保留最近的 maxHistorySize 个
	if len(recent) != maxHistorySize {
		t.Errorf("Expected %d entries, got %d", maxHistorySize, len(recent))
	}
}

func TestLoadCorruptedHistory(t *testing.T) {
	tmpDir := t.TempDir()
	historyFile := filepath.Join(tmpDir, "history.json")

	// 写入损坏的 JSON
	os.WriteFile(historyFile, []byte("{invalid json"), 0644)

	// 加载时应该报错
	mgr, err := NewManager(tmpDir)
	if err == nil {
		t.Error("Expected error when loading corrupted history")
	}

	// 即使出错，manager 也应该是 nil（因为无法正确初始化）
	if mgr != nil {
		t.Error("Expected nil manager when load fails")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, _ := NewManager(tmpDir)

	mgr.Record("server1")
	mgr.Record("server2")
	mgr.Record("server3")

	// 测试内部 remove 函数（通过 Record 触发）
	mgr.remove("server2")

	recent := mgr.GetRecent()
	if len(recent) != 2 {
		t.Errorf("Expected 2 entries after remove, got %d", len(recent))
	}

	for _, name := range recent {
		if name == "server2" {
			t.Error("server2 should have been removed")
		}
	}
}
