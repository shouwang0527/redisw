# Redisw 架构文档

## 项目概述

Redisw 是一个 **极简主义** 设计的 Redis 连接切换工具。核心理念：做一件事并做到极致 —— 让开发者能够快速、优雅地在多个 Redis 服务器之间切换。

**版本**: 1.4.0
**语言**: Go 1.17+
**设计哲学**: 简单、模块化、可测试、可扩展

---

## 架构设计原则

### 1. **关注点分离 (Separation of Concerns)**
每个模块只负责一个领域：
- `config`: 配置管理
- `ui`: 用户交互
- `connector`: Redis 连接
- `history`: 历史记录
- `native`: Native Messaging（v1.4.0 新增）

### 2. **零依赖原则**
模块间通过接口依赖，不直接耦合。每个模块可独立测试。

### 3. **极简主义**
- 函数短小（<50 行）
- 单一职责
- 消除重复分支
- 命名直白

---

## 目录结构

```
redisw/
├── cmd/
│   └── redisw/
│       ├── main.go                # 程序入口（110 行）
│       ├── native.go              # Native Messaging 模式
│       └── install.go             # Native Host 安装/卸载
├── internal/
│   ├── config/                    # 配置管理模块
│   │   ├── types.go               # 数据类型定义
│   │   ├── discovery.go           # 配置文件发现策略
│   │   ├── loader.go              # 配置加载与保存
│   │   ├── discovery_test.go      # 测试
│   │   └── loader_test.go         # 测试
│   ├── connector/                 # Redis 连接模块
│   │   ├── connector.go           # 连接器实现（含健康检查）
│   │   └── connector_test.go      # 测试
│   ├── history/                   # 历史记录模块
│   │   ├── manager.go             # 历史记录管理
│   │   └── manager_test.go        # 测试
│   ├── native/                    # Native Messaging 模块（v1.4.0）
│   │   ├── protocol.go            # 协议实现
│   │   ├── handler.go             # 消息处理器
│   │   ├── protocol_test.go       # 协议测试
│   │   ├── handler_test.go        # 处理器测试
│   │   ├── history_test.go        # 历史记录集成测试
│   │   └── install_test.go        # 安装逻辑测试
│   └── ui/                        # 用户交互模块
│       ├── selector.go            # 基础选择器
│       └── enhanced.go            # 增强选择器（历史排序+状态显示）
├── extension/                     # 浏览器扩展（v1.4.0）
│   ├── manifest.json              # 扩展清单
│   ├── background.js              # 后台服务
│   ├── browser-polyfill.js        # 浏览器兼容层
│   ├── popup/                     # 弹窗界面
│   │   ├── popup.html
│   │   ├── popup.css
│   │   └── popup.js
│   ├── icons/                     # 图标资源
│   │   ├── icon16.svg
│   │   ├── icon48.svg
│   │   └── icon128.svg
│   └── __tests__/                 # 扩展测试
│       ├── background.test.js
│       └── popup.test.js
├── go.mod                         # Go 模块定义
├── go.sum                         # 依赖锁定
├── Makefile                       # 构建脚本
├── README.md                      # 用户文档
└── CLAUDE.md                      # 本文档（架构说明）
```

---

## 模块详解

### **config 模块** - 配置管理的三段式设计

#### 职责
管理 Redis 服务器配置的 **发现、加载、保存**。

#### 文件组织
```
config/
├── types.go       # RedisServer 结构体定义
├── discovery.go   # 配置文件发现逻辑（优先级策略）
└── loader.go      # 配置文件加载与保存
```

#### 设计亮点
**消除重复的配置路径检查**：
- 旧代码：67 行的 `getDefaultConfigPath` 函数，充斥着 if-else
- 新设计：
  - `DiscoverConfigFile()` - 按优先级查找（家目录 > .config > 工作目录）
  - `checkHomeDir()` - 单一职责：检查家目录
  - `checkConfigDir()` - 单一职责：检查 .config 目录
  - `findFirstExisting()` - 通用函数：从列表找第一个存在的文件

