# 实现计划: Redisw 浏览器插件

## 概述

本计划将 Redisw 浏览器插件功能分解为可执行的编码任务。实现分为两个主要部分：Go 端 Native Messaging 支持和浏览器插件开发。

## 任务

- [x] 1. Go 端 Native Messaging 协议层
  - [x] 1.1 创建 `internal/native/protocol.go`，实现消息编解码
    - 实现 Message 和 Response 结构体
    - 实现 ReadMessage 函数（4 字节小端序长度前缀解码）
    - 实现 WriteResponse 函数（4 字节小端序长度前缀编码）
    - _需求: 1.1, 1.3, 10.1, 10.2, 10.3, 10.4, 10.7_

  - [x] 1.2 编写协议层属性测试
    - **Property 1: 消息编码往返一致性**
    - **验证: 需求 1.3, 10.8**

  - [x] 1.3 编写协议层单元测试
    - 测试有效 JSON 解析
    - 测试无效 JSON 错误处理
    - _需求: 1.1, 1.2_

- [x] 2. Go 端消息处理器
  - [x] 2.1 创建 `internal/native/handler.go`，实现请求路由
    - 实现 Handler 结构体
    - 实现 Handle 方法，根据 action 分发请求
    - 实现错误响应生成
    - _需求: 1.1, 10.5, 10.6_

  - [x] 2.2 实现 list_servers 处理器
    - 调用现有 config.Load 获取服务器列表
    - 返回服务器配置数组
    - _需求: 3.1_

  - [x] 2.3 实现 add_server 处理器
    - 验证服务器名称唯一性
    - 调用 config.Save 持久化
    - _需求: 3.2, 3.5, 3.7_

  - [x] 2.4 实现 update_server 处理器
    - 验证服务器存在
    - 更新配置并持久化
    - _需求: 3.3, 3.6_

  - [x] 2.5 实现 delete_server 处理器
    - 验证服务器存在
    - 删除配置并持久化
    - _需求: 3.4, 3.6_

  - [x] 2.6 编写服务器配置属性测试
    - **Property 6: 服务器配置往返一致性**
    - **Property 7: 服务器更新持久化**
    - **Property 8: 服务器删除持久化**
    - **验证: 需求 3.1, 3.2, 3.3, 3.4, 3.7**

- [x] 3. 检查点 - 确保配置管理测试通过
  - 确保所有测试通过，如有问题请询问用户

- [x] 4. Go 端健康检查和命令执行
  - [x] 4.1 实现 health_check 处理器
    - 复用现有 connector.BatchHealthCheck
    - 返回服务器名称到健康状态的映射
    - _需求: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [x] 4.2 实现 execute_command 处理器
    - 验证服务器存在
    - 创建 Redis 连接并执行命令
    - 格式化返回结果
    - _需求: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_

  - [x] 4.3 实现 flush_db 和 flush_all 处理器
    - 验证 confirm 字段为 true
    - 执行 FLUSHDB 或 FLUSHALL 命令
    - 返回操作结果
    - _需求: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 4.4 编写危险操作属性测试
    - **Property 10: 危险操作确认验证**
    - **验证: 需求 6.3, 6.4**

- [x] 5. Go 端历史记录和导入功能
  - [x] 5.1 实现 get_history 和 record_history 处理器
    - 复用现有 history.Manager
    - _需求: 7.1, 7.2, 7.3, 7.4_

  - [x] 5.2 实现 import_servers 处理器
    - 支持 YAML 和 JSON 格式解析
    - 实现冲突处理策略（skip/overwrite/rename）
    - 返回导入统计
    - _需求: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6_

  - [x] 5.3 编写历史记录属性测试
    - **Property 11: 历史记录往返一致性**
    - **Property 12: 历史记录容量限制**
    - **Property 13: 历史记录去重和排序**
    - **验证: 需求 7.1, 7.2, 7.3, 7.4**

  - [x] 5.4 编写导入功能属性测试
    - **Property 18: 导入格式解析**
    - **Property 19: 导入冲突处理**
    - **Property 20: 导入结果统计**
    - **验证: 需求 11.1, 11.2, 11.3, 11.4, 11.5**

- [x] 6. Go 端命令入口
  - [x] 6.1 创建 `cmd/redisw/native.go`，实现 native 子命令
    - 解析命令行参数
    - 创建 Handler 实例
    - 主循环：读取消息 -> 处理 -> 写入响应
    - _需求: 1.4, 1.5_

  - [x] 6.2 创建 `cmd/redisw/install.go`，实现 install-native 子命令
    - 检测操作系统
    - 生成 Native Messaging Host manifest
    - 写入各浏览器的 NativeMessagingHosts 目录
    - _需求: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

  - [x] 6.3 更新 `cmd/redisw/main.go`，集成新子命令
    - 添加 native 和 install-native 子命令路由
    - _需求: 1.4, 2.1_

  - [x] 6.4 编写安装命令属性测试
    - **Property 4: Manifest 内容完整性**
    - **Property 5: 安装命令幂等性**
    - **验证: 需求 2.4, 2.5**

