package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ========================================
// Native Messaging Host Installation
// 实现 redisw install-native 子命令
// ========================================

// NativeManifest Native Messaging Host manifest 结构
type NativeManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

// BrowserConfig 浏览器配置
type BrowserConfig struct {
	Name    string
	MacPath string
	LinPath string
}

// 支持的浏览器配置
var browsers = []BrowserConfig{
	{
		Name:    "Chrome",
		MacPath: "Library/Application Support/Google/Chrome/NativeMessagingHosts",
		LinPath: ".config/google-chrome/NativeMessagingHosts",
	},
	{
		Name:    "Firefox",
		MacPath: "Library/Application Support/Mozilla/NativeMessagingHosts",
		LinPath: ".mozilla/native-messaging-hosts",
	},
	{
		Name:    "Edge",
		MacPath: "Library/Application Support/Microsoft Edge/NativeMessagingHosts",
		LinPath: ".config/microsoft-edge/NativeMessagingHosts",
	},
}

// 默认的 Extension ID（需要在发布时更新）
var defaultAllowedOrigins = []string{
	"chrome-extension://YOUR_CHROME_EXTENSION_ID/",
}

// InstallNative 注册 Native Messaging Host 到各浏览器
func InstallNative() error {
	// 获取 redisw 二进制路径
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// 解析符号链接
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// 创建 manifest
	manifest := NativeManifest{
		Name:           "com.redisw.native",
		Description:    "Redisw Native Messaging Host",
		Path:           execPath,
		Type:           "stdio",
		AllowedOrigins: defaultAllowedOrigins,
	}

	// 序列化 manifest
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// 安装到各浏览器
	fmt.Println("Installing Native Messaging Host...")
	fmt.Printf("Binary path: %s\n\n", execPath)

	successCount := 0
	for _, browser := range browsers {
		var manifestDir string
		switch runtime.GOOS {
		case "darwin":
			manifestDir = filepath.Join(homeDir, browser.MacPath)
		case "linux":
			manifestDir = filepath.Join(homeDir, browser.LinPath)
		default:
			fmt.Printf("⚠️  %s: Unsupported OS (%s)\n", browser.Name, runtime.GOOS)
			continue
		}

		// 创建目录
		if err := os.MkdirAll(manifestDir, 0755); err != nil {
			fmt.Printf("✗  %s: Failed to create directory: %v\n", browser.Name, err)
			continue
		}

		// 写入 manifest 文件
		manifestPath := filepath.Join(manifestDir, "com.redisw.native.json")
		if err := os.WriteFile(manifestPath, manifestJSON, 0644); err != nil {
			fmt.Printf("✗  %s: Failed to write manifest: %v\n", browser.Name, err)
			continue
		}

		fmt.Printf("✓  %s: %s\n", browser.Name, manifestPath)
		successCount++
	}

	fmt.Println()
	if successCount > 0 {
		fmt.Printf("Successfully installed to %d browser(s).\n", successCount)
		fmt.Println("\nNext steps:")
		fmt.Println("1. Install the Redisw browser extension")
		fmt.Println("2. Update the extension ID in the manifest files if needed")
	} else {
		fmt.Println("No browsers were configured. Please check your system.")
	}

	return nil
}

// UninstallNative 卸载 Native Messaging Host
func UninstallNative() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	fmt.Println("Uninstalling Native Messaging Host...")

	for _, browser := range browsers {
		var manifestDir string
		switch runtime.GOOS {
		case "darwin":
			manifestDir = filepath.Join(homeDir, browser.MacPath)
		case "linux":
			manifestDir = filepath.Join(homeDir, browser.LinPath)
		default:
			continue
		}

		manifestPath := filepath.Join(manifestDir, "com.redisw.native.json")
		if err := os.Remove(manifestPath); err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("✗  %s: Failed to remove manifest: %v\n", browser.Name, err)
			}
			continue
		}

		fmt.Printf("✓  %s: Removed\n", browser.Name)
	}

	return nil
}
