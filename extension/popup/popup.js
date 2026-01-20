// ========================================
// Redisw Browser Extension - Popup Script
// ========================================

// ========================================
// State
// ========================================

let servers = [];
let healthStatus = {};
let recentServers = [];
let currentServer = null;
let commandHistory = [];
let historyIndex = -1;
let editingServer = null;
let importData = null;

// ========================================
// API Helpers
// ========================================

async function api(action, params = null) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage({ action, params }, (response) => {
      if (chrome.runtime.lastError) {
        reject(new Error(chrome.runtime.lastError.message));
      } else if (response.success) {
        resolve(response.data);
      } else {
        reject(new Error(response.error));
      }
    });
  });
}

// ========================================
// View Management
// ========================================

function showView(viewId) {
  document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
  document.getElementById(viewId).classList.add('active');
}

// ========================================
// Server List
// ========================================

async function loadServers() {
  showLoading(true);
  hideError();
  
  try {
    [servers, healthStatus, recentServers] = await Promise.all([
      api('listServers'),
      api('healthCheck'),
      api('getHistory')
    ]);
    
    renderServerList();
  } catch (error) {
    showError(error.message);
  } finally {
    showLoading(false);
  }
}

function renderServerList(filter = '') {
  const list = document.getElementById('server-list');
  const filterLower = filter.toLowerCase();
  
  // 按历史记录排序
  const sorted = [...servers].sort((a, b) => {
    const aIndex = recentServers.indexOf(a.name);
    const bIndex = recentServers.indexOf(b.name);
    if (aIndex >= 0 && bIndex >= 0) return aIndex - bIndex;
    if (aIndex >= 0) return -1;
    if (bIndex >= 0) return 1;
    return 0;
  });
  
  // 过滤
  const filtered = sorted.filter(s => 
    s.name.toLowerCase().includes(filterLower)
  );
  
  if (filtered.length === 0) {
    list.innerHTML = '<div class="loading">没有找到服务器</div>';
    return;
  }
  
  list.innerHTML = filtered.map(server => {
    const isHealthy = healthStatus[server.name];
    const isRecent = recentServers.includes(server.name);
    
    return `
      <div class="server-item" data-name="${escapeHtml(server.name)}">
        <span class="name">${escapeHtml(server.name)}</span>
        ${isRecent ? '<span class="recent">★</span>' : ''}
        <span class="status ${isHealthy ? 'healthy' : 'unhealthy'}">
          ${isHealthy ? '✓' : '✗'}
        </span>
        <div class="actions">
          <button class="btn-edit" title="编辑">✏️</button>
          <button class="btn-delete" title="删除">🗑️</button>
        </div>
      </div>
    `;
  }).join('');
  
  // 绑定事件
  list.querySelectorAll('.server-item').forEach(item => {
    const name = item.dataset.name;
    
    item.addEventListener('click', (e) => {
      if (!e.target.closest('.actions')) {
        connectToServer(name);
      }
    });
    
    item.querySelector('.btn-edit').addEventListener('click', (e) => {
      e.stopPropagation();
      editServer(name);
    });
    
    item.querySelector('.btn-delete').addEventListener('click', (e) => {
      e.stopPropagation();
      confirmDeleteServer(name);
    });
  });
}

// ========================================
// Terminal
// ========================================

async function connectToServer(name) {
  const server = servers.find(s => s.name === name);
  if (!server) return;
  
  currentServer = server;
  commandHistory = [];
  historyIndex = -1;
  
  // 更新 UI
  document.getElementById('current-server-name').textContent = server.name;
  document.getElementById('current-server-status').className = 
    `status ${healthStatus[server.name] ? 'healthy' : 'unhealthy'}`;
  document.getElementById('server-address').textContent = 
    `${server.host}:${server.port}`;
  document.getElementById('terminal-prompt').textContent = 
    `${server.host}:${server.port}> `;
  document.getElementById('terminal-output').innerHTML = '';
  document.getElementById('terminal-input').value = '';
  
  // 记录历史
  try {
    await api('recordHistory', { server: name });
  } catch (e) {
    console.error('Failed to record history:', e);
  }
  
  showView('terminal-view');
  document.getElementById('terminal-input').focus();
}

async function executeCommand(command) {
  if (!currentServer || !command.trim()) return;
  
  const output = document.getElementById('terminal-output');
  const prompt = `${currentServer.host}:${currentServer.port}> `;
  
  // 显示命令
  output.innerHTML += `<div><span class="prompt">${prompt}</span>${escapeHtml(command)}</div>`;
  
  // 添加到历史
  commandHistory.push(command);
  historyIndex = commandHistory.length;
  
  try {
    const result = await api('executeCommand', { 
      server: currentServer.name, 
      command 
    });
    output.innerHTML += formatResult(result.result);
  } catch (error) {
    output.innerHTML += `<div class="result-error">(error) ${escapeHtml(error.message)}</div>`;
  }
  
  // 滚动到底部
  const terminal = document.getElementById('terminal');
  terminal.scrollTop = terminal.scrollHeight;
}

