package connector

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
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

// Connect 连接到 Redis 服务器（使用内置 go-redis 客户端）
// 该函数会阻塞直到用户退出交互式会话
func (c *Connector) Connect(server *config.RedisServer) error {
	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", server.Host, server.Port),
		Password: server.Password,
		DB:       0,
	})
	defer client.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// 显示欢迎信息
	fmt.Printf("Connected to Redis at %s:%d\n", server.Host, server.Port)
	fmt.Println("Type 'exit' or 'quit' to disconnect, or press Ctrl+C")
	fmt.Println()

	// 启动交互式 REPL
	return c.startREPL(client, server)
}

// startREPL 启动交互式 Redis REPL
func (c *Connector) startREPL(client *redis.Client, server *config.RedisServer) error {
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		// 显示提示符
		fmt.Printf("%s:%d> ", server.Host, server.Port)

		// 读取用户输入
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// 去除首尾空白字符
		input = strings.TrimSpace(input)

		// 空输入，继续
		if input == "" {
			continue
		}

		// 退出命令
		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			fmt.Println("Goodbye!")
			return nil
		}

		// 解析命令和参数
		args := parseCommand(input)
		if len(args) == 0 {
			continue
		}

		// 执行 Redis 命令
		result, err := client.Do(ctx, args...).Result()
		if err != nil {
			fmt.Printf("(error) %v\n", err)
			continue
		}

		// 格式化输出结果
		printResult(result)
	}
}

// parseCommand 解析命令字符串为参数列表
// 支持引号包裹的参数 (例如: SET "my key" "my value")
func parseCommand(input string) []interface{} {
	var args []interface{}
	var current strings.Builder
	inQuote := false
	escapeNext := false

	for i, char := range input {
		if escapeNext {
			current.WriteRune(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inQuote = !inQuote
			continue
		}

		if char == ' ' && !inQuote {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(char)

		// 最后一个字符，追加当前积累的字符串
		if i == len(input)-1 && current.Len() > 0 {
			args = append(args, current.String())
		}
	}

	return args
}

// printResult 格式化打印 Redis 命令结果
func printResult(result interface{}) {
	switch v := result.(type) {
	case nil:
		fmt.Println("(nil)")
	case string:
		fmt.Printf("\"%s\"\n", v)
	case int64:
		fmt.Printf("(integer) %d\n", v)
	case []interface{}:
		if len(v) == 0 {
			fmt.Println("(empty array)")
		} else {
			for i, item := range v {
				fmt.Printf("%d) ", i+1)
				printResult(item)
			}
		}
	case map[string]interface{}:
		if len(v) == 0 {
			fmt.Println("(empty hash)")
		} else {
			for k, val := range v {
				fmt.Printf("%s: ", k)
				printResult(val)
			}
		}
	default:
		fmt.Printf("%v\n", v)
	}
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
