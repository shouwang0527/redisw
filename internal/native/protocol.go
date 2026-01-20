package native

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// ========================================
// Native Messaging Protocol
// 实现浏览器 Native Messaging 协议的消息编解码
// 使用 4 字节小端序长度前缀
// ========================================

// Message 表示 Native Messaging 请求消息
type Message struct {
	ID     string          `json:"id"`     // 请求 ID，用于匹配响应
	Action string          `json:"action"` // 操作类型
	Params json.RawMessage `json:"params"` // 操作参数（可选）
}

// Response 表示 Native Messaging 响应消息
type Response struct {
	ID      string      `json:"id"`      // 对应请求 ID
	Success bool        `json:"success"` // 操作是否成功
	Data    interface{} `json:"data"`    // 成功时的数据
	Error   string      `json:"error"`   // 失败时的错误信息
}

// ReadMessage 从 Reader 读取一条 Native Messaging 消息
// Native Messaging 协议使用 4 字节小端序长度前缀
func ReadMessage(r io.Reader) (*Message, error) {
	// 读取 4 字节长度前缀
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lengthBuf); err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}

	// 解析长度（小端序）
	length := binary.LittleEndian.Uint32(lengthBuf)

	// 安全检查：限制消息大小（最大 1MB）
	if length > 1024*1024 {
		return nil, fmt.Errorf("message too large: %d bytes", length)
	}

	// 读取消息体
	msgBuf := make([]byte, length)
	if _, err := io.ReadFull(r, msgBuf); err != nil {
		return nil, fmt.Errorf("failed to read message body: %w", err)
	}

	// 解析 JSON
	var msg Message
	if err := json.Unmarshal(msgBuf, &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return &msg, nil
}

// WriteResponse 向 Writer 写入响应消息
// 使用 4 字节小端序长度前缀
func WriteResponse(w io.Writer, resp *Response) error {
	// 序列化响应为 JSON
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// 写入 4 字节长度前缀（小端序）
	lengthBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBuf, uint32(len(data)))

	if _, err := w.Write(lengthBuf); err != nil {
		return fmt.Errorf("failed to write length prefix: %w", err)
	}

	// 写入消息体
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write message body: %w", err)
	}

	return nil
}

// EncodeMessage 将消息编码为带长度前缀的字节数组
// 用于测试和调试
func EncodeMessage(msg *Message) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// 创建带长度前缀的缓冲区
	result := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(result[:4], uint32(len(data)))
	copy(result[4:], data)

	return result, nil
}

// SuccessResponse 创建成功响应
func SuccessResponse(id string, data interface{}) *Response {
	return &Response{
		ID:      id,
		Success: true,
		Data:    data,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(id string, errMsg string) *Response {
	return &Response{
		ID:      id,
		Success: false,
		Error:   errMsg,
	}
}