function formatResult(result) {
  if (result === null) {
    return '<div class="result-nil">(nil)</div>';
  }
  
  if (typeof result === 'string') {
    return `<div class="result-string">"${escapeHtml(result)}"</div>`;
  }
  
  if (typeof result === 'number') {
    return `<div class="result-number">(integer) ${result}</div>`;
  }
  
  if (Array.isArray(result)) {
    if (result.length === 0) {
      return '<div class="result-nil">(empty array)</div>';
    }
    return result.map((item, i) => 
      `<div>${i + 1}) ${formatResultInline(item)}</div>`
    ).join('');
  }
  
  return `<div>${escapeHtml(String(result))}</div>`;
}

function formatResultInline(result) {
  if (result === null) return '<span class="result-nil">(nil)</span>';
  if (typeof result === 'string') return `<span class="result-string">"${escapeHtml(result)}"</span>`;
  if (typeof result === 'number') return `<span class="result-number">${result}</span>`;
  return escapeHtml(String(result));
}

// ========================================
// Dangerous Operations
// ========================================

function showConfirmDialog(title, message, serverInfo, onConfirm) {
  document.getElementById('dialog-title').textContent = title;
  document.getElementById('dialog-message').textContent = message;
  document.getElementById('dialog-server-info').textContent = serverInfo;
  
  const dialog = document.getElementById('confirm-dialog');
  dialog.classList.remove('hidden');
  
  const confirmBtn = document.getElementById('btn-dialog-confirm');
  const cancelBtn = document.getElementById('btn-dialog-cancel');
  
  const cleanup = () => {
    dialog.classList.add('hidden');
    confirmBtn.replaceWith(confirmBtn.cloneNode(true));
    cancelBtn.replaceWith(cancelBtn.cloneNode(true));
  };
  
  document.getElementById('btn-dialog-confirm').addEventListener('click', () => {
    cleanup();
    onConfirm();
  });
  
  document.getElementById('btn-dialog-cancel').addEventListener('click', cleanup);
}

async function flushDatabase(type) {
  if (!currentServer) return;
  
  const action = type === 'all' ? 'FLUSHALL' : 'FLUSHDB';
  const message = type === 'all' 
    ? '这将清空所有数据库的所有数据，此操作不可撤销！'
    : '这将清空当前数据库的所有数据，此操作不可撤销！';
  
  showConfirmDialog(
    `⚠️ 确认执行 ${action}`,
    message,
    `服务器: ${currentServer.name} (${currentServer.host}:${currentServer.port})`,
    async () => {
      const output = document.getElementById('terminal-output');
      try {
        const apiAction = type === 'all' ? 'flushAll' : 'flushDb';
        await api(apiAction, { server: currentServer.name, confirm: true });
        output.innerHTML += `<div class="result-string">OK</div>`;
      } catch (error) {
        output.innerHTML += `<div class="result-error">(error) ${escapeHtml(error.message)}</div>`;
      }
    }
  );
}

// ========================================
// Server Form
// ========================================

function showAddServerForm() {
  editingServer = null;
  document.getElementById('form-title').textContent = '添加服务器';
  document.getElementById('server-form').reset();
  document.getElementById('server-port').value = '6379';
  showView('server-form-view');
}

function editServer(name) {
  const server = servers.find(s => s.name === name);
  if (!server) return;
  
  editingServer = server;
  document.getElementById('form-title').textContent = '编辑服务器';
  document.getElementById('server-name').value = server.name;
  document.getElementById('server-host').value = server.host;
  document.getElementById('server-port').value = server.port;
  document.getElementById('server-password').value = server.password || '';
  showView('server-form-view');
}

async function saveServer(e) {
  e.preventDefault();
  
  const serverData = {
    name: document.getElementById('server-name').value.trim(),
    host: document.getElementById('server-host').value.trim(),
    port: parseInt(document.getElementById('server-port').value, 10),
    password: document.getElementById('server-password').value
  };
  
  try {
    if (editingServer) {
      await api('updateServer', { name: editingServer.name, server: serverData });
    } else {
      await api('addServer', serverData);
    }
    
    showView('server-list-view');
    await loadServers();
  } catch (error) {
    alert('保存失败: ' + error.message);
  }
}

function confirmDeleteServer(name) {
  const server = servers.find(s => s.name === name);
  if (!server) return;
  
  showConfirmDialog(
    '确认删除',
    `确定要删除服务器 "${name}" 吗？`,
    `${server.host}:${server.port}`,
    async () => {
      try {
        await api('deleteServer', { name });
        await loadServers();
      } catch (error) {
        alert('删除失败: ' + error.message);
      }
    }
  );
}

// ========================================
// Import
// ========================================

function showImportView() {
  importData = null;
  document.getElementById('import-preview').classList.add('hidden');
  document.getElementById('import-actions').classList.add('hidden');
  showView('import-view');
}