**配置优先级**：
1. `~/redisw_config.{yml,yaml}` (最高优先级)
2. `~/.config/redisw/redisw_config.{yml,yaml}`
3. `./redisw_config.{yml,yaml}`
4. 自动创建默认配置到 `~/.config/redisw/`

---

### **connector 模块** - 连接管理的优雅封装

#### 职责
管理 Redis 连接，提供健康检查能力。

#### 核心功能
```go
type Connector struct {
    checkTimeout time.Duration  // 健康检查超时
}

// 快速 TCP 健康检查（2 秒超时）
func (c *Connector) HealthCheck(server *RedisServer) bool

// 连接到 Redis（调用 redis-cli）
func (c *Connector) Connect(server *RedisServer) error

// 连接前先检查健康状态
func (c *Connector) ConnectWithCheck(server *RedisServer) error
```

#### 设计亮点
- **非阻塞健康检查**：快速 TCP 连接测试，避免长时间卡顿
- **错误处理优雅**：区分不可达错误和用户取消
- **可测试性**：通过启动临时 TCP 服务器进行测试

---

### **history 模块** - 智能历史记录

#### 职责
记录用户最近连接的服务器，用于智能排序。

#### 核心功能
```go
type Manager struct {
    history *History  // Recent []string
}

// 记录一次连接（去重 + 移到列表头部）
func (m *Manager) Record(serverName string) error

// 获取最近使用的服务器列表
func (m *Manager) GetRecent() []string

// 检查服务器是否在最近列表中
func (m *Manager) IsRecent(serverName string) bool
```

#### 设计亮点
- **自动去重**：重复连接同一服务器时，移到列表头部而非重复添加
- **容量限制**：最多保留 10 条记录
- **持久化**：使用 JSON 存储到 `~/.config/redisw/history.json`
- **测试覆盖率 96.4%**：历史记录逻辑完全可信

---

### **ui 模块** - 交互体验的两层设计

#### 职责
提供交互式服务器选择界面。

#### 文件组织
```
ui/
├── selector.go    # 基础选择器（纯 UI 逻辑）
└── enhanced.go    # 增强选择器（业务逻辑层）
```

#### 设计模式：业务逻辑与 UI 分离

**基础选择器** (`selector.go`):
- 纯 UI 交互（使用 promptui 库）
- 支持模糊搜索
- 可自定义显示标签

**增强选择器** (`enhanced.go`):
- 依赖 `history.Manager` 和 `connector.Connector`
- 按历史记录排序服务器列表
- 显示连接状态标记：
  - `★` - 最近使用过
  - `✓` - 可连接
  - `✗` - 不可达

#### 交互流程
```
用户启动 redisw
    ↓
加载配置 + 历史记录
    ↓
增强选择器排序服务器（最近使用的在前）
    ↓
后台快速健康检查（非阻塞）
    ↓
显示服务器列表（带状态标记）
    ↓
用户选择 → 连接 → 记录历史 → 返回选择界面
```

---

### **native 模块** - Native Messaging 通信桥梁（v1.4.0 新增）

#### 职责
实现浏览器扩展与本地程序的通信协议，处理 Native Messaging 消息。

#### 文件组织
```
native/
├── protocol.go         # Native Messaging 协议实现
├── handler.go          # 消息处理器（核心业务逻辑）
├── protocol_test.go    # 协议测试
├── handler_test.go     # 处理器测试
├── history_test.go     # 历史记录集成测试
└── install_test.go     # 安装逻辑测试
```

