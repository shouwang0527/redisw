package history

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ========================================
// Connection History Management
// 记录最近连接的服务器，用于优先排序
// ========================================

const (
	maxHistorySize = 10 // 保留最近 10 次连接记录
)

// History 连接历史记录
type History struct {
	Recent []string `json:"recent"` // 服务器名称列表（最近使用的在前）
}

// Manager 历史记录管理器
type Manager struct {
	filePath string
	history  *History
}

// NewManager 创建历史记录管理器
func NewManager(configDir string) (*Manager, error) {
	filePath := filepath.Join(configDir, "history.json")

	manager := &Manager{
		filePath: filePath,
		history:  &History{Recent: []string{}},
	}

	// 尝试加载现有历史
	if err := manager.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return manager, nil
}

// Record 记录一次连接（更新历史）
func (m *Manager) Record(serverName string) error {
	// 移除重复项
	m.remove(serverName)

	// 添加到列表头部
	m.history.Recent = append([]string{serverName}, m.history.Recent...)

	// 限制历史大小
	if len(m.history.Recent) > maxHistorySize {
		m.history.Recent = m.history.Recent[:maxHistorySize]
	}

	// 持久化
	return m.save()
}

// GetRecent 获取最近使用的服务器名称列表
func (m *Manager) GetRecent() []string {
	return m.history.Recent
}

// IsRecent 检查服务器是否在最近使用列表中
func (m *Manager) IsRecent(serverName string) bool {
	for _, name := range m.history.Recent {
		if name == serverName {
			return true
		}
	}
	return false
}

// load 从文件加载历史记录
func (m *Manager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m.history)
}

// save 保存历史记录到文件
func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0644)
}

// remove 从历史中移除指定服务器（内部辅助函数）
func (m *Manager) remove(serverName string) {
	filtered := []string{}
	for _, name := range m.history.Recent {
		if name != serverName {
			filtered = append(filtered, name)
		}
	}
	m.history.Recent = filtered
}
