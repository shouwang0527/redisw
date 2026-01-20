package native

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v2"
	"redisw/internal/config"
	"redisw/internal/connector"
	"redisw/internal/history"
)

// ========================================
// Native Messaging Handler
// 处理来自浏览器插件的请求
// ========================================

// Handler 处理 Native Messaging 请求
type Handler struct {
	configPath string
	connector  *connector.Connector
	historyMgr *history.Manager
}

// NewHandler 创建处理器
func NewHandler(configPath string, configDir string) (*Handler, error) {
	conn := connector.NewConnector()

	historyMgr, err := history.NewManager(configDir)
	if err != nil {
		// 历史记录加载失败不是致命错误
		historyMgr = nil
	}

	return &Handler{
		configPath: configPath,
		connector:  conn,
		historyMgr: historyMgr,
	}, nil
}

// Handle 处理单个请求并返回响应
func (h *Handler) Handle(msg *Message) *Response {
	switch msg.Action {
	case "list_servers":
		return h.handleListServers(msg)
	case "add_server":
		return h.handleAddServer(msg)
	case "update_server":
		return h.handleUpdateServer(msg)
	case "delete_server":
		return h.handleDeleteServer(msg)
	case "health_check":
		return h.handleHealthCheck(msg)
	case "execute_command":
		return h.handleExecuteCommand(msg)
	case "flush_db":
		return h.handleFlushDB(msg)
	case "flush_all":
		return h.handleFlushAll(msg)
	case "get_history":
		return h.handleGetHistory(msg)
	case "record_history":
		return h.handleRecordHistory(msg)
	case "import_servers":
		return h.handleImportServers(msg)
	default:
		return ErrorResponse(msg.ID, fmt.Sprintf("unknown action: %s", msg.Action))
	}
}

// ========================================
// Server Configuration Handlers
// ========================================

func (h *Handler) handleListServers(msg *Message) *Response {
	servers, err := config.Load(h.configPath)
	if err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}
	return SuccessResponse(msg.ID, servers)
}

type addServerParams struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
}

func (h *Handler) handleAddServer(msg *Message) *Response {
	var params addServerParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("invalid params: %v", err))
	}

	// 加载现有配置
	servers, err := config.Load(h.configPath)
	if err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	// 检查名称唯一性
	for _, s := range servers {
		if s.Name == params.Name {
			return ErrorResponse(msg.ID, fmt.Sprintf("server already exists: %s", params.Name))
		}
	}

	// 添加新服务器
	newServer := config.RedisServer{
		Name:     params.Name,
		Host:     params.Host,
		Port:     params.Port,
		Password: params.Password,
	}
	servers = append(servers, newServer)

	// 保存配置
	if err := config.Save(h.configPath, servers); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	return SuccessResponse(msg.ID, newServer)
}

type updateServerParams struct {
	Name   string          `json:"name"`
	Server addServerParams `json:"server"`
}

func (h *Handler) handleUpdateServer(msg *Message) *Response {
	var params updateServerParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("invalid params: %v", err))
	}

	// 加载现有配置
	servers, err := config.Load(h.configPath)
	if err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	// 查找并更新服务器
	found := false
	for i, s := range servers {
		if s.Name == params.Name {
			servers[i] = config.RedisServer{
				Name:     params.Server.Name,
				Host:     params.Server.Host,
				Port:     params.Server.Port,
				Password: params.Server.Password,
			}
			found = true
			break
		}
	}

	if !found {
		return ErrorResponse(msg.ID, fmt.Sprintf("server not found: %s", params.Name))
	}

	// 保存配置
	if err := config.Save(h.configPath, servers); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	return SuccessResponse(msg.ID, servers)
}

type deleteServerParams struct {
	Name string `json:"name"`
}

func (h *Handler) handleDeleteServer(msg *Message) *Response {
	var params deleteServerParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("invalid params: %v", err))
	}

	// 加载现有配置
	servers, err := config.Load(h.configPath)
	if err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	// 查找并删除服务器
	found := false
	newServers := make([]config.RedisServer, 0, len(servers))
	for _, s := range servers {
		if s.Name == params.Name {
			found = true
		} else {
			newServers = append(newServers, s)
		}
	}

	if !found {
		return ErrorResponse(msg.ID, fmt.Sprintf("server not found: %s", params.Name))
	}

	// 保存配置
	if err := config.Save(h.configPath, newServers); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	return SuccessResponse(msg.ID, nil)
}

// ========================================
// Health Check Handler
// ========================================

func (h *Handler) handleHealthCheck(msg *Message) *Response {
	servers, err := config.Load(h.configPath)
	if err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	// 并发健康检查
	results := h.connector.BatchHealthCheck(servers)

	// 构建名称到状态的映射
	statusMap := make(map[string]bool)
	for i, server := range servers {
		statusMap[server.Name] = results[i]
	}

	return SuccessResponse(msg.ID, statusMap)
}

// ========================================
// Command Execution Handlers
// ========================================

type executeCommandParams struct {
	Server  string `json:"server"`
	Command string `json:"command"`
}

func (h *Handler) handleExecuteCommand(msg *Message) *Response {
	var params executeCommandParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("invalid params: %v", err))
	}

	// 查找服务器
	server, err := h.findServer(params.Server)
	if err != nil {
		return ErrorResponse(msg.ID, err.Error())
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", server.Host, server.Port),
		Password: server.Password,
		DB:       0,
	})
	defer client.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("connection failed: %v", err))
	}

	// 解析并执行命令
	args := parseCommand(params.Command)
	if len(args) == 0 {
		return ErrorResponse(msg.ID, "empty command")
	}

	result, err := client.Do(ctx, args...).Result()
	if err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("redis error: %v", err))
	}

	return SuccessResponse(msg.ID, map[string]interface{}{"result": result})
}