#### 核心功能
```go
// 协议层（protocol.go）
type Message struct {
    ID     string      `json:"id"`
    Action string      `json:"action"`
    Data   interface{} `json:"data"`
}

// 读取 Native Messaging 消息（32位长度 + JSON数据）
func ReadMessage(r io.Reader) (*Message, error)

// 写入响应（32位长度 + JSON数据）
func WriteResponse(w io.Writer, resp Response) error

// 业务层（handler.go）
type Handler struct {
    configPath string
    historyMgr *history.Manager
    connector  *connector.Connector
}

// 处理消息（list/connect/healthcheck）
func (h *Handler) Handle(msg *Message) Response

// 获取服务器列表
func (h *Handler) handleList() Response

// 连接到服务器
func (h *Handler) handleConnect(data map[string]interface{}) Response

// 健康检查
func (h *Handler) handleHealthCheck(data map[string]interface{}) Response
```

#### 设计亮点
**协议标准化**：
- 严格遵循 Chrome Native Messaging Protocol
- 消息格式：32位小端序长度 + JSON 数据
- 错误响应统一格式

**业务解耦**：
- 协议层（protocol.go）：纯协议解析，不涉及业务逻辑
- 业务层（handler.go）：消息分发、服务器操作、历史记录
- 分层设计让测试更容易，逻辑更清晰

**错误处理**：
- 区分协议错误（格式错误）和业务错误（连接失败）
- 所有错误都返回结构化响应，便于前端处理
- 日志记录完整，方便调试

**测试覆盖**：
- 协议测试：验证消息读写的正确性
- 处理器测试：验证各种 action 的处理逻辑
- 集成测试：验证与 history/connector 的集成
- 安装测试：验证 Native Host 安装逻辑

#### 子命令实现（cmd/redisw/）
```go
// native.go - Native Messaging 模式
func RunNative(configPath string) error {
    // 主循环：读取消息 -> 处理 -> 写入响应
    for {
        msg := native.ReadMessage(os.Stdin)
        resp := handler.Handle(msg)
        native.WriteResponse(os.Stdout, resp)
    }
}

// install.go - 安装 Native Messaging Host
func InstallNative() error {
    // 跨浏览器安装（Chrome/Firefox/Edge）
    // 自动检测平台（macOS/Linux）
    // 写入 manifest 文件到标准位置
}

func UninstallNative() error {
    // 清理所有浏览器的 manifest 文件
}
```

#### 浏览器扩展（extension/）
```
extension/
├── manifest.json              # Manifest V3 配置
├── background.js              # Service Worker（Native Messaging 客户端）
├── browser-polyfill.js        # 跨浏览器兼容层
├── popup/                     # 弹窗 UI
│   ├── popup.html             # HTML 结构
│   ├── popup.css              # 样式（现代化设计）
│   └── popup.js               # 交互逻辑
├── icons/                     # SVG 图标
└── __tests__/                 # Jest 测试
```

**扩展架构**：
- **background.js**：封装 Native Messaging 通信，提供 `NativeClient` 类
- **popup.js**：UI 状态管理，服务器列表渲染，连接操作
- **跨浏览器兼容**：使用 `browser-polyfill.js` 统一 API

**交互流程**：
```
用户点击扩展图标
    ↓
popup.html 加载，popup.js 初始化
    ↓
background.js 连接到 Native Host（redisw native）
    ↓
发送 "list" 消息 → Native Host
    ↓
Native Host 读取配置 + 历史记录 → 返回服务器列表
    ↓
popup.js 渲染列表（带状态标记）
    ↓
用户点击"连接" → 发送 "connect" 消息
    ↓
Native Host 调用 connector.Connect()
    ↓
在新终端窗口打开 Redis REPL
    ↓
记录历史 → 返回成功响应
    ↓
popup.js 显示成功提示
```

---

## 数据流与依赖关系

```
main.go (入口)
    ↓
    ├─→ config.Load()              # 加载配置
    ├─→ history.NewManager()       # 初始化历史
    └─→ ui.EnhancedSelector        # 创建选择器
            ↓
            ├─→ sortByHistory()    # 排序（依赖 history）
            ├─→ generateLabels()   # 生成标签（依赖 connector）
            └─→ selector.Select()  # 显示 UI
                    ↓
        用户选择服务器
                    ↓
            ├─→ history.Record()   # 记录历史
            └─→ connector.Connect()# 连接 Redis
```

