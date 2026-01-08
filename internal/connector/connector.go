package connector

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"redisw/internal/config"
)

// ========================================
// Redis Connection Management
// ========================================

// Connector Redis 连接器
type Connector struct {
	checkTimeout time.Duration // 健康检查超时时间
}

// NewConnector 创建连接器
func NewConnector() *Connector {
	return &Connector{
		checkTimeout: 2 * time.Second,
	}
}

// HealthCheck 快速健康检查（TCP 连接测试）
// 返回 true 表示服务器可达
func (c *Connector) HealthCheck(server *config.RedisServer) bool {
	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	conn, err := net.DialTimeout("tcp", addr, c.checkTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Connect 连接到 Redis 服务器（使用 redis-cli）
// 该函数会阻塞直到用户退出 redis-cli
func (c *Connector) Connect(server *config.RedisServer) error {
	// 构建 redis-cli 命令
	args := []string{
		"-h", server.Host,
		"-p", fmt.Sprintf("%d", server.Port),
		"-c", // 支持集群模式
	}

	// 添加密码参数
	if server.Password != "" {
		args = append(args, "-a", server.Password)
	}

	cmd := exec.Command("redis-cli", args...)

	// 设置标准输入输出（直接连接到终端）
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 启动 redis-cli
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start redis-cli: %w", err)
	}

	// 等待 redis-cli 退出
	if err := cmd.Wait(); err != nil {
		// 用户按 Ctrl+C 退出是正常情况，不报错
		if _, ok := err.(*exec.ExitError); ok {
			return nil
		}
		return fmt.Errorf("redis-cli exited with error: %w", err)
	}

	return nil
}

// ConnectWithCheck 连接前先进行健康检查
func (c *Connector) ConnectWithCheck(server *config.RedisServer) error {
	// 健康检查
	if !c.HealthCheck(server) {
		return fmt.Errorf("server %s (%s:%d) is unreachable",
			server.Name, server.Host, server.Port)
	}

	// 连接
	return c.Connect(server)
}

// ========================================
// Batch Health Check - 并发健康检查
// ========================================

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Index     int  // 服务器在列表中的索引
	Reachable bool // 是否可达
}

// BatchHealthCheck 并发检查多个服务器的健康状态
// 返回每个服务器的可达性结果（索引对应输入列表）
func (c *Connector) BatchHealthCheck(servers []config.RedisServer) []bool {
	results := make([]bool, len(servers))

	// 使用 channel 收集结果
	resultChan := make(chan HealthCheckResult, len(servers))

	// 并发检查所有服务器
	var wg sync.WaitGroup
	for i, server := range servers {
		wg.Add(1)
		go func(index int, srv config.RedisServer) {
			defer wg.Done()
			reachable := c.HealthCheck(&srv)
			resultChan <- HealthCheckResult{
				Index:     index,
				Reachable: reachable,
			}
		}(i, server)
	}

	// 等待所有检查完成并关闭 channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for result := range resultChan {
		results[result.Index] = result.Reachable
	}

	return results
}

