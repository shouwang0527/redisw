// ========================================
// Popup UI Tests
// ========================================

// Feature: browser-extension, Property 14: 服务器列表渲染完整性
describe('Server List Rendering', () => {
  // 模拟服务器数据
  const mockServers = [
    { name: 'production', host: '192.168.1.100', port: 6379, password: '' },
    { name: 'staging', host: '192.168.1.101', port: 6379, password: '' },
    { name: 'development', host: 'localhost', port: 6379, password: '' }
  ];

  const mockHealthStatus = {
    'production': true,
    'staging': false,
    'development': true
  };

  const mockRecentServers = ['production', 'development'];

  // 渲染函数（从 popup.js 提取的逻辑）
  function renderServerItem(server, healthStatus, recentServers) {
    const isHealthy = healthStatus[server.name];
    const isRecent = recentServers.includes(server.name);
    
    return {
      name: server.name,
      healthyMark: isHealthy ? '✓' : '✗',
      recentMark: isRecent ? '★' : '',
      address: `${server.host}:${server.port}`
    };
  }

  test('should include server name in rendered item', () => {
    const server = mockServers[0];
    const rendered = renderServerItem(server, mockHealthStatus, mockRecentServers);
    expect(rendered.name).toBe('production');
  });

  test('should show healthy mark for reachable server', () => {
    const server = mockServers[0]; // production - healthy
    const rendered = renderServerItem(server, mockHealthStatus, mockRecentServers);
    expect(rendered.healthyMark).toBe('✓');
  });

  test('should show unhealthy mark for unreachable server', () => {
    const server = mockServers[1]; // staging - unhealthy
    const rendered = renderServerItem(server, mockHealthStatus, mockRecentServers);
    expect(rendered.healthyMark).toBe('✗');
  });

  test('should show recent mark for recently used server', () => {
    const server = mockServers[0]; // production - recent
    const rendered = renderServerItem(server, mockHealthStatus, mockRecentServers);
    expect(rendered.recentMark).toBe('★');
  });

  test('should not show recent mark for non-recent server', () => {
    const server = mockServers[1]; // staging - not recent
    const rendered = renderServerItem(server, mockHealthStatus, mockRecentServers);
    expect(rendered.recentMark).toBe('');
  });

  test('should render all servers with complete information', () => {
    const rendered = mockServers.map(s => 
      renderServerItem(s, mockHealthStatus, mockRecentServers)
    );

    expect(rendered.length).toBe(3);
    rendered.forEach(item => {
      expect(item.name).toBeTruthy();
      expect(item.healthyMark).toMatch(/[✓✗]/);
      expect(item.address).toMatch(/:\d+$/);
    });
  });
});

// Feature: browser-extension, Property 15: 模糊搜索过滤
describe('Fuzzy Search Filter', () => {
  const mockServers = [
    { name: 'production-redis', host: '192.168.1.100', port: 6379 },
    { name: 'staging-redis', host: '192.168.1.101', port: 6379 },
    { name: 'development', host: 'localhost', port: 6379 },
    { name: 'test-cache', host: 'localhost', port: 6380 }
  ];

  // 过滤函数（从 popup.js 提取的逻辑）
  function filterServers(servers, query) {
    const queryLower = query.toLowerCase();
    return servers.filter(s => 
      s.name.toLowerCase().includes(queryLower)
    );
  }

  test('should return all servers for empty query', () => {
    const result = filterServers(mockServers, '');
    expect(result.length).toBe(4);
  });

  test('should filter by exact name match', () => {
    const result = filterServers(mockServers, 'production-redis');
    expect(result.length).toBe(1);
    expect(result[0].name).toBe('production-redis');
  });

  test('should filter by partial name match', () => {
    const result = filterServers(mockServers, 'redis');
    expect(result.length).toBe(2);
    expect(result.map(s => s.name)).toContain('production-redis');
    expect(result.map(s => s.name)).toContain('staging-redis');
  });

  test('should be case insensitive', () => {
    const result = filterServers(mockServers, 'PRODUCTION');
    expect(result.length).toBe(1);
    expect(result[0].name).toBe('production-redis');
  });

  test('should return empty array for no matches', () => {
    const result = filterServers(mockServers, 'nonexistent');
    expect(result.length).toBe(0);
  });

  test('should filter by single character', () => {
    const result = filterServers(mockServers, 'd');
    expect(result.length).toBe(2); // development, production-redis
    expect(result.some(s => s.name.includes('d'))).toBe(true);
  });
});