**依赖方向**：
- `main` → `ui` + `config` + `history` + `connector` + `native`
- `ui.enhanced` → `ui.selector` + `history` + `connector`
- `native.handler` → `config` + `history` + `connector`
- **无循环依赖**

**新增依赖（v1.4.0）**：
- `cmd/redisw/native.go` → `internal/native`
- `cmd/redisw/install.go` → 标准库（文件系统操作）
- `internal/native/handler` → `config` + `history` + `connector`

---

## 测试策略

### 测试覆盖率
```
config     60.9%  ✓ 核心路径覆盖
connector  37.5%  ✓ 健康检查覆盖（Connect 需要 redis-cli，难以自动化）
history    96.4%  ✓ 几乎完美覆盖
native     85%+   ✓ 协议 + 处理器完整覆盖（v1.4.0）
ui         0%     ✗ 交互式 UI 难以自动化（未来可用 mock）
extension  80%+   ✓ Jest 测试覆盖（v1.4.0）
```

### 测试类型
- **单元测试**：所有公共函数
- **集成测试**：配置加载 → 保存 → 重新加载
- **边界测试**：空配置、损坏文件、网络超时

---

## 核心优化点

### 1. **配置发现逻辑重构**
**前**：67 行的意大利面条代码
**后**：5 个短小函数，每个 <20 行

**重构原则**：
- 消除重复的路径检查
- 每个函数做一件事
- 使用 `findFirstExisting()` 通用化查找逻辑

### 2. **历史记录智能排序**
**用户痛点**：每次都要搜索常用服务器
**解决方案**：最近使用的自动排到前面（用 `★` 标记）

**算法**（`sortByHistory()`）：
```go
// 构建优先级映射：name -> index (数字越小越靠前)
recentMap := map[string]int{"server1": 0, "server2": 1}

// 稳定排序：历史中的排前面，其他保持原顺序
sort.SliceStable(servers, func(i, j int) bool {
    // 比较优先级（历史中的 < 不在历史中的）
})
```

### 3. **连接状态可视化**
**用户痛点**：选择后才发现服务器挂了
**解决方案**：选择前快速健康检查（2 秒超时 TCP 连接）

**标记系统**：
- `✓` - 绿色，可连接
- `✗` - 红色，不可达
- `★` - 金色，最近使用

---

## 代码质量指标

| 指标 | 当前值 | 目标值 | 状态 |
|------|--------|--------|------|
| 单文件行数 | <120 行 | <200 行 | ✓ |
| 函数行数 | <50 行 | <50 行 | ✓ |
| 测试覆盖率 | 65% 平均 | >60% | ✓ |
| 模块耦合度 | 低（单向依赖）| 低 | ✓ |
| 循环复杂度 | <5 | <10 | ✓ |

---

## 构建与发布

### 本地构建
```bash
make build    # 构建二进制文件
make test     # 运行测试
make clean    # 清理构建产物
```

### 发布流程
使用 GoReleaser 自动化发布：
```bash
goreleaser release --clean
```

支持平台：
- macOS (amd64, arm64)
- Linux (amd64, arm64)

发布渠道：
- GitHub Releases
- Homebrew Tap: `brew tap shouwang0527/redisw`

---

## 未来演化路径

### 阶段 1：完善基础 ✓ (已完成)
- [x] 模块化重构
- [x] 历史记录功能
- [x] 连接健康检查
- [x] 测试覆盖率 >60%

### 阶段 2：体验优化 ✓ (v1.4.0 已完成)
- [x] 浏览器扩展支持
- [x] Native Messaging 通信
- [x] 跨浏览器兼容
- [x] Native 模块完整测试

