# Redisw

Redisw 是一个 **极简主义** 设计的 Redis 服务器连接切换工具。快速、优雅地在多个 Redis 服务器之间切换，提升开发和运维效率。

## ✨ 功能特点

### v1.2.0 新特性

- **📊 历史记录智能排序**：最近使用的服务器自动排到最前面（标记 ★）
- **⚡ 并发健康检查**：启动时并发检查所有服务器，速度提升 3 倍
- **🔍 连接状态可视化**：一眼看出服务器可达性（✓ 可达 / ✗ 不可达）
- **🖥️ Windows 支持**：新增 Windows 平台支持

### 基础功能

- 支持多个 Redis 服务器配置
- 交互式服务器选择界面
- 支持模糊搜索（服务器名称过滤）
- 支持密码保护的 Redis 连接
- 支持 Redis 集群模式
- 支持自定义配置文件路径
- 命令行界面简洁直观

## 📦 安装

### 系统要求

- **Redis CLI 工具**（必需）
- 支持平台：macOS、Linux、Windows

### 安装 Redis CLI

**macOS:**
```bash
brew install redis
```

**Linux (Ubuntu/Debian):**
```bash
apt-get install redis-tools
```

**Windows:**
从 [Redis 官网](https://redis.io/download) 下载安装包。

### 方式 1: Homebrew (macOS 推荐)

```bash
brew tap zhaojy0527/redisw
brew install redisw
```

### 方式 2: 二进制包安装

1. 访问 [Releases](https://github.com/zhaojy0527/redisw/releases) 页面
2. 下载对应平台的压缩包：
   - macOS (arm64): `redisw_1.2.0_Darwin_arm64.tar.gz`
   - macOS (amd64): `redisw_1.2.0_Darwin_x86_64.tar.gz`
   - Linux (arm64): `redisw_1.2.0_Linux_arm64.tar.gz`
   - Linux (amd64): `redisw_1.2.0_Linux_x86_64.tar.gz`
   - Windows (amd64): `redisw_1.2.0_Windows_x86_64.zip`

**macOS/Linux:**
```bash
tar -xzf redisw_*.tar.gz
chmod +x redisw
sudo mv redisw /usr/local/bin/
```

**Windows:**
解压 zip 文件，将 `redisw.exe` 添加到系统 PATH。

### 方式 3: 从源码编译

```bash
git clone https://github.com/zhaojy0527/redisw.git
cd redisw
make build
sudo make install  # 可选：安装到系统
```

## ⚙️ 配置

### 配置文件优先级

Redisw 按以下优先级查找配置文件：

1. `~/redisw_config.yml` (最高优先级)
2. `~/redisw_config.yaml`
3. `~/.config/redisw/redisw_config.yml`
4. `~/.config/redisw/redisw_config.yaml`
5. `./redisw_config.yml` (最低优先级)

如果都不存在，会自动创建默认配置到 `~/.config/redisw/redisw_config.yml`

### 配置文件示例

```yaml
- name: "本地 Redis"
  host: "127.0.0.1"
  port: 6379
  password: ""

- name: "开发环境"
  host: "dev.redis.example.com"
  port: 6379
  password: "your-password"

- name: "生产环境 Redis"
  host: "prod.redis.example.com"
  port: 6379
  password: "prod-password"

- name: "测试集群"
  host: "test.redis.cluster"
  port: 6379
  password: "cluster-password"
```

## 🚀 使用方法

### 基本使用

```bash
# 使用默认配置文件启动
redisw

# 指定配置文件启动
redisw -config /path/to/redisw_config.yml
```

### 交互界面说明

启动后会显示服务器列表，包含状态标记：

```
✨ Select Redis Server
➤ 本地 Redis ★ ✓              # ★ = 最近使用, ✓ = 可连接
  开发环境 ★ ✓                # 自动排序到前面
  生产环境 Redis ✓            # 可连接但未最近使用
  测试集群 ✗                  # 不可达
```

**操作快捷键：**
- `↑↓` 或 `j/k`：选择服务器
- 输入关键字：模糊搜索过滤
- `Enter`：连接选中的服务器
- `Ctrl+C`：返回选择界面（连接后）/ 退出程序

### 新功能演示

**历史记录：**
- 自动记录最近连接的 10 个服务器
- 下次启动时自动排到最前面（标记 ★）
- 历史记录保存在 `~/.config/redisw/history.json`

**健康检查：**
- 启动时自动并发检查所有服务器可达性
- 可达的服务器标记 ✓（绿色）
- 不可达的服务器标记 ✗（红色）
- 超时时间：2 秒（快速响应）

## 🛠️ 高级用法

### 团队共享配置

```bash
# 将配置文件加入项目仓库
git add redisw_config.yml
git commit -m "Add team Redis config"

# 团队成员使用
redisw -config ./redisw_config.yml
```

### 重置历史记录

```bash
rm ~/.config/redisw/history.json
```

### 查看配置文件位置

```bash
# 启动 redisw，配置文件路径会在日志中显示
redisw
```

## 📊 性能对比

| 场景 | v1.1.0 | v1.2.0 | 提升 |
|------|--------|--------|------|
| 启动时间（10 个服务器） | ~20 秒 | ~2 秒 | **10 倍** |
| 常用服务器访问 | 3-5 秒 | 0.5 秒 | **6-10 倍** |
| 健康检查方式 | 串行 | 并发 | **3 倍加速** |

## 🤝 贡献指南

欢迎贡献代码、报告 Bug 或提出新功能建议！

1. Fork 本仓库
2. 创建特性分支：`git checkout -b feature/AmazingFeature`
3. 提交更改：`git commit -m 'feat: add some amazing feature'`
4. 推送到分支：`git push origin feature/AmazingFeature`
5. 提交 Pull Request

## 📝 常见问题

**Q: 如何添加新服务器？**
A: 编辑配置文件 `~/.config/redisw/redisw_config.yml`，添加新条目后重启程序。

**Q: 提示 redis-cli 未找到？**
A: 请先安装 Redis CLI 工具（见安装要求）。

**Q: Windows 上如何使用？**
A: 下载 Windows 版本的 zip 包，解压后将 `redisw.exe` 添加到系统 PATH。

**Q: 如何查看历史记录？**
A: 历史记录保存在 `~/.config/redisw/history.json`，可以直接查看或删除。

**Q: 支持 SSH 隧道吗？**
A: 当前版本暂不支持，计划在 v1.3.0 中添加。

## �� 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## ❤️ 支持项目

如果这个项目对你有帮助，请考虑：

- ⭐ **Star**：帮助更多开发者发现这个项目
- 🍴 **Fork**：参与项目开发，提交改进建议
- 👀 **Watch**：及时获取项目更新动态

你的支持是我们持续改进的动力！

## 📮 联系方式

- **问题反馈**：[GitHub Issues](https://github.com/zhaojy0527/redisw/issues)
- **邮件**：zhaojianyong0527@gmail.com
- **变更日志**：[CHANGELOG.md](CHANGELOG.md)
- **架构文档**：[CLAUDE.md](CLAUDE.md)

---

**Made with ❤️ by the Redisw Team**