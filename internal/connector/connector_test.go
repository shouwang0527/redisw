package connector

import (
	"net"
	"testing"
	"time"

	"redisw/internal/config"
)

// ========================================
// Connector Tests
// ========================================

func TestNewConnector(t *testing.T) {
	conn := NewConnector()

	if conn == nil {
		t.Fatal("Expected non-nil connector")
	}

	if conn.checkTimeout != 2*time.Second {
		t.Errorf("Expected timeout 2s, got %v", conn.checkTimeout)
	}
}

func TestHealthCheck(t *testing.T) {
	conn := NewConnector()

	t.Run("check unreachable server", func(t *testing.T) {
		server := &config.RedisServer{
			Name: "unreachable",
			Host: "192.0.2.1", // TEST-NET-1: 保证不可达
			Port: 9999,
		}

		if conn.HealthCheck(server) {
			t.Error("Expected health check to fail for unreachable server")
		}
	})

	t.Run("check localhost", func(t *testing.T) {
		// 启动一个临时 TCP 服务器
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Skip("Cannot start test server")
		}
		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port

		server := &config.RedisServer{
			Name: "test-server",
			Host: "127.0.0.1",
			Port: port,
		}

		if !conn.HealthCheck(server) {
			t.Error("Expected health check to succeed for reachable server")
		}
	})

	t.Run("check with custom timeout", func(t *testing.T) {
		conn := &Connector{
			checkTimeout: 100 * time.Millisecond,
		}

		server := &config.RedisServer{
			Name: "timeout-test",
			Host: "192.0.2.1",
			Port: 9999,
		}

		start := time.Now()
		conn.HealthCheck(server)
		elapsed := time.Since(start)

		// 应该在 200ms 内完成（100ms 超时 + 一些余量）
		if elapsed > 300*time.Millisecond {
			t.Errorf("Health check took too long: %v", elapsed)
		}
	})
}

func TestConnectWithCheck(t *testing.T) {
	conn := NewConnector()

	t.Run("fail on unreachable server", func(t *testing.T) {
		server := &config.RedisServer{
			Name: "unreachable",
			Host: "192.0.2.1",
			Port: 9999,
		}

		err := conn.ConnectWithCheck(server)
		if err == nil {
			t.Error("Expected error when connecting to unreachable server")
		}

		if err != nil && err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	})
}

func TestBatchHealthCheck(t *testing.T) {
	conn := NewConnector()

	t.Run("check multiple servers concurrently", func(t *testing.T) {
		// 启动两个测试服务器
		listener1, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Skip("Cannot start test server 1")
		}
		defer listener1.Close()

		listener2, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Skip("Cannot start test server 2")
		}
		defer listener2.Close()

		port1 := listener1.Addr().(*net.TCPAddr).Port
		port2 := listener2.Addr().(*net.TCPAddr).Port

		// 创建测试服务器列表（2 个可达 + 1 个不可达）
		servers := []config.RedisServer{
			{Name: "server1", Host: "127.0.0.1", Port: port1},
			{Name: "server2", Host: "127.0.0.1", Port: port2},
			{Name: "unreachable", Host: "192.0.2.1", Port: 9999},
		}

		// 执行批量检查
		start := time.Now()
		results := conn.BatchHealthCheck(servers)
		elapsed := time.Since(start)

		// 验证结果数量
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// 验证可达性
		if !results[0] {
			t.Error("Expected server1 to be reachable")
		}
		if !results[1] {
			t.Error("Expected server2 to be reachable")
		}
		if results[2] {
			t.Error("Expected unreachable server to be unreachable")
		}

		// 验证并发性能：并发检查应该比串行快
		// 串行需要 2 * 0s（可达）+ 2s（不可达）≈ 2s
		// 并发应该只需要 max(0s, 0s, 2s) ≈ 2s
		if elapsed > 3*time.Second {
			t.Errorf("Batch check took too long: %v (expected ~2s)", elapsed)
		}
	})

	t.Run("empty server list", func(t *testing.T) {
		results := conn.BatchHealthCheck([]config.RedisServer{})
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty list, got %d", len(results))
		}
	})

	t.Run("single server", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Skip("Cannot start test server")
		}
		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port
		servers := []config.RedisServer{
			{Name: "single", Host: "127.0.0.1", Port: port},
		}

		results := conn.BatchHealthCheck(servers)
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if !results[0] {
			t.Error("Expected single server to be reachable")
		}
	})
}

// ========================================
// Benchmark Tests
// ========================================

func BenchmarkHealthCheck(b *testing.B) {
	conn := NewConnector()

	// 启动测试服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Skip("Cannot start test server")
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	server := &config.RedisServer{
		Host: "127.0.0.1",
		Port: port,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn.HealthCheck(server)
	}
}

func BenchmarkHealthCheckUnreachable(b *testing.B) {
	conn := &Connector{
		checkTimeout: 10 * time.Millisecond, // 短超时用于基准测试
	}

	server := &config.RedisServer{
		Host: "192.0.2.1",
		Port: 9999,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn.HealthCheck(server)
	}
}

// BenchmarkBatchHealthCheck 测试批量健康检查性能
func BenchmarkBatchHealthCheck(b *testing.B) {
	conn := NewConnector()

	// 创建多个不可达服务器（模拟真实场景）
	servers := []config.RedisServer{
		{Host: "192.0.2.1", Port: 6379},
		{Host: "192.0.2.2", Port: 6379},
		{Host: "192.0.2.3", Port: 6379},
	}

	// 使用短超时以加速基准测试
	conn.checkTimeout = 50 * time.Millisecond

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn.BatchHealthCheck(servers)
	}
}

// BenchmarkBatchVsSequential 对比并发检查 vs 串行检查的性能
func BenchmarkBatchVsSequential(b *testing.B) {
	conn := &Connector{
		checkTimeout: 50 * time.Millisecond,
	}

	servers := []config.RedisServer{
		{Host: "192.0.2.1", Port: 6379},
		{Host: "192.0.2.2", Port: 6379},
		{Host: "192.0.2.3", Port: 6379},
	}

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := range servers {
				conn.HealthCheck(&servers[j])
			}
		}
	})

	b.Run("Batch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			conn.BatchHealthCheck(servers)
		}
	})
}

