# 设计文档

## 概述

本设计文档描述 Redisw 浏览器插件的技术架构和实现方案。系统由两部分组成：

1. **Go 端扩展**：为现有 redisw 添加 `native` 和 `install-native` 命令
2. **浏览器插件**：使用 WebExtension API 开发的跨浏览器扩展

两者通过 Native Messaging 协议通信，共享同一份配置文件。

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        浏览器插件 (Extension)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │
│  │  Popup   │  │ Options  │  │Background│  │ Native Messaging │ │
│  │   UI     │  │  Page    │  │  Script  │  │     Client       │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬─────────┘ │
│       │             │             │                  │           │
│       └─────────────┴─────────────┴──────────────────┘           │
│                              │                                    │
└──────────────────────────────┼────────────────────────────────────┘
                               │ Native Messaging (stdin/stdout JSON)
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Native Host (redisw native)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐│
│  │   Message    │  │   Handler    │  │      Existing Modules    ││
│  │   Protocol   │  │   Router     │  │  ┌────────┐ ┌─────────┐  ││
│  │   (IO)       │  │              │  │  │ config │ │connector│  ││
│  └──────┬───────┘  └──────┬───────┘  │  └────────┘ └─────────┘  ││
│         │                 │          │  ┌────────┐              ││
│         └─────────────────┘          │  │history │              ││
│                   │                  │  └────────┘              ││
│                   └──────────────────┴──────────────────────────┘│
└──────────────────────────────────────────────────────────────────┘
                               │
                               ▼
                    ~/.config/redisw/
                    ├── redisw_config.yml
                    └── history.json
```

## 组件和接口

### Go 端组件

#### 1. Native Messaging 协议层 (`internal/native/protocol.go`)

负责 Native Messaging 消息的编解码。

```go
// Message 表示 Native Messaging 消息
type Message struct {
    ID      string          `json:"id"`      // 请求 ID，用于匹配响应
    Action  string          `json:"action"`  // 操作类型
    Params  json.RawMessage `json:"params"`  // 操作参数（可选）
}

// Response 表示响应消息
type Response struct {
    ID      string      `json:"id"`      // 对应请求 ID
    Success bool        `json:"success"` // 操作是否成功
    Data    interface{} `json:"data"`    // 成功时的数据
    Error   string      `json:"error"`   // 失败时的错误信息
}

// ReadMessage 从 stdin 读取一条消息
// Native Messaging 使用 4 字节小端序长度前缀
func ReadMessage(r io.Reader) (*Message, error)

// WriteResponse 向 stdout 写入响应
func WriteResponse(w io.Writer, resp *Response) error
```

#### 2. 消息处理路由 (`internal/native/handler.go`)

路由和处理各类请求。

```go
// Handler 处理 Native Messaging 请求
type Handler struct {
    configPath string
    connector  *connector.Connector
    historyMgr *history.Manager
}

// NewHandler 创建处理器
func NewHandler(configPath string) (*Handler, error)

// Handle 处理单个请求并返回响应
func (h *Handler) Handle(msg *Message) *Response

// 支持的 Action:
// - list_servers: 列出所有服务器
// - add_server: 添加服务器
// - update_server: 更新服务器
// - delete_server: 删除服务器
// - health_check: 健康检查
// - execute_command: 执行 Redis 命令
// - flush_db: 清空当前数据库
// - flush_all: 清空所有数据库
// - get_history: 获取历史记录
// - record_history: 记录历史
// - import_servers: 批量导入服务器配置
```

#### 3. Native 命令入口 (`cmd/redisw/native.go`)

实现 `redisw native` 子命令。

```go
// RunNative 启动 Native Messaging 模式
// 持续从 stdin 读取消息，处理后写入 stdout
func RunNative(configPath string) error
```

#### 4. 安装命令 (`cmd/redisw/install.go`)

实现 `redisw install-native` 子命令。

```go
// InstallNative 注册 Native Messaging Host
// 在各浏览器的 Native Messaging Hosts 目录创建 manifest 文件
func InstallNative() error

