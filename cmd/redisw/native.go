package main

import (
	"io"
	"log"
	"os"

	"redisw/internal/config"
	"redisw/internal/native"
)

// ========================================
// Native Messaging Mode
// 实现 redisw native 子命令
// ========================================

// RunNative 启动 Native Messaging 模式
// 持续从 stdin 读取消息，处理后写入 stdout
func RunNative(configPath string) error {
	// 发现配置文件
	if configPath == "" {
		configPath = config.DiscoverConfigFile()
	}

	// 获取配置目录（用于历史记录）
	configDir := getConfigDir()

	// 创建处理器
	handler, err := native.NewHandler(configPath, configDir)
	if err != nil {
		return err
	}

	// 主循环：读取消息 -> 处理 -> 写入响应
	for {
		// 读取消息
		msg, err := native.ReadMessage(os.Stdin)
		if err != nil {
			if err == io.EOF {
				// 正常退出
				return nil
			}
			// 返回错误响应
			resp := native.ErrorResponse("", err.Error())
			native.WriteResponse(os.Stdout, resp)
			continue
		}

		// 处理消息
		resp := handler.Handle(msg)

		// 写入响应
		if err := native.WriteResponse(os.Stdout, resp); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
	}
}
