// ========================================
// Redisw Browser Extension - Background Script
// Native Messaging 通信管理
// ========================================

const NATIVE_HOST_NAME = 'com.redisw.native';

/**
 * NativeClient - 封装 Native Messaging 通信
 */
class NativeClient {
  constructor() {
    this.port = null;
    this.pendingRequests = new Map();
    this.requestId = 0;
    this.connected = false;
  }

  /**
   * 连接到 Native Host
   */
  connect() {
    if (this.connected && this.port) {
      return true;
    }

    try {
      this.port = chrome.runtime.connectNative(NATIVE_HOST_NAME);
      
      this.port.onMessage.addListener((response) => {
        this.handleResponse(response);
      });

      this.port.onDisconnect.addListener(() => {
        this.handleDisconnect();
      });

      this.connected = true;
      console.log('Connected to Native Host');
      return true;
    } catch (error) {
      console.error('Failed to connect to Native Host:', error);
      return false;
    }
  }

  /**
   * 断开连接
   */
  disconnect() {
    if (this.port) {
      this.port.disconnect();
      this.port = null;
    }
    this.connected = false;
    this.pendingRequests.clear();
  }

  /**
   * 处理响应消息
   */
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

  /**
   * 处理断开连接
   */
  handleDisconnect() {
    const error = chrome.runtime.lastError;
    console.log('Disconnected from Native Host:', error?.message || 'Unknown reason');
    
    this.connected = false;
    this.port = null;

    // 拒绝所有待处理的请求
    for (const [id, pending] of this.pendingRequests) {
      pending.reject(new Error('Connection lost'));
    }
    this.pendingRequests.clear();
  }

  /**
   * 发送请求并等待响应
   */
  async request(action, params = null) {
    if (!this.connected) {
      if (!this.connect()) {
        throw new Error('Failed to connect to Native Host. Please ensure redisw is installed and run "redisw install-native".');
      }
    }

    const id = String(++this.requestId);
    const message = { id, action };
    if (params !== null) {
      message.params = params;
    }

    return new Promise((resolve, reject) => {
      // 设置超时
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(id);
        reject(new Error('Request timeout'));
      }, 30000);

      this.pendingRequests.set(id, {
        resolve: (data) => {
          clearTimeout(timeout);
          resolve(data);
        },
        reject: (error) => {
          clearTimeout(timeout);
          reject(error);
        }
      });

      try {
        this.port.postMessage(message);
      } catch (error) {
        this.pendingRequests.delete(id);
        clearTimeout(timeout);
        reject(error);
      }
    });
  }
}

// 全局 NativeClient 实例
const nativeClient = new NativeClient();

// ========================================
// API Functions
// ========================================

async function listServers() {
  return nativeClient.request('list_servers');
}

async function addServer(server) {
  return nativeClient.request('add_server', server);
}

async function updateServer(name, server) {
  return nativeClient.request('update_server', { name, server });
}

async function deleteServer(name) {
  return nativeClient.request('delete_server', { name });
}

async function healthCheck() {
  return nativeClient.request('health_check');
}

async function executeCommand(server, command) {
  return nativeClient.request('execute_command', { server, command });
}

async function flushDb(server, confirm) {
  return nativeClient.request('flush_db', { server, confirm });
}

async function flushAll(server, confirm) {
  return nativeClient.request('flush_all', { server, confirm });
}

async function getHistory() {
  return nativeClient.request('get_history');
}

async function recordHistory(server) {
  return nativeClient.request('record_history', { server });
}

async function importServers(data, format, conflict) {
  return nativeClient.request('import_servers', { data, format, conflict });
}

// ========================================
// Message Handler for Popup
// ========================================

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  const { action, params } = request;

  let promise;
  switch (action) {
    case 'listServers':
      promise = listServers();
      break;
    case 'addServer':
      promise = addServer(params);
      break;
    case 'updateServer':
      promise = updateServer(params.name, params.server);
      break;
    case 'deleteServer':
      promise = deleteServer(params.name);
      break;
    case 'healthCheck':
      promise = healthCheck();
      break;
    case 'executeCommand':
      promise = executeCommand(params.server, params.command);
      break;
    case 'flushDb':
      promise = flushDb(params.server, params.confirm);
      break;
    case 'flushAll':
      promise = flushAll(params.server, params.confirm);
      break;
    case 'getHistory':
      promise = getHistory();
      break;
    case 'recordHistory':
      promise = recordHistory(params.server);
      break;
    case 'importServers':
      promise = importServers(params.data, params.format, params.conflict);
      break;
    default:
      sendResponse({ success: false, error: `Unknown action: ${action}` });
      return false;
  }

  promise
    .then(data => sendResponse({ success: true, data }))
    .catch(error => sendResponse({ success: false, error: error.message }));

  return true; // 保持消息通道开放
});