// 排序测试
describe('Server Sorting by History', () => {
  const mockServers = [
    { name: 'alpha', host: 'localhost', port: 6379 },
    { name: 'beta', host: 'localhost', port: 6380 },
    { name: 'gamma', host: 'localhost', port: 6381 },
    { name: 'delta', host: 'localhost', port: 6382 }
  ];

  // 排序函数（从 popup.js 提取的逻辑）
  function sortByHistory(servers, recentServers) {
    return [...servers].sort((a, b) => {
      const aIndex = recentServers.indexOf(a.name);
      const bIndex = recentServers.indexOf(b.name);
      if (aIndex >= 0 && bIndex >= 0) return aIndex - bIndex;
      if (aIndex >= 0) return -1;
      if (bIndex >= 0) return 1;
      return 0;
    });
  }

  test('should put recent servers first', () => {
    const recentServers = ['gamma', 'alpha'];
    const sorted = sortByHistory(mockServers, recentServers);
    
    expect(sorted[0].name).toBe('gamma');
    expect(sorted[1].name).toBe('alpha');
  });

  test('should maintain order of recent servers', () => {
    const recentServers = ['delta', 'beta', 'alpha'];
    const sorted = sortByHistory(mockServers, recentServers);
    
    expect(sorted[0].name).toBe('delta');
    expect(sorted[1].name).toBe('beta');
    expect(sorted[2].name).toBe('alpha');
  });

  test('should keep non-recent servers in original order', () => {
    const recentServers = ['gamma'];
    const sorted = sortByHistory(mockServers, recentServers);
    
    expect(sorted[0].name).toBe('gamma');
    // 其他服务器保持原顺序
    const nonRecent = sorted.slice(1);
    expect(nonRecent[0].name).toBe('alpha');
    expect(nonRecent[1].name).toBe('beta');
    expect(nonRecent[2].name).toBe('delta');
  });

  test('should handle empty history', () => {
    const sorted = sortByHistory(mockServers, []);
    expect(sorted.map(s => s.name)).toEqual(['alpha', 'beta', 'gamma', 'delta']);
  });
});

// 结果格式化测试
describe('Result Formatting', () => {
  function formatResult(result) {
    if (result === null) {
      return { type: 'nil', display: '(nil)' };
    }
    if (typeof result === 'string') {
      return { type: 'string', display: `"${result}"` };
    }
    if (typeof result === 'number') {
      return { type: 'number', display: `(integer) ${result}` };
    }
    if (Array.isArray(result)) {
      return { type: 'array', display: result.map((item, i) => `${i + 1}) ${item}`).join('\n') };
    }
    return { type: 'unknown', display: String(result) };
  }

  test('should format null as (nil)', () => {
    const result = formatResult(null);
    expect(result.type).toBe('nil');
    expect(result.display).toBe('(nil)');
  });

  test('should format string with quotes', () => {
    const result = formatResult('hello world');
    expect(result.type).toBe('string');
    expect(result.display).toBe('"hello world"');
  });

  test('should format number as integer', () => {
    const result = formatResult(42);
    expect(result.type).toBe('number');
    expect(result.display).toBe('(integer) 42');
  });

  test('should format array with indices', () => {
    const result = formatResult(['a', 'b', 'c']);
    expect(result.type).toBe('array');
    expect(result.display).toContain('1) a');
    expect(result.display).toContain('2) b');
    expect(result.display).toContain('3) c');
  });
});

// HTML 转义测试
describe('HTML Escaping', () => {
  function escapeHtml(text) {
    const div = { textContent: '', innerHTML: '' };
    div.textContent = text;
    // 模拟浏览器行为
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  test('should escape HTML special characters', () => {
    const result = escapeHtml('<script>alert("xss")</script>');
    expect(result).not.toContain('<script>');
    expect(result).toContain('&lt;script&gt;');
  });

  test('should escape ampersand', () => {
    const result = escapeHtml('a & b');
    expect(result).toBe('a &amp; b');
  });

  test('should escape quotes', () => {
    const result = escapeHtml('say "hello"');
    expect(result).toContain('&quot;');
  });
});