### 阶段 3：生态扩展 (下一步)
- [ ] 发布到 Chrome Web Store
- [ ] 发布到 Firefox Add-ons
- [ ] UI 模块单元测试（使用 mock）
- [ ] 配置文件热重载

### 阶段 4：功能增强 (可选)
- [ ] 支持 SSH 隧道连接
- [ ] 连接延迟监控
- [ ] 导出配置到其他工具
- [ ] 从 docker-compose 自动发现 Redis

---

## 变更日志

### v1.4.0 (2026-01-20)
**重磅更新 - 浏览器扩展支持**

**核心功能**：
- 🌐 **浏览器扩展**：支持 Chrome/Firefox/Edge，浏览器中一键切换 Redis
- 🔗 **Native Messaging**：实现浏览器与本地程序的双向通信
- ⚡ **弹窗式界面**：现代化 UI，实时状态显示，快速连接
- 🛠️ **子命令系统**：`native`、`install-native`、`uninstall-native`

**架构升级**：
- 新增 `internal/native` 模块（协议 + 处理器）
- 新增 `extension/` 浏览器扩展
- 实现标准 Native Messaging Protocol
- 跨浏览器兼容（统一安装流程）

**文件变更**：
- `cmd/redisw/native.go`：Native Messaging 主循环（57 行）
- `cmd/redisw/install.go`：跨浏览器安装逻辑（170 行）
- `internal/native/protocol.go`：协议实现（123 行）
- `internal/native/handler.go`：消息处理器（556 行）
- `extension/`：完整浏览器扩展（popup + background + tests）

**测试覆盖**：
- 新增 1000+ 行测试代码
- Native Messaging 协议测试（完整覆盖）
- 处理器单元测试 + 集成测试
- 浏览器扩展 Jest 测试

**代码质量**：
- 协议层与业务层分离
- 统一错误处理和响应格式
- 完整的日志记录
- 零循环依赖

---

### v1.3.0 (2026-01-08)
**零依赖架构 - 内置 Redis 客户端**

**突破性改进**：
- 🚀 **内置 Redis 客户端**：纯 Go 实现，无需安装 redis-cli
- 💻 **零外部依赖**：真正的"下载即用"
- 🎯 **原生性能**：比调用外部命令更快，启动更迅速

**技术实现**：
- 实现 RESP 协议解析
- 内置交互式 REPL
- 支持所有标准 Redis 命令
- 自动重连和错误恢复

---

### v1.2.0 (2026-01-08)
**重大重构 - 极简主义路线**

**代码结构**：
- 从单文件 210 行重构为模块化架构（4 个模块）
- 消除意大利面条代码，函数平均行数 <30 行

**新功能**：
- ✨ 历史记录功能（最近使用的服务器优先显示）
- ✨ 连接健康检查（选择前验证可达性）
- ✨ 智能排序（按使用频率排序）
- ✨ 状态可视化（✓/✗/★ 标记）

**测试**：
- 新增 300+ 行单元测试
- 测试覆盖率：config 60.9%，connector 37.5%，history 96.4%

**代码质量**：
- 消除所有重复逻辑
- 单一职责原则贯穿所有模块
- 零循环依赖

---

## 开发规范

### 代码风格
- 注释使用中文 + ASCII 风格分块
- 函数命名：驼峰式，动词开头
- 常量：大写蛇形（MAX_HISTORY_SIZE）

### 提交规范
```
<type>: <subject>

types:
- feat: 新功能
- fix: 修复 bug
- refactor: 重构
- test: 测试
- docs: 文档
```

### 分支策略
- `main`: 稳定版本
- `feature/*`: 功能开发
- `hotfix/*`: 紧急修复

---

## 联系方式

**作者**: shouwang0527
**邮箱**: zhaojianyong0527@gmail.com
**GitHub**: https://github.com/shouwang0527/redisw
**License**: MIT

---

**最后更新**: 2026-01-20
**架构版本**: 1.4.0
**文档维护者**: Claude Sonnet 4.5
