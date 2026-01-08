package ui

import (
	"fmt"
	"sort"

	"redisw/internal/config"
	"redisw/internal/connector"
	"redisw/internal/history"
)

// ========================================
// Enhanced UI with History & Status
// ========================================

// EnhancedSelector 增强型选择器（支持历史记录排序和状态显示）
type EnhancedSelector struct {
	servers   []config.RedisServer
	history   *history.Manager
	connector *connector.Connector
}

// NewEnhancedSelector 创建增强型选择器
func NewEnhancedSelector(
	servers []config.RedisServer,
	historyMgr *history.Manager,
	conn *connector.Connector,
) *EnhancedSelector {
	return &EnhancedSelector{
		servers:   servers,
		history:   historyMgr,
		connector: conn,
	}
}

// Select 显示增强的选择界面
// - 按历史记录排序（最近使用的在前）
// - 显示连接状态（可达/不可达）
func (e *EnhancedSelector) Select() *config.RedisServer {
	// 1. 按历史记录排序
	sortedServers := e.sortByHistory()

	// 2. 生成带状态的标签
	labels := e.generateLabelsWithStatus(sortedServers)

	// 3. 创建选择器并选择
	selector := &Selector{
		servers: sortedServers,
		labels:  labels,
	}

	return selector.Select()
}

// sortByHistory 按历史记录排序服务器列表
// 最近使用的排在前面，其他按原顺序
func (e *EnhancedSelector) sortByHistory() []config.RedisServer {
	sorted := make([]config.RedisServer, len(e.servers))
	copy(sorted, e.servers)

	recent := e.history.GetRecent()
	recentMap := make(map[string]int) // name -> 优先级（数字越小越靠前）

	for i, name := range recent {
		recentMap[name] = i
	}

	sort.SliceStable(sorted, func(i, j int) bool {
		priorityI, hasI := recentMap[sorted[i].Name]
		priorityJ, hasJ := recentMap[sorted[j].Name]

		// 两个都在历史中：按优先级排序
		if hasI && hasJ {
			return priorityI < priorityJ
		}

		// 只有 i 在历史中：i 排前面
		if hasI {
			return true
		}

		// 只有 j 在历史中：j 排前面
		if hasJ {
			return false
		}

		// 都不在历史中：保持原顺序
		return false
	})

	return sorted
}

// generateLabelsWithStatus 生成带连接状态的标签
// 使用并发健康检查加速（所有服务器并行检查）
func (e *EnhancedSelector) generateLabelsWithStatus(servers []config.RedisServer) []string {
	labels := make([]string, len(servers))

	// 并发健康检查所有服务器
	healthResults := e.connector.BatchHealthCheck(servers)

	// 生成标签
	for i, server := range servers {
		// 检查是否是最近使用
		recentTag := ""
		if e.history.IsRecent(server.Name) {
			recentTag = " ★"
		}

		// 获取健康检查结果
		statusTag := " ✗"
		if healthResults[i] {
			statusTag = " ✓"
		}

		// 组合标签
		labels[i] = fmt.Sprintf("%s%s%s", server.Name, recentTag, statusTag)
	}

	return labels
}
