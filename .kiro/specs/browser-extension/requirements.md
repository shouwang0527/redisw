# 需求文档

## 简介

本文档定义了 Redisw 浏览器插件版本的需求。该插件通过 Native Messaging 协议与本地 redisw 进程通信，让用户可以在浏览器中管理 Redis 服务器连接，执行 Redis 命令，并与 CLI 版本共享配置。

## 术语表

- **Extension**: 浏览器插件，使用 WebExtension API 开发的跨浏览器扩展程序
- **Native_Host**: 本地原生应用程序，即 redisw 二进制文件，通过 Native Messaging 与 Extension 通信
- **Native_Messaging**: 浏览器提供的原生消息传递机制，允许 Extension 与本地应用程序通过 stdin/stdout 交换 JSON 消息
- **Message_Protocol**: Extension 与 Native_Host 之间的 JSON 消息格式规范
- **Config_File**: 配置文件，位于 ~/.config/redisw/redisw_config.yml
- **History_File**: 历史记录文件，位于 ~/.config/redisw/history.json
- **Server_Entry**: 单个 Redis 服务器配置条目，包含名称、主机、端口、密码
- **Health_Status**: 服务器健康状态，表示服务器是否可达

## 需求

### 需求 1：Native Messaging 模式

**用户故事：** 作为浏览器插件，我希望通过 Native Messaging 与本地 redisw 进程通信，以便在浏览器中使用 Redis 管理功能。

#### 验收标准

1. WHEN Extension 发送 JSON 消息到 stdin THEN Native_Host SHALL 解析消息并返回 JSON 响应到 stdout
2. WHEN Native_Host 接收到格式错误的 JSON THEN Native_Host SHALL 返回包含错误描述的 JSON 响应
3. THE Message_Protocol SHALL 使用 4 字节小端序长度前缀编码消息长度
4. WHEN Native_Host 启动 THEN Native_Host SHALL 持续监听 stdin 直到收到 EOF 或退出命令
5. THE Native_Host SHALL 在 5 秒内响应每个请求

### 需求 2：Native Host 注册

**用户故事：** 作为用户，我希望 redisw 能自动注册到浏览器的 Native Messaging Hosts 目录，以便插件能够自动启动 redisw 进程。

#### 验收标准

1. WHEN 用户执行 `redisw install-native` THEN Native_Host SHALL 在 Chrome 的 Native Messaging Hosts 目录创建 manifest 文件
2. WHEN 用户执行 `redisw install-native` THEN Native_Host SHALL 在 Firefox 的 Native Messaging Hosts 目录创建 manifest 文件
3. WHEN 用户执行 `redisw install-native` THEN Native_Host SHALL 在 Edge 的 Native Messaging Hosts 目录创建 manifest 文件
4. THE manifest 文件 SHALL 包含正确的 redisw 二进制路径、名称和允许的 Extension ID
5. IF manifest 文件已存在 THEN Native_Host SHALL 更新文件内容而非报错
6. WHEN 注册完成 THEN Native_Host SHALL 输出各浏览器的注册状态

### 需求 3：配置管理 API

**用户故事：** 作为浏览器插件用户，我希望能够查看、添加、编辑和删除 Redis 服务器配置，以便管理我的服务器列表。

#### 验收标准

1. WHEN Extension 发送 list_servers 请求 THEN Native_Host SHALL 返回所有服务器配置列表
2. WHEN Extension 发送 add_server 请求 THEN Native_Host SHALL 将新服务器添加到 Config_File
3. WHEN Extension 发送 update_server 请求 THEN Native_Host SHALL 更新 Config_File 中对应的服务器配置
4. WHEN Extension 发送 delete_server 请求 THEN Native_Host SHALL 从 Config_File 中删除对应的服务器
5. IF 添加的服务器名称已存在 THEN Native_Host SHALL 返回错误响应
6. IF 更新或删除的服务器不存在 THEN Native_Host SHALL 返回错误响应
7. WHEN 配置变更成功 THEN Native_Host SHALL 立即持久化到 Config_File

### 需求 4：健康检查 API

**用户故事：** 作为浏览器插件用户，我希望能够查看每个服务器的健康状态，以便快速了解哪些服务器可用。

#### 验收标准

1. WHEN Extension 发送 health_check 请求 THEN Native_Host SHALL 返回所有服务器的 Health_Status
2. THE Native_Host SHALL 并发检查所有服务器的健康状态
3. THE 健康检查 SHALL 在 2 秒超时内完成单个服务器检查
4. WHEN 服务器可达 THEN Health_Status SHALL 为 true
5. WHEN 服务器不可达 THEN Health_Status SHALL 为 false

### 需求 5：Redis 命令执行 API

**用户故事：** 作为浏览器插件用户，我希望能够在指定服务器上执行 Redis 命令，以便直接在浏览器中操作 Redis。

#### 验收标准

1. WHEN Extension 发送 execute_command 请求 THEN Native_Host SHALL 在指定服务器上执行 Redis 命令
2. WHEN 命令执行成功 THEN Native_Host SHALL 返回格式化的命令结果
3. IF 指定的服务器不存在 THEN Native_Host SHALL 返回错误响应
4. IF 服务器连接失败 THEN Native_Host SHALL 返回包含错误原因的响应
5. IF 命令执行失败 THEN Native_Host SHALL 返回 Redis 错误信息
6. THE Native_Host SHALL 支持所有标准 Redis 命令