// 支持的浏览器和 manifest 路径:
// Chrome (macOS): ~/Library/Application Support/Google/Chrome/NativeMessagingHosts/
// Chrome (Linux): ~/.config/google-chrome/NativeMessagingHosts/
// Firefox (macOS): ~/Library/Application Support/Mozilla/NativeMessagingHosts/
// Firefox (Linux): ~/.mozilla/native-messaging-hosts/
// Edge (macOS): ~/Library/Application Support/Microsoft Edge/NativeMessagingHosts/
// Edge (Linux): ~/.config/microsoft-edge/NativeMessagingHosts/
```

### 浏览器插件组件

#### 1. Background Script (`extension/background.js`)

管理与 Native Host 的通信。

```javascript
// NativeClient 封装 Native Messaging 通信
class NativeClient {
  constructor(hostName) {
    this.hostName = hostName;  // "com.redisw.native"
    this.port = null;
    this.pendingRequests = new Map();
    this.requestId = 0;
  }

  // 连接到 Native Host
  connect() { }

  // 断开连接
  disconnect() { }

  // 发送请求并等待响应
  async request(action, params) { }
}

// API 方法
async function listServers() { }
async function addServer(server) { }
async function updateServer(name, server) { }
async function deleteServer(name) { }
async function healthCheck() { }
async function executeCommand(serverName, command) { }
async function flushDb(serverName, confirm) { }
async function flushAll(serverName, confirm) { }
async function getHistory() { }
async function recordHistory(serverName) { }
async function importServers(data, format, conflictStrategy) { }
```

#### 2. Popup UI (`extension/popup/`)

主界面组件。

```
popup/
├── popup.html      # 主 HTML 结构
├── popup.css       # 样式
└── popup.js        # 交互逻辑
```

**UI 状态机**:
```
┌─────────────┐     选择服务器     ┌─────────────┐
│  服务器列表  │ ───────────────► │  命令执行    │
│   (默认)    │ ◄─────────────── │   界面      │
└─────────────┘     返回列表      └─────────────┘
      │                                 │
      │ 添加/编辑/导入                   │ FLUSH
      ▼                                 ▼
┌─────────────┐                  ┌─────────────┐
│  服务器表单  │                  │  确认对话框  │
│  / 导入界面 │                  │             │
└─────────────┘                  └─────────────┘
```

**批量导入界面设计**:

```
┌──────────────────────────────────────────────────┐
│  ← 返回列表          批量导入配置                 │
├──────────────────────────────────────────────────┤
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │                                            │  │
│  │     📁 点击选择文件或拖拽到此处             │  │
│  │                                            │  │
│  │     支持 YAML (.yml, .yaml) 和 JSON (.json) │  │
│  │                                            │  │
│  └────────────────────────────────────────────┘  │
│                                                  │
│  冲突处理策略:                                   │
│  ○ 跳过 - 保留现有配置                          │
│  ● 覆盖 - 用导入的配置替换                       │
│  ○ 重命名 - 自动添加后缀 (如 server_1)          │
│                                                  │
├──────────────────────────────────────────────────┤
│  预览 (3 个服务器):                              │
│  ┌────────────────────────────────────────────┐  │
│  │ ✓ production    192.168.1.100:6379         │  │
│  │ ⚠ staging       192.168.1.101:6379 (冲突)  │  │
│  │ ✓ development   localhost:6379             │  │
│  └────────────────────────────────────────────┘  │
│                                                  │
│  ┌────────────┐  ┌────────────────┐             │
│  │   取消     │  │  确认导入 (3)  │             │
│  └────────────┘  └────────────────┘             │
└──────────────────────────────────────────────────┘
```

**命令执行界面设计**:

```
┌──────────────────────────────────────────────────┐
│  ← 返回列表          localhost ✓                 │
│                      127.0.0.1:6379              │
├──────────────────────────────────────────────────┤
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │ 127.0.0.1:6379> GET mykey                  │  │
│  │ "hello world"                              │  │
│  │                                            │  │
│  │ 127.0.0.1:6379> KEYS *                     │  │
│  │ 1) "mykey"                                 │  │
│  │ 2) "counter"                               │  │
│  │ 3) "user:1"                                │  │
│  │                                            │  │
│  │ 127.0.0.1:6379> HGETALL user:1             │  │
│  │ 1) "name"                                  │  │
│  │ 2) "Alice"                                 │  │
│  │ 3) "age"                                   │  │
│  │ 4) "30"                                    │  │
│  │                                            │  │
│  │ 127.0.0.1:6379> _                          │  │
│  └────────────────────────────────────────────┘  │
│                                                  │
├──────────────────────────────────────────────────┤
│  ⚠️ 危险操作                                     │
│  ┌──────────────┐  ┌──────────────┐             │
│  │  FLUSHDB     │  │  FLUSHALL    │             │
│  │  清空当前库   │  │  清空所有库   │             │
│  └──────────────┘  └──────────────┘             │
└──────────────────────────────────────────────────┘
```

**界面元素说明**:

1. **顶部导航栏**
   - 返回按钮：返回服务器列表
   - 服务器名称和状态标记
   - 连接地址显示

2. **终端区域** (统一的终端风格)
   - 深色背景，等宽字体
   - 显示历史命令和结果
   - 底部显示当前输入行，带闪烁光标
   - 提示符格式：`host:port> `
   - 支持 Enter 键执行命令
   - 支持上下箭头浏览历史命令
   - 支持复制文本

3. **危险操作区域**
   - 红色警告样式
   - FLUSHDB 和 FLUSHALL 按钮
   - 点击后弹出二次确认对话框

**终端样式规范**:

```css
.terminal {
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.4;
  padding: 12px;
  border-radius: 4px;
  height: 300px;
  overflow-y: auto;
}

