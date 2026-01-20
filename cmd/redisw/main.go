package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"redisw/internal/config"
	"redisw/internal/connector"
	"redisw/internal/history"
	"redisw/internal/ui"
)

// ========================================
// Redisw - Redis Connection Switcher
// 极简主义设计：快速切换 Redis 服务器
// ========================================

var (
	configPath = flag.String("config", "", "path to config file (auto-discover if not specified)")
)

func main() {
	// 检查子命令
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "native":
			// Native Messaging 模式
			if err := RunNative(*configPath); err != nil {
				log.Fatalf("Native mode error: %v", err)
			}
			return
		case "install-native":
			// 安装 Native Messaging Host
			if err := InstallNative(); err != nil {
				log.Fatalf("Install error: %v", err)
			}
			return
		case "uninstall-native":
			// 卸载 Native Messaging Host
			if err := UninstallNative(); err != nil {
				log.Fatalf("Uninstall error: %v", err)
			}
			return
		}
	}

	flag.Parse()

	// 1. 发现并加载配置
	if *configPath == "" {
		*configPath = config.DiscoverConfigFile()
	}

	servers, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if len(servers) == 0 {
		fmt.Println("No Redis servers configured. Please edit:", *configPath)
		os.Exit(1)
	}

	// 2. 初始化组件
	conn := connector.NewConnector()
	historyMgr, err := history.NewManager(getConfigDir())
	if err != nil {
		log.Printf("Warning: failed to load history: %v", err)
	}

	// 3. 主循环：选择 -> 连接 -> 返回选择
	for {
		// 选择服务器（带历史记录排序和状态显示）
		selector := ui.NewEnhancedSelector(servers, historyMgr, conn)
		selectedServer := selector.Select()

		// 用户取消选择，退出程序
		if selectedServer == nil {
			fmt.Println("\nGoodbye!")
			return
		}

		// 记录历史
		if historyMgr != nil {
			historyMgr.Record(selectedServer.Name)
		}

		// 连接到选中的服务器
		fmt.Printf("\nConnecting to %s (%s:%d)...\n",
			selectedServer.Name, selectedServer.Host, selectedServer.Port)

		if err := conn.Connect(selectedServer); err != nil {
			fmt.Printf("Connection failed: %v\n\n", err)
			continue
		}

		// 连接断开后返回选择界面
		fmt.Println()
	}
}

// getConfigDir 获取配置目录（用于历史记录）
func getConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return homeDir + "/.config/redisw"
}
