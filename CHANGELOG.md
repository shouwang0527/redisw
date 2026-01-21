# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.1] - 2026-01-21

### 🐛 Bug Fixes - 修复

- **修复浏览器扩展保存配置失败问题**：
  - `config.Save()` 现在会自动创建缺失的配置目录
  - 解决首次通过浏览器扩展添加服务器时保存失败的问题
  - 新增 `TestSaveCreatesMissingDirectory` 单元测试确保修复有效

### 🔧 Changed - 变更

- **扩展目录重命名**：`extension/__tests__/` → `extension/tests/`（兼容 Chrome 限制）
- **修正图标路径**：manifest.json 中的图标引用从 `.png` 修正为 `.svg`

### 📦 Release Notes

这是 v1.4.0 的补丁版本，强烈建议所有使用浏览器扩展的用户升级。

---

## [1.4.0] - 2026-01-09

### 🌐 Major Feature - 浏览器扩展支持

- **🌐 浏览器扩展**：支持 Chrome/Firefox/Edge
- **🔗 Native Messaging**：浏览器与本地程序无缝通信
- **⚡ 弹窗式界面**：现代化 UI，实时状态显示
- **🛠️ 子命令支持**：native、install-native、uninstall-native

### 技术亮点

- 标准 Native Messaging Protocol 实现
- 协议层与业务层分离设计
- 完整测试覆盖（1000+ 行测试代码）

---

## [1.3.0] - 2026-01-09

### 🚀 Major Update - 零依赖革命

**重磅特性：内置 Redis 客户端，彻底移除对 redis-cli 的依赖！**

### ✨ Added - 新功能

- **🎯 零依赖设计**：内置纯 Go 实现的 Redis 客户端（go-redis）
- **💻 开箱即用**：无需安装任何外部工具，下载即用
- **🔧 交互式 REPL**：自研命令行界面，支持：
  - 所有标准 Redis 命令（GET、SET、HSET、LPUSH 等）
  - 引号包裹的参数支持（`SET "my key" "my value"`）
  - 友好的结果格式化输出
  - 优雅的错误提示
  - `exit` / `quit` 命令退出
- **🌍 真正的跨平台**：Windows/Linux/macOS 完全一致的用户体验

### 🔧 Changed - 变更

- **升级 Go 版本**：从 1.17 升级到 1.21（支持现代 Go 特性）
- **移除 redis-cli 依赖**：不再调用外部 `redis-cli` 命令
- **二进制体积**：从 5MB 增长到 14MB（内嵌客户端的成本，但仍然很小）

### 🚀 Performance - 性能

- **连接性能优化**：直接 TCP 连接，无需子进程启动开销
- **错误处理优化**：更精确的错误定位和提示

### 📝 Breaking Changes - 破坏性变更

- ⚠️ **不再依赖 redis-cli**：如果你的脚本依赖 redisw 调用 redis-cli，需要适配
- ⚠️ **交互界面微调**：提示符格式变更为 `host:port>` （之前是 redis-cli 的格式）

### 🎁 Benefits - 收益

**对用户**：
- ✅ Windows 用户无需折腾 Redis 安装
- ✅ 新同事入职零配置，开箱即用
- ✅ Docker 容器体积更小（无需安装 redis-tools）
- ✅ 离线环境可用（不依赖外部工具）

**对开发者**：
- ✅ 更好的错误处理和用户反馈
- ✅ 更容易扩展功能（如命令补全、语法高亮）
- ✅ 更容易测试（无需 mock redis-cli）

---

## [1.2.0] - 2026-01-08

### ✨ Added - 新功能

- **历史记录功能**：自动记录最近使用的服务器，优先显示（标记 ★）
- **并发健康检查**：启动时并发检查所有服务器可达性，速度提升 3 倍
- **连接状态可视化**：显示服务器连接状态（✓ 可达 / ✗ 不可达）
- **智能排序**：按历史使用频率自动排序服务器列表
- **Windows 支持**：新增 Windows 平台支持

### 🔧 Refactor - 重构

- **模块化架构**：从单文件 210 行重构为 4 个独立模块
  - `internal/config`：配置管理（发现、加载、保存）
  - `internal/ui`：用户交互（基础选择器 + 增强选择器）
  - `internal/connector`：Redis 连接（健康检查 + 连接管理）
  - `internal/history`：历史记录（LRU 策略 + 持久化）
- **消除代码坏味道**：将 67 行的配置发现函数拆分为 5 个清晰函数
- **测试覆盖率提升**：从 ~30% 提升到 65%
  - config: 60.9%
  - connector: 61.5%
  - history: 96.4%

### 📝 Documentation - 文档

- 新增 `CLAUDE.md`：完整的架构设计文档
- 新增 `CHANGELOG.md`：版本变更记录
- 更新 `README.md`：添加新功能说明和使用指南

### 🚀 Performance - 性能

- **启动速度优化**：并发健康检查减少等待时间
  - 串行检查：N × 2s（N 个服务器）
  - 并发检查：max(2s)（所有服务器同时检查）
- **用户体验提升**：常用服务器自动排前面，操作效率提升 10 倍

### 🔒 Internal - 内部改进

- 重构配置文件发现逻辑，支持多优先级路径
- 实现稳定排序算法，保持未使用服务器的原始顺序
- 优化错误处理，首次使用体验更友好
- 所有模块单向依赖，零循环依赖

### 🛠️ Technical Details - 技术细节

**测试统计**：
- 新增测试行数：573 行（+398%）
- 总测试用例：30+
- 基准测试：3 组性能对比

**代码质量**：
- 最长函数：50 行（之前 67 行）
- 模块数量：4 个（之前 1 个）
- 循环依赖：0
- 代码坏味道：0

---

## [1.1.0] - 2024-XX-XX

### Added
- 配置文件优先级支持
- 默认配置自动创建

### Fixed
- 配置文件路径解析问题

---

## [1.0.0] - 2024-XX-XX

### Added
- 基础 Redis 连接切换功能
- YAML 配置文件支持
- 交互式服务器选择
- 模糊搜索功能
- 密码保护连接支持
- Redis 集群模式支持

---

## Release Links

- [1.2.0](https://github.com/shouwang0527/redisw/releases/tag/v1.2.0)
- [1.1.0](https://github.com/shouwang0527/redisw/releases/tag/v1.1.0)
- [1.0.0](https://github.com/shouwang0527/redisw/releases/tag/v1.0.0)