.terminal .prompt {
  color: #569cd6;  /* 蓝色提示符 */
}

.terminal .result-string {
  color: #ce9178;  /* 橙色字符串 */
}

.terminal .result-number {
  color: #b5cea8;  /* 绿色数字 */
}

.terminal .result-error {
  color: #f44747;  /* 红色错误 */
}

.terminal .cursor {
  animation: blink 1s infinite;
}
```

**确认对话框设计**:

```
┌──────────────────────────────────────┐
│  ⚠️ 危险操作确认                      │
├──────────────────────────────────────┤
│                                      │
│  你确定要执行 FLUSHDB 吗？            │
│                                      │
│  这将清空当前数据库的所有数据，        │
│  此操作不可撤销！                     │
│                                      │
│  服务器: localhost (127.0.0.1:6379)  │
│                                      │
│  ┌────────────┐  ┌────────────────┐  │
│  │   取消     │  │  确认执行 ⚠️   │  │
│  └────────────┘  └────────────────┘  │
└──────────────────────────────────────┘
```

#### 3. Manifest (`extension/manifest.json`)

WebExtension 配置文件。

```json
{
  "manifest_version": 3,
  "name": "Redisw",
  "version": "1.0.0",
  "description": "Redis 服务器连接管理工具",
  "permissions": ["nativeMessaging", "storage"],
  "background": {
    "service_worker": "background.js"
  },
  "action": {
    "default_popup": "popup/popup.html",
    "default_icon": {
      "16": "icons/icon16.png",
      "48": "icons/icon48.png",
      "128": "icons/icon128.png"
    }
  },
  "icons": {
    "16": "icons/icon16.png",
    "48": "icons/icon48.png",
    "128": "icons/icon128.png"
  }
}
```

## 数据模型

### 消息协议

#### 请求消息格式

```typescript
interface Request {
  id: string;       // 唯一请求 ID (UUID 或递增数字)
  action: string;   // 操作类型
  params?: object;  // 操作参数
}
```

#### 响应消息格式

```typescript
interface Response {
  id: string;       // 对应请求 ID
  success: boolean; // 操作是否成功
  data?: any;       // 成功时的数据
  error?: string;   // 失败时的错误信息
}
```

#### 操作类型定义

| Action | Params | Response Data |
|--------|--------|---------------|
| `list_servers` | 无 | `Server[]` |
| `add_server` | `Server` | `Server` |
| `update_server` | `{name: string, server: Server}` | `Server` |
| `delete_server` | `{name: string}` | `null` |
| `health_check` | 无 | `{[name: string]: boolean}` |
| `execute_command` | `{server: string, command: string}` | `{result: any}` |
| `flush_db` | `{server: string, confirm: boolean}` | `{keys_deleted: number}` |
| `flush_all` | `{server: string, confirm: boolean}` | `{keys_deleted: number}` |
| `get_history` | 无 | `string[]` |
| `record_history` | `{server: string}` | `null` |
| `import_servers` | `{data: string, format: "yaml"\|"json", conflict: "skip"\|"overwrite"\|"rename"}` | `{imported: number, skipped: number, failed: number, errors: string[]}` |

### 服务器配置

```typescript
interface Server {
  name: string;      // 服务器名称（唯一标识）
  host: string;      // 主机地址
  port: number;      // 端口号
  password?: string; // 认证密码（可选）
}
```

### Native Messaging Host Manifest

```json
{
  "name": "com.redisw.native",
  "description": "Redisw Native Messaging Host",
  "path": "/usr/local/bin/redisw",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://EXTENSION_ID/",
    "chrome-extension://FIREFOX_EXTENSION_ID/"
  ]
}
```

## 正确性属性

*正确性属性是一种应该在系统所有有效执行中保持为真的特征或行为——本质上是关于系统应该做什么的形式化陈述。属性作为人类可读规范和机器可验证正确性保证之间的桥梁。*

### Property 1: 消息编码往返一致性

*对于任意*有效的 Native Messaging 消息，使用 4 字节小端序长度前缀编码后再解码，应该产生与原始消息等价的结构。

**验证: 需求 1.3, 10.8**

### Property 2: JSON 消息解析和响应

*对于任意*有效的 JSON 请求消息，Native_Host 解析后应该返回有效的 JSON 响应。

**验证: 需求 1.1, 10.7**

### Property 3: 无效 JSON 错误处理

*对于任意*格式错误的 JSON 输入，Native_Host 应该返回包含 error 字段的 JSON 响应，而非崩溃或返回无效响应。

**验证: 需求 1.2**

### Property 4: Manifest 内容完整性

*对于任意*有效的安装配置（二进制路径、Extension ID），生成的 manifest 文件应该包含 name、description、path、type 和 allowed_origins 所有必需字段。

**验证: 需求 2.4**

### Property 5: 安装命令幂等性

*对于任意*初始状态（manifest 存在或不存在），执行 install-native 命令两次应该产生与执行一次相同的最终状态。

**验证: 需求 2.5**

### Property 6: 服务器配置往返一致性

*对于任意*有效的服务器配置，添加到配置后再查询列表，返回的列表应该包含该服务器且字段值相等。

**验证: 需求 3.1, 3.2, 3.7**

### Property 7: 服务器更新持久化

*对于任意*已存在的服务器和有效的更新数据，更新后再查询，返回的服务器配置应该反映更新后的值。

**验证: 需求 3.3**

### Property 8: 服务器删除持久化

*对于任意*已存在的服务器，删除后再查询列表，返回的列表不应该包含该服务器。

**验证: 需求 3.4**

### Property 9: 健康检查结果完整性

*对于任意*服务器配置列表，健康检查返回的状态映射应该包含列表中每个服务器的条目。

**验证: 需求 4.1**

### Property 10: 危险操作确认验证

*对于任意* flush_db 或 flush_all 请求，如果 confirm 字段不为 true，Native_Host 应该返回错误响应而非执行操作。

**验证: 需求 6.3, 6.4**

### Property 11: 历史记录往返一致性

*对于任意*服务器名称，记录到历史后再查询历史列表，返回的列表应该包含该服务器名称。

**验证: 需求 7.1, 7.2**

### Property 12: 历史记录容量限制

*对于任意*历史记录操作序列，历史列表的长度永远不应该超过 10。

**验证: 需求 7.3**

### Property 13: 历史记录去重和排序

*对于任意*服务器名称，如果重复记录该服务器，历史列表中应该只有一个该服务器的条目，且位于列表头部。

**验证: 需求 7.4**

### Property 14: 服务器列表渲染完整性

*对于任意*服务器配置和状态（健康状态、最近使用），渲染后的显示字符串应该包含服务器名称、健康状态标记和最近使用标记。

**验证: 需求 8.2**

### Property 15: 模糊搜索过滤

*对于任意*服务器列表和搜索词，过滤后的结果应该只包含名称中包含搜索词（忽略大小写）的服务器。

**验证: 需求 8.3**

### Property 16: 请求-响应 ID 匹配

*对于任意*有效请求，响应中的 id 字段应该与请求中的 id 字段相等。

**验证: 需求 10.2, 10.4**

### Property 17: 响应格式一致性

*对于任意*请求，响应应该包含 success 布尔字段；如果 success 为 false，应该包含 error 字段；如果 success 为 true，应该包含 data 字段。

**验证: 需求 10.3, 10.5, 10.6**

### Property 18: 导入格式解析

*对于任意*有效的 YAML 或 JSON 格式服务器配置，导入后应该能够正确解析并添加到配置列表。

**验证: 需求 11.1, 11.2, 11.3**

### Property 19: 导入冲突处理

*对于任意*导入操作，如果存在名称冲突，系统应该根据指定的策略（跳过/覆盖/重命名）正确处理，且最终配置列表中不应该存在重复名称。

**验证: 需求 11.4**

### Property 20: 导入结果统计

*对于任意*导入操作，返回的统计数据（imported + skipped + failed）应该等于导入数据中的服务器总数。

**验证: 需求 11.5**

## 错误处理

### Go 端错误处理

| 错误场景 | 处理方式 |
|---------|---------|
| 无效 JSON 输入 | 返回 `{success: false, error: "invalid JSON: ..."}` |
| 未知 action | 返回 `{success: false, error: "unknown action: ..."}` |
| 服务器名称已存在 | 返回 `{success: false, error: "server already exists: ..."}` |
| 服务器不存在 | 返回 `{success: false, error: "server not found: ..."}` |
| 配置文件读写失败 | 返回 `{success: false, error: "config error: ..."}` |
| Redis 连接失败 | 返回 `{success: false, error: "connection failed: ..."}` |
| Redis 命令执行失败 | 返回 `{success: false, error: "redis error: ..."}` |
| 危险操作未确认 | 返回 `{success: false, error: "confirmation required"}` |

### 浏览器插件错误处理

| 错误场景 | 处理方式 |
|---------|---------|
| Native Host 未安装 | 显示安装指引对话框 |
| Native Host 连接失败 | 显示错误提示，提供重试按钮 |
| 请求超时 | 显示超时错误，提供重试按钮 |
| 服务器操作失败 | 显示错误消息，保持当前状态 |

## 测试策略

### 双重测试方法

本项目采用单元测试和属性测试相结合的方式：

- **单元测试**：验证具体示例、边界情况和错误条件
- **属性测试**：验证所有输入上的通用属性

### Go 端测试

**属性测试库**: `github.com/leanovate/gopter`

**测试配置**:
- 每个属性测试最少运行 100 次迭代
- 每个测试用注释标记对应的设计属性
- 标记格式: `// Feature: browser-extension, Property N: 属性描述`

**测试文件结构**:
```
internal/native/
├── protocol_test.go      # 协议编解码测试
├── handler_test.go       # 消息处理测试
└── install_test.go       # 安装命令测试
```

### 浏览器插件测试

**测试框架**: Jest + @testing-library

**测试文件结构**:
```
extension/
├── __tests__/
│   ├── background.test.js   # Background script 测试
│   ├── popup.test.js        # Popup UI 测试
│   └── filter.test.js       # 搜索过滤测试
```

### 集成测试

使用实际的 Redis 服务器进行端到端测试：

1. 启动测试用 Redis 容器
2. 运行 Native Host 进程
3. 模拟浏览器插件发送请求
4. 验证 Redis 状态变化

### 测试覆盖目标

| 模块 | 目标覆盖率 |
|------|-----------|
| internal/native/protocol | >90% |
| internal/native/handler | >80% |
| internal/native/install | >70% |
| extension/background | >80% |
| extension/popup | >60% |

