package config

import (
	"os"
	"path/filepath"
)

// ========================================
// Configuration File Discovery
// 配置文件发现策略：优先级从高到低
// 1. 家目录 ~/redisw_config.{yml,yaml}
// 2. XDG 配置目录 ~/.config/redisw/redisw_config.{yml,yaml}
// 3. 当前工作目录 ./redisw_config.{yml,yaml}
// ========================================

// DiscoverConfigFile 按优先级查找配置文件
// 返回第一个找到的配置文件路径，若都不存在则返回默认创建路径
func DiscoverConfigFile() string {
	// 策略 1: 家目录
	if path := checkHomeDir(); path != "" {
		return path
	}

	// 策略 2: XDG 配置目录
	if path := checkConfigDir(); path != "" {
		return path
	}

	// 策略 3: 当前工作目录
	if path := checkWorkingDir(); path != "" {
		return path
	}

	// 兜底: 返回默认创建路径 (XDG 配置目录)
	return getDefaultConfigPath()
}

// checkHomeDir 检查家目录下的配置文件
func checkHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// 检查 .yml 和 .yaml 两种扩展名
	candidates := []string{
		filepath.Join(homeDir, "redisw_config.yml"),
		filepath.Join(homeDir, "redisw_config.yaml"),
	}

	return findFirstExisting(candidates)
}

// checkConfigDir 检查 XDG 配置目录下的配置文件
func checkConfigDir() string {
	configDir := getConfigDir()
	if configDir == "" {
		return ""
	}

	candidates := []string{
		filepath.Join(configDir, "redisw_config.yml"),
		filepath.Join(configDir, "redisw_config.yaml"),
	}

	return findFirstExisting(candidates)
}

// checkWorkingDir 检查当前工作目录下的配置文件
func checkWorkingDir() string {
	candidates := []string{
		"./redisw_config.yml",
		"./redisw_config.yaml",
	}

	return findFirstExisting(candidates)
}

// getConfigDir 获取 XDG 配置目录路径
func getConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(homeDir, ".config", "redisw")
}

// getDefaultConfigPath 返回默认配置文件创建路径
func getDefaultConfigPath() string {
	configDir := getConfigDir()
	if configDir == "" {
		return "./redisw_config.yml"
	}

	return filepath.Join(configDir, "redisw_config.yml")
}

// findFirstExisting 从候选路径中找到第一个存在的文件
func findFirstExisting(paths []string) string {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// EnsureConfigDir 确保配置目录存在
func EnsureConfigDir() error {
	configDir := getConfigDir()
	if configDir == "" {
		return nil
	}

	return os.MkdirAll(configDir, 0755)
}
