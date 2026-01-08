package ui

import (
	"fmt"
	"strings"

	"redisw/internal/config"

	"github.com/manifoldco/promptui"
)

// ========================================
// Interactive Server Selection UI
// ========================================

// Selector 交互式服务器选择器
type Selector struct {
	servers []config.RedisServer
	labels  []string // 用于显示的标签（包含状态提示）
}

// NewSelector 创建选择器
func NewSelector(servers []config.RedisServer) *Selector {
	return &Selector{
		servers: servers,
		labels:  extractLabels(servers),
	}
}

// SetLabels 更新显示标签（用于显示连接状态）
func (s *Selector) SetLabels(labels []string) {
	s.labels = labels
}

// Select 显示交互式选择界面，返回选中的服务器
// 返回 nil 表示用户取消选择
func (s *Selector) Select() *config.RedisServer {
	if len(s.servers) == 0 {
		fmt.Println("No Redis servers configured")
		return nil
	}

	prompt := promptui.Select{
		Label:    "Select Redis Server",
		Items:    s.labels,
		Size:     len(s.servers),
		Searcher: s.searchFunc(),
		Templates: &promptui.SelectTemplates{
			Label:    "✨ {{ . | green}}",
			Selected: `{{ "➤ " | green }}{{ . | faint }}`,
			Active:   `{{ "➤ " | green }}{{ . | green }}`,
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		// 用户按 Ctrl+C 取消
		return nil
	}

	return &s.servers[index]
}

// searchFunc 返回搜索函数（支持模糊搜索）
func (s *Selector) searchFunc() func(string, int) bool {
	return func(input string, index int) bool {
		server := s.servers[index]
		// 移除空格，转小写，进行模糊匹配
		name := strings.ReplaceAll(strings.ToLower(server.Name), " ", "")
		query := strings.ToLower(input)
		return strings.Contains(name, query)
	}
}

// extractLabels 从服务器列表提取显示标签
func extractLabels(servers []config.RedisServer) []string {
	labels := make([]string, len(servers))
	for i, server := range servers {
		labels[i] = server.Name
	}
	return labels
}
