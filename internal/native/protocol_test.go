package native

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// ========================================
// Protocol Layer Tests
// ========================================

// Feature: browser-extension, Property 1: 消息编码往返一致性
// 对于任意有效的 Native Messaging 消息，使用 4 字节小端序长度前缀编码后再解码，
// 应该产生与原始消息等价的结构。
func TestProperty_MessageRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("消息编码往返一致性", prop.ForAll(
		func(id, action string) bool {
			// 创建原始消息
			original := &Message{
				ID:     id,
				Action: action,
			}

			// 编码
			encoded, err := EncodeMessage(original)
			if err != nil {
				return false
			}

			// 解码
			decoded, err := ReadMessage(bytes.NewReader(encoded))
			if err != nil {
				return false
			}

			// 验证往返一致性
			return original.ID == decoded.ID && original.Action == decoded.Action
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// Feature: browser-extension, Property 1: 带参数的消息往返一致性
func TestProperty_MessageWithParamsRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("带参数消息往返一致性", prop.ForAll(
		func(id, action, paramKey, paramValue string) bool {
			// 创建带参数的消息
			params := map[string]string{paramKey: paramValue}
			paramsJSON, _ := json.Marshal(params)

			original := &Message{
				ID:     id,
				Action: action,
				Params: paramsJSON,
			}

			// 编码
			encoded, err := EncodeMessage(original)
			if err != nil {
				return false
			}

			// 解码
			decoded, err := ReadMessage(bytes.NewReader(encoded))
			if err != nil {
				return false
			}

			// 验证往返一致性
			if original.ID != decoded.ID || original.Action != decoded.Action {
				return false
			}

			// 验证参数
			var decodedParams map[string]string
			if err := json.Unmarshal(decoded.Params, &decodedParams); err != nil {
				return false
			}

			return decodedParams[paramKey] == paramValue
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// ========================================
// Unit Tests
// ========================================

func TestReadMessage_ValidJSON(t *testing.T) {
	msg := &Message{
		ID:     "test-123",
		Action: "list_servers",
	}

	encoded, err := EncodeMessage(msg)
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	decoded, err := ReadMessage(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if decoded.ID != msg.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, msg.ID)
	}

	if decoded.Action != msg.Action {
		t.Errorf("Action mismatch: got %s, want %s", decoded.Action, msg.Action)
	}
}

func TestReadMessage_InvalidJSON(t *testing.T) {
	// 创建无效 JSON 的消息
	invalidJSON := []byte("not valid json")
	buf := make([]byte, 4+len(invalidJSON))
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(invalidJSON)))
	copy(buf[4:], invalidJSON)

	_, err := ReadMessage(bytes.NewReader(buf))
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestReadMessage_EOF(t *testing.T) {
	_, err := ReadMessage(bytes.NewReader([]byte{}))
	if err != io.EOF {
		t.Errorf("Expected io.EOF, got %v", err)
	}
}

func TestReadMessage_TooLarge(t *testing.T) {
	// 创建超大消息长度
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 2*1024*1024) // 2MB

	_, err := ReadMessage(bytes.NewReader(buf))
	if err == nil {
		t.Error("Expected error for too large message, got nil")
	}
}

func TestWriteResponse_Success(t *testing.T) {
	resp := SuccessResponse("test-123", map[string]string{"key": "value"})

	var buf bytes.Buffer
	err := WriteResponse(&buf, resp)
	if err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}

	// 验证长度前缀
	if buf.Len() < 4 {
		t.Fatal("Response too short")
	}

	length := binary.LittleEndian.Uint32(buf.Bytes()[:4])
	if int(length) != buf.Len()-4 {
		t.Errorf("Length mismatch: got %d, want %d", length, buf.Len()-4)
	}

	// 验证 JSON 内容
	var decoded Response
	if err := json.Unmarshal(buf.Bytes()[4:], &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decoded.ID != "test-123" {
		t.Errorf("ID mismatch: got %s, want test-123", decoded.ID)
	}

	if !decoded.Success {
		t.Error("Expected success to be true")
	}
}

func TestWriteResponse_Error(t *testing.T) {
	resp := ErrorResponse("test-456", "something went wrong")

	var buf bytes.Buffer
	err := WriteResponse(&buf, resp)
	if err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}

	// 验证 JSON 内容
	var decoded Response
	if err := json.Unmarshal(buf.Bytes()[4:], &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decoded.ID != "test-456" {
		t.Errorf("ID mismatch: got %s, want test-456", decoded.ID)
	}

	if decoded.Success {
		t.Error("Expected success to be false")
	}

	if decoded.Error != "something went wrong" {
		t.Errorf("Error mismatch: got %s, want 'something went wrong'", decoded.Error)
	}
}

func TestSuccessResponse(t *testing.T) {
	resp := SuccessResponse("id-1", "data")

	if resp.ID != "id-1" {
		t.Errorf("ID mismatch: got %s, want id-1", resp.ID)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}

	if resp.Data != "data" {
		t.Errorf("Data mismatch: got %v, want 'data'", resp.Data)
	}

	if resp.Error != "" {
		t.Errorf("Error should be empty, got %s", resp.Error)
	}
}

func TestErrorResponse(t *testing.T) {
	resp := ErrorResponse("id-2", "error message")

	if resp.ID != "id-2" {
		t.Errorf("ID mismatch: got %s, want id-2", resp.ID)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}

	if resp.Error != "error message" {
		t.Errorf("Error mismatch: got %s, want 'error message'", resp.Error)
	}
}