- [x] 7. 检查点 - 确保 Go 端所有测试通过
  - 确保所有测试通过，如有问题请询问用户

- [x] 8. 浏览器插件基础结构
  - [x] 8.1 创建 `extension/manifest.json`
    - 配置 manifest_version 3
    - 声明 nativeMessaging 和 storage 权限
    - 配置 background service worker 和 popup
    - _需求: 9.1, 9.2, 9.3, 9.4_

  - [x] 8.2 创建 `extension/background.js`，实现 NativeClient 类
    - 实现 connect/disconnect 方法
    - 实现 request 方法（Promise 封装）
    - 实现请求-响应匹配逻辑
    - _需求: 1.1, 10.2, 10.4_

  - [x] 8.3 创建 API 封装函数
    - listServers, addServer, updateServer, deleteServer
    - healthCheck, executeCommand
    - flushDb, flushAll
    - getHistory, recordHistory
    - importServers
    - _需求: 3.1-3.7, 4.1, 5.1, 6.1-6.5, 7.1-7.4, 11.1-11.6_

  - [x] 8.4 编写 Background Script 单元测试
    - 测试请求-响应 ID 匹配
    - 测试错误处理
    - **Property 16: 请求-响应 ID 匹配**
    - **验证: 需求 10.2, 10.4**

- [x] 9. 浏览器插件 Popup UI - 服务器列表
  - [x] 9.1 创建 `extension/popup/popup.html` 基础结构
    - 服务器列表视图
    - 命令执行视图
    - 服务器表单视图
    - 导入视图
    - _需求: 8.1_

  - [x] 9.2 创建 `extension/popup/popup.css` 样式
    - 服务器列表样式
    - 终端样式（深色背景、等宽字体、语法高亮）
    - 表单样式
    - 对话框样式
    - _需求: 8.2_

  - [x] 9.3 实现服务器列表渲染
    - 显示服务器名称、健康状态（✓/✗）、最近使用标记（★）
    - 实现模糊搜索过滤
    - _需求: 8.2, 8.3_

  - [x] 9.4 编写服务器列表属性测试
    - **Property 14: 服务器列表渲染完整性**
    - **Property 15: 模糊搜索过滤**
    - **验证: 需求 8.2, 8.3**

- [x] 10. 浏览器插件 Popup UI - 服务器管理
  - [x] 10.1 实现添加服务器表单
    - 名称、主机、端口、密码输入
    - 表单验证
    - _需求: 8.4_

  - [x] 10.2 实现编辑服务器表单
    - 预填充现有配置
    - 更新逻辑
    - _需求: 8.5_

  - [x] 10.3 实现删除确认对话框
    - 显示服务器信息
    - 确认/取消按钮
    - _需求: 8.6_

- [x] 11. 浏览器插件 Popup UI - 命令执行
  - [x] 11.1 实现终端界面
    - 深色背景终端区域
    - 命令历史显示
    - 当前输入行（带闪烁光标）
    - _需求: 8.7, 8.8_

  - [x] 11.2 实现命令输入和执行
    - Enter 键执行命令
    - 上下箭头浏览历史
    - 结果格式化显示（语法高亮）
    - _需求: 5.1, 5.2_

  - [x] 11.3 实现危险操作按钮和确认对话框
    - FLUSHDB 和 FLUSHALL 按钮
    - 二次确认对话框
    - _需求: 8.9, 6.1, 6.2_

- [x] 12. 浏览器插件 Popup UI - 批量导入
  - [x] 12.1 实现文件选择和拖拽上传
    - 支持 .yml, .yaml, .json 文件
    - 拖拽区域样式
    - _需求: 11.7_

  - [x] 12.2 实现冲突策略选择
    - 跳过/覆盖/重命名单选按钮
    - _需求: 11.4_

  - [x] 12.3 实现导入预览和确认
    - 显示待导入服务器列表
    - 标记冲突项
    - 导入结果显示
    - _需求: 11.5, 11.8_

- [x] 13. 检查点 - 确保浏览器插件测试通过
  - 确保所有测试通过，如有问题请询问用户

- [x] 14. 跨浏览器兼容性
  - [x] 14.1 添加 browser-polyfill
    - 引入 webextension-polyfill 库
    - 处理 Chrome/Firefox/Edge API 差异
    - _需求: 9.5_

  - [x] 14.2 创建插件图标
    - 16x16, 48x48, 128x128 尺寸
    - Redis 风格设计
    - _需求: 9.1_

- [x] 15. 最终检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户

## 备注

- 每个任务引用具体需求以确保可追溯性
- 检查点确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
