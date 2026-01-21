package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// ========================================
// Configuration Loading & Management
// ========================================

// Load 从指定路径加载配置文件
// 若文件不存在则创建默认配置
func Load(filePath string) ([]RedisServer, error) {
	// 尝试打开配置文件
	file, err := os.Open(filePath)
	if err != nil {
		// 文件不存在则创建默认配置
		if os.IsNotExist(err) {
			if err := createDefaultConfig(filePath); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			// 重新加载
			return Load(filePath)
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// 解析 YAML
	var servers []RedisServer
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return servers, nil
}

// createDefaultConfig 创建默认配置文件
func createDefaultConfig(filePath string) error {
	// 确保目录存在
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	// 默认配置：本地 Redis
	defaultConfig := []RedisServer{
		{
			Name:     "localhost",
			Host:     "127.0.0.1",
			Port:     6379,
			Password: "",
		},
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// 写入 YAML
	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(defaultConfig); err != nil {
		return fmt.Errorf("failed to encode default config: %w", err)
	}

	return nil
}

// Save 保存配置到文件
func Save(filePath string, servers []RedisServer) error {
	// 确保配置目录存在
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(servers); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