function handleFileSelect(file) {
  if (!file) return;
  
  const reader = new FileReader();
  reader.onload = (e) => {
    const content = e.target.result;
    const format = file.name.endsWith('.json') ? 'json' : 'yaml';
    
    try {
      // 简单验证
      if (format === 'json') {
        importData = { data: content, format, servers: JSON.parse(content) };
      } else {
        // YAML 解析在后端进行，这里只存储原始数据
        importData = { data: content, format, servers: [] };
      }
      
      showImportPreview();
    } catch (error) {
      alert('文件解析失败: ' + error.message);
    }
  };
  reader.readAsText(file);
}

function showImportPreview() {
  if (!importData) return;
  
  const previewList = document.getElementById('preview-list');
  const existingNames = new Set(servers.map(s => s.name));
  
  if (importData.format === 'json' && importData.servers.length > 0) {
    document.getElementById('preview-count').textContent = importData.servers.length;
    
    previewList.innerHTML = importData.servers.map(server => {
      const isConflict = existingNames.has(server.name);
      return `
        <div class="preview-item ${isConflict ? 'conflict' : ''}">
          <span class="icon">${isConflict ? '⚠' : '✓'}</span>
          <span class="name">${escapeHtml(server.name)}</span>
          <span class="address">${escapeHtml(server.host)}:${server.port}${isConflict ? ' (冲突)' : ''}</span>
        </div>
      `;
    }).join('');
  } else {
    document.getElementById('preview-count').textContent = '?';
    previewList.innerHTML = '<div class="preview-item">YAML 文件将在导入时解析</div>';
  }
  
  document.getElementById('import-preview').classList.remove('hidden');
  document.getElementById('import-actions').classList.remove('hidden');
}

async function confirmImport() {
  if (!importData) return;
  
  const conflict = document.querySelector('input[name="conflict"]:checked').value;
  
  try {
    const result = await api('importServers', {
      data: importData.data,
      format: importData.format,
      conflict
    });
    
    alert(`导入完成！\n成功: ${result.imported}\n跳过: ${result.skipped}\n失败: ${result.failed}`);
    showView('server-list-view');
    await loadServers();
  } catch (error) {
    alert('导入失败: ' + error.message);
  }
}

// ========================================
// Utilities
// ========================================

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

function showLoading(show) {
  document.getElementById('loading').classList.toggle('hidden', !show);
}

function showError(message) {
  const el = document.getElementById('error-message');
  el.textContent = message;
  el.classList.remove('hidden');
}

function hideError() {
  document.getElementById('error-message').classList.add('hidden');
}

// ========================================
// Event Listeners
// ========================================

document.addEventListener('DOMContentLoaded', () => {
  // 初始加载
  loadServers();
  
  // 搜索
  document.getElementById('search-input').addEventListener('input', (e) => {
    renderServerList(e.target.value);
  });
  
  // 添加服务器
  document.getElementById('btn-add').addEventListener('click', showAddServerForm);
  
  // 导入
  document.getElementById('btn-import').addEventListener('click', showImportView);
  
  // 返回按钮
  document.getElementById('btn-back').addEventListener('click', () => {
    currentServer = null;
    showView('server-list-view');
    loadServers();
  });
  
  document.getElementById('btn-form-back').addEventListener('click', () => {
    showView('server-list-view');
  });
  
  document.getElementById('btn-form-cancel').addEventListener('click', () => {
    showView('server-list-view');
  });
  
  document.getElementById('btn-import-back').addEventListener('click', () => {
    showView('server-list-view');
  });
  
  // 服务器表单
  document.getElementById('server-form').addEventListener('submit', saveServer);
  
  // 终端输入
  document.getElementById('terminal-input').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      const input = e.target;
      executeCommand(input.value);
      input.value = '';
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (historyIndex > 0) {
        historyIndex--;
        e.target.value = commandHistory[historyIndex];
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIndex < commandHistory.length - 1) {
        historyIndex++;
        e.target.value = commandHistory[historyIndex];
      } else {
        historyIndex = commandHistory.length;
        e.target.value = '';
      }
    }
  });
  
  // 危险操作
  document.getElementById('btn-flushdb').addEventListener('click', () => flushDatabase('db'));
  document.getElementById('btn-flushall').addEventListener('click', () => flushDatabase('all'));
  
  // 文件导入
  const dropZone = document.getElementById('drop-zone');
  const fileInput = document.getElementById('import-file');
  
  dropZone.addEventListener('click', () => fileInput.click());
  
  dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropZone.classList.add('dragover');
  });
  
  dropZone.addEventListener('dragleave', () => {
    dropZone.classList.remove('dragover');
  });
  
  dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    dropZone.classList.remove('dragover');
    handleFileSelect(e.dataTransfer.files[0]);
  });
  
  fileInput.addEventListener('change', (e) => {
    handleFileSelect(e.target.files[0]);
  });
  
  document.getElementById('btn-import-cancel').addEventListener('click', () => {
    showView('server-list-view');
  });
  
  document.getElementById('btn-import-confirm').addEventListener('click', confirmImport);
});