type flushParams struct {
	Server  string `json:"server"`
	Confirm bool   `json:"confirm"`
}

func (h *Handler) handleFlushDB(msg *Message) *Response {
	var params flushParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("invalid params: %v", err))
	}

	if !params.Confirm {
		return ErrorResponse(msg.ID, "confirmation required")
	}

	// 查找服务器
	server, err := h.findServer(params.Server)
	if err != nil {
		return ErrorResponse(msg.ID, err.Error())
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", server.Host, server.Port),
		Password: server.Password,
		DB:       0,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 执行 FLUSHDB
	if err := client.FlushDB(ctx).Err(); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("redis error: %v", err))
	}

	return SuccessResponse(msg.ID, map[string]interface{}{"success": true})
}

func (h *Handler) handleFlushAll(msg *Message) *Response {
	var params flushParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("invalid params: %v", err))
	}

	if !params.Confirm {
		return ErrorResponse(msg.ID, "confirmation required")
	}

	// 查找服务器
	server, err := h.findServer(params.Server)
	if err != nil {
		return ErrorResponse(msg.ID, err.Error())
	}

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", server.Host, server.Port),
		Password: server.Password,
		DB:       0,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 执行 FLUSHALL
	if err := client.FlushAll(ctx).Err(); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("redis error: %v", err))
	}

	return SuccessResponse(msg.ID, map[string]interface{}{"success": true})
}

// ========================================
// History Handlers
// ========================================

func (h *Handler) handleGetHistory(msg *Message) *Response {
	if h.historyMgr == nil {
		return SuccessResponse(msg.ID, []string{})
	}
	return SuccessResponse(msg.ID, h.historyMgr.GetRecent())
}

type recordHistoryParams struct {
	Server string `json:"server"`
}

func (h *Handler) handleRecordHistory(msg *Message) *Response {
	var params recordHistoryParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("invalid params: %v", err))
	}

	if h.historyMgr != nil {
		if err := h.historyMgr.Record(params.Server); err != nil {
			return ErrorResponse(msg.ID, fmt.Sprintf("history error: %v", err))
		}
	}

	return SuccessResponse(msg.ID, nil)
}

// ========================================
// Import Handlers
// ========================================

type importServersParams struct {
	Data     string `json:"data"`
	Format   string `json:"format"`   // "yaml" or "json"
	Conflict string `json:"conflict"` // "skip", "overwrite", "rename"
}

type importResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors"`
}

func (h *Handler) handleImportServers(msg *Message) *Response {
	var params importServersParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("invalid params: %v", err))
	}

	// 解析导入数据
	var importedServers []config.RedisServer
	var parseErr error

	switch params.Format {
	case "yaml":
		parseErr = yaml.Unmarshal([]byte(params.Data), &importedServers)
	case "json":
		parseErr = json.Unmarshal([]byte(params.Data), &importedServers)
	default:
		return ErrorResponse(msg.ID, fmt.Sprintf("unsupported format: %s", params.Format))
	}

	if parseErr != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("parse error: %v", parseErr))
	}

	// 加载现有配置
	existingServers, err := config.Load(h.configPath)
	if err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	// 构建现有服务器名称集合
	existingNames := make(map[string]bool)
	for _, s := range existingServers {
		existingNames[s.Name] = true
	}

	// 处理导入
	result := importResult{Errors: []string{}}

	for _, server := range importedServers {
		if existingNames[server.Name] {
			// 名称冲突
			switch params.Conflict {
			case "skip":
				result.Skipped++
			case "overwrite":
				// 更新现有服务器
				for i, s := range existingServers {
					if s.Name == server.Name {
						existingServers[i] = server
						break
					}
				}
				result.Imported++
			case "rename":
				// 自动重命名
				newName := h.generateUniqueName(server.Name, existingNames)
				server.Name = newName
				existingServers = append(existingServers, server)
				existingNames[newName] = true
				result.Imported++
			default:
				result.Skipped++
			}
		} else {
			// 无冲突，直接添加
			existingServers = append(existingServers, server)
			existingNames[server.Name] = true
			result.Imported++
		}
	}

	// 保存配置
	if err := config.Save(h.configPath, existingServers); err != nil {
		return ErrorResponse(msg.ID, fmt.Sprintf("config error: %v", err))
	}

	return SuccessResponse(msg.ID, result)
}

// ========================================
// Helper Functions
// ========================================

func (h *Handler) findServer(name string) (*config.RedisServer, error) {
	servers, err := config.Load(h.configPath)
	if err != nil {
		return nil, fmt.Errorf("config error: %v", err)
	}

	for _, s := range servers {
		if s.Name == name {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("server not found: %s", name)
}

func (h *Handler) generateUniqueName(baseName string, existingNames map[string]bool) string {
	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s_%d", baseName, i)
		if !existingNames[newName] {
			return newName
		}
	}
}

// parseCommand 解析命令字符串为参数列表
func parseCommand(input string) []interface{} {
	var args []interface{}
	var current strings.Builder
	inQuote := false
	escapeNext := false

	for i, char := range input {
		if escapeNext {
			current.WriteRune(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inQuote = !inQuote
			continue
		}

		if char == ' ' && !inQuote {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(char)

		if i == len(input)-1 && current.Len() > 0 {
			args = append(args, current.String())
		}
	}

	return args
}