### 需求 6：危险操作 API

**用户故事：** 作为浏览器插件用户，我希望能够执行 FLUSHDB 和 FLUSHALL 操作，并且系统能够防止误操作。

#### 验收标准

1. WHEN Extension 发送 flush_db 请求 THEN Native_Host SHALL 执行 FLUSHDB 命令清空当前数据库
2. WHEN Extension 发送 flush_all 请求 THEN Native_Host SHALL 执行 FLUSHALL 命令清空所有数据库
3. THE flush_db 和 flush_all 请求 SHALL 包含 confirm 字段，值必须为 true
4. IF confirm 字段不为 true THEN Native_Host SHALL 拒绝执行并返回错误
5. WHEN 危险操作执行成功 THEN Native_Host SHALL 返回成功响应和受影响的键数量

### 需求 7：历史记录 API

**用户故事：** 作为浏览器插件用户，我希望能够查看最近使用的服务器，以便快速访问常用服务器。

#### 验收标准

1. WHEN Extension 发送 get_history 请求 THEN Native_Host SHALL 返回最近使用的服务器名称列表
2. WHEN Extension 发送 record_history 请求 THEN Native_Host SHALL 将服务器名称记录到历史
3. THE 历史记录 SHALL 最多保留 10 条记录
4. WHEN 记录重复的服务器 THEN Native_Host SHALL 将其移动到列表头部而非重复添加

### 需求 8：浏览器插件 UI

**用户故事：** 作为浏览器插件用户，我希望有一个直观的界面来管理 Redis 服务器和执行命令。

#### 验收标准

1. WHEN 用户点击插件图标 THEN Extension SHALL 显示服务器列表弹窗
2. THE 服务器列表 SHALL 显示每个服务器的名称、健康状态（✓/✗）和最近使用标记（★）
3. WHEN 用户在搜索框输入 THEN Extension SHALL 模糊过滤服务器列表
4. WHEN 用户点击添加按钮 THEN Extension SHALL 显示添加服务器表单
5. WHEN 用户点击服务器的编辑按钮 THEN Extension SHALL 显示编辑服务器表单
6. WHEN 用户点击服务器的删除按钮 THEN Extension SHALL 显示确认对话框
7. WHEN 用户选择服务器 THEN Extension SHALL 显示命令执行界面
8. THE 命令执行界面 SHALL 包含命令输入框和结果显示区域
9. WHEN 用户点击 FLUSHDB 或 FLUSHALL 按钮 THEN Extension SHALL 显示二次确认对话框

### 需求 9：跨浏览器兼容性

**用户故事：** 作为用户，我希望插件能在 Chrome、Firefox 和 Edge 上运行，以便在我常用的浏览器中使用。

#### 验收标准

1. THE Extension SHALL 使用 WebExtension API 标准开发
2. THE Extension SHALL 兼容 Chrome 88+ 版本
3. THE Extension SHALL 兼容 Firefox 78+ 版本
4. THE Extension SHALL 兼容 Edge 88+ 版本
5. THE Extension SHALL 使用 browser polyfill 处理 API 差异

### 需求 10：消息协议规范

**用户故事：** 作为开发者，我希望有清晰的消息协议规范，以便正确实现 Extension 和 Native_Host 之间的通信。

#### 验收标准

1. THE 请求消息 SHALL 包含 action 字段指定操作类型
2. THE 请求消息 SHALL 包含 id 字段用于请求-响应匹配
3. THE 响应消息 SHALL 包含 success 布尔字段表示操作是否成功
4. THE 响应消息 SHALL 包含 id 字段与请求 id 匹配
5. IF 操作失败 THEN 响应消息 SHALL 包含 error 字段描述错误原因
6. IF 操作成功 THEN 响应消息 SHALL 包含 data 字段携带结果数据
7. THE Message_Protocol SHALL 使用 JSON 格式序列化消息
8. FOR ALL 有效请求消息，解析后再序列化 SHALL 产生等价的消息结构（往返一致性）

### 需求 11：批量导入配置

**用户故事：** 作为用户，我希望能够批量导入服务器配置，以便快速迁移或共享配置。

#### 验收标准

1. WHEN Extension 发送 import_servers 请求 THEN Native_Host SHALL 解析并导入配置数据
2. THE 导入格式 SHALL 支持 YAML 格式（与 CLI 配置文件格式一致）
3. THE 导入格式 SHALL 支持 JSON 格式
4. WHEN 导入的服务器名称与现有配置冲突 THEN Native_Host SHALL 根据策略处理（跳过/覆盖/重命名）
5. WHEN 导入完成 THEN Native_Host SHALL 返回导入结果统计（成功数、跳过数、失败数）
6. IF 导入数据格式无效 THEN Native_Host SHALL 返回详细的错误信息
7. THE Extension SHALL 提供文件选择器让用户选择配置文件
8. THE Extension SHALL 在导入前显示预览，让用户确认要导入的服务器列表
