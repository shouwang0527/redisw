package config

// ========================================
// Redis Server Configuration
// ========================================

// RedisServer 表示单个 Redis 服务器的配置
type RedisServer struct {
	Name     string `yaml:"name"`     // 服务器显示名称
	Host     string `yaml:"host"`     // 主机地址
	Port     int    `yaml:"port"`     // 端口号
	Password string `yaml:"password"` // 认证密码（可选）
}
