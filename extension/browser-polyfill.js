// ========================================
// Browser Polyfill for Cross-Browser Compatibility
// 简化版 polyfill，处理 Chrome/Firefox/Edge API 差异
// ========================================

(function() {
  'use strict';

  // 如果 browser 对象已存在（Firefox），直接使用
  if (typeof globalThis.browser !== 'undefined') {
    return;
  }

  // Chrome 使用 chrome 对象，需要包装为 Promise 风格
  const chrome = globalThis.chrome;

  if (typeof chrome === 'undefined') {
    throw new Error('This script must be run in a browser extension context');
  }

  // 简单的 Promise 包装器
  function wrapAsyncFunction(fn, getReceiverFn) {
    return function(...args) {
      return new Promise((resolve, reject) => {
        const callback = (result) => {
          if (chrome.runtime.lastError) {
            reject(new Error(chrome.runtime.lastError.message));
          } else {
            resolve(result);
          }
        };
        
        const receiver = getReceiverFn ? getReceiverFn() : this;
        fn.apply(receiver, [...args, callback]);
      });
    };
  }

  // 创建 browser 对象
  const browser = {
    runtime: {
      ...chrome.runtime,
      sendMessage: wrapAsyncFunction(chrome.runtime.sendMessage, () => chrome.runtime),
      connectNative: chrome.runtime.connectNative.bind(chrome.runtime),
      onMessage: chrome.runtime.onMessage,
      lastError: chrome.runtime.lastError
    },
    storage: {
      local: {
        get: wrapAsyncFunction(chrome.storage.local.get, () => chrome.storage.local),
        set: wrapAsyncFunction(chrome.storage.local.set, () => chrome.storage.local),
        remove: wrapAsyncFunction(chrome.storage.local.remove, () => chrome.storage.local)
      },
      sync: {
        get: wrapAsyncFunction(chrome.storage.sync.get, () => chrome.storage.sync),
        set: wrapAsyncFunction(chrome.storage.sync.set, () => chrome.storage.sync),
        remove: wrapAsyncFunction(chrome.storage.sync.remove, () => chrome.storage.sync)
      }
    }
  };

  globalThis.browser = browser;
})();
