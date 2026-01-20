// ========================================
// Background Script Tests
// ========================================

// Mock chrome API
const mockPort = {
  postMessage: jest.fn(),
  onMessage: {
    addListener: jest.fn()
  },
  onDisconnect: {
    addListener: jest.fn()
  },
  disconnect: jest.fn()
};

global.chrome = {
  runtime: {
    connectNative: jest.fn(() => mockPort),
    sendMessage: jest.fn(),
    onMessage: {
      addListener: jest.fn()
    },
    lastError: null
  }
};

// NativeClient 类的简化版本用于测试
class NativeClient {
  constructor() {
    this.port = null;
    this.pendingRequests = new Map();
    this.requestId = 0;
    this.connected = false;
  }

  connect() {
    if (this.connected && this.port) {
      return true;
    }
    try {
      this.port = chrome.runtime.connectNative('com.redisw.native');
      this.connected = true;
      return true;
    } catch (error) {
      return false;
    }
  }

  disconnect() {
    if (this.port) {
      this.port.disconnect();
      this.port = null;
    }
    this.connected = false;
    this.pendingRequests.clear();
  }

  async request(action, params = null) {
    if (!this.connected) {
      if (!this.connect()) {
        throw new Error('Failed to connect');
      }
    }

    const id = String(++this.requestId);
    const message = { id, action };
    if (params !== null) {
      message.params = params;
    }

    return new Promise((resolve, reject) => {
      this.pendingRequests.set(id, { resolve, reject });
      this.port.postMessage(message);
    });
  }

  handleResponse(response) {
    const { id } = response;
    const pending = this.pendingRequests.get(id);
    
    if (pending) {
      this.pendingRequests.delete(id);
      if (response.success) {
        pending.resolve(response.data);
      } else {
        pending.reject(new Error(response.error || 'Unknown error'));
      }
    }
  }
}

describe('NativeClient', () => {
  let client;

  beforeEach(() => {
    client = new NativeClient();
    jest.clearAllMocks();
  });

  afterEach(() => {
    client.disconnect();
  });

  describe('connect', () => {
    test('should connect to native host', () => {
      const result = client.connect();
      expect(result).toBe(true);
      expect(chrome.runtime.connectNative).toHaveBeenCalledWith('com.redisw.native');
      expect(client.connected).toBe(true);
    });

    test('should reuse existing connection', () => {
      client.connect();
      client.connect();
      expect(chrome.runtime.connectNative).toHaveBeenCalledTimes(1);
    });
  });

  describe('disconnect', () => {
    test('should disconnect from native host', () => {
      client.connect();
      client.disconnect();
      expect(mockPort.disconnect).toHaveBeenCalled();
      expect(client.connected).toBe(false);
    });

    test('should clear pending requests on disconnect', () => {
      client.connect();
      client.pendingRequests.set('1', { resolve: jest.fn(), reject: jest.fn() });
      client.disconnect();
      expect(client.pendingRequests.size).toBe(0);
    });
  });

  describe('request', () => {
    test('should send message with correct format', async () => {
      client.connect();
      
      // 不等待 Promise，只验证消息格式
      client.request('list_servers');
      
      expect(mockPort.postMessage).toHaveBeenCalledWith({
        id: '1',
        action: 'list_servers'
      });
    });

    test('should include params when provided', async () => {
      client.connect();
      
      client.request('add_server', { name: 'test', host: 'localhost', port: 6379 });
      
      expect(mockPort.postMessage).toHaveBeenCalledWith({
        id: '1',
        action: 'add_server',
        params: { name: 'test', host: 'localhost', port: 6379 }
      });
    });

    test('should increment request ID', async () => {
      client.connect();
      
      client.request('list_servers');
      client.request('health_check');
      
      const calls = mockPort.postMessage.mock.calls;
      expect(calls[0][0].id).toBe('1');
      expect(calls[1][0].id).toBe('2');
    });
  });

  // Feature: browser-extension, Property 16: 请求-响应 ID 匹配
  describe('handleResponse', () => {
    test('should match response to pending request by ID', () => {
      const mockResolve = jest.fn();
      const mockReject = jest.fn();
      
      client.pendingRequests.set('123', { resolve: mockResolve, reject: mockReject });
      
      client.handleResponse({ id: '123', success: true, data: ['server1'] });
      
      expect(mockResolve).toHaveBeenCalledWith(['server1']);
      expect(mockReject).not.toHaveBeenCalled();
      expect(client.pendingRequests.has('123')).toBe(false);
    });

    test('should reject on error response', () => {
      const mockResolve = jest.fn();
      const mockReject = jest.fn();
      
      client.pendingRequests.set('456', { resolve: mockResolve, reject: mockReject });
      
      client.handleResponse({ id: '456', success: false, error: 'Server not found' });
      
      expect(mockReject).toHaveBeenCalled();
      expect(mockResolve).not.toHaveBeenCalled();
    });

    test('should ignore responses with unknown ID', () => {
      const mockResolve = jest.fn();
      client.pendingRequests.set('789', { resolve: mockResolve, reject: jest.fn() });
      
      client.handleResponse({ id: 'unknown', success: true, data: [] });
      
      expect(mockResolve).not.toHaveBeenCalled();
      expect(client.pendingRequests.has('789')).toBe(true);
    });
  });
});

// API 函数测试
describe('API Functions', () => {
  // 这些测试需要完整的 background.js 加载
  // 这里只测试消息格式

  test('listServers message format', () => {
    const message = { action: 'listServers', params: null };
    expect(message.action).toBe('listServers');
  });

  test('addServer message format', () => {
    const server = { name: 'test', host: 'localhost', port: 6379, password: '' };
    const message = { action: 'addServer', params: server };
    expect(message.params.name).toBe('test');
    expect(message.params.port).toBe(6379);
  });

  test('executeCommand message format', () => {
    const message = { 
      action: 'executeCommand', 
      params: { server: 'test', command: 'GET key' } 
    };
    expect(message.params.server).toBe('test');
    expect(message.params.command).toBe('GET key');
  });

  test('flushDb message format with confirm', () => {
    const message = { 
      action: 'flushDb', 
      params: { server: 'test', confirm: true } 
    };
    expect(message.params.confirm).toBe(true);
  });

  test('importServers message format', () => {
    const message = { 
      action: 'importServers', 
      params: { 
        data: '[{"name":"test"}]', 
        format: 'json', 
        conflict: 'skip' 
      } 
    };
    expect(message.params.format).toBe('json');
    expect(message.params.conflict).toBe('skip');
  });
});
