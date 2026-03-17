/**
 * 增强版安全工具函数
 * 包含水印、禁用复制粘贴、右键菜单和开发者工具
 */

/**
 * 禁用右键菜单
 */
export const disableRightClick = () => {
  document.addEventListener('contextmenu', (e) => {
    e.preventDefault();
    return false;
  });
};

/**
 * 禁用复制功能
 */
export const disableCopyPaste = () => {
  // 禁用 Ctrl+C, Ctrl+V, Ctrl+X, Ctrl+S
  document.addEventListener('keydown', (e) => {
    if (
      (e.ctrlKey && (e.key === 'c' || e.key === 'C' || 
                      e.key === 'v' || e.key === 'V' || 
                      e.key === 'x' || e.key === 'X' || 
                      e.key === 's' || e.key === 'S'))
    ) {
      e.preventDefault();
      return false;
    }
  });

  // 禁用选择和复制
  document.addEventListener('selectstart', (e) => {
    // 允许在输入框中选择文本
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
      return true;
    }
    e.preventDefault();
    return false;
  });

  document.addEventListener('copy', (e) => {
    // 允许在输入框中复制文本
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
      return true;
    }
    e.preventDefault();
    return false;
  });
};

/**
 * 检测并禁用开发者工具
 */
export const disableDeveloperTools = () => {
  // 监听按键，禁用F12
  document.addEventListener('keydown', (e) => {
    if (e.key === 'F12' || (e.ctrlKey && e.shiftKey && (e.key === 'i' || e.key === 'I' || e.key === 'j' || e.key === 'J'))) {
      e.preventDefault();
      return false;
    }
  });

  // 检测开发者工具是否打开
  const checkDevTools = () => {
    const widthThreshold = window.outerWidth - window.innerWidth > 160;
    const heightThreshold = window.outerHeight - window.innerHeight > 160;
    
    if (widthThreshold || heightThreshold) {
      // 开发者工具可能打开，可以采取措施
      document.body.innerHTML = '网站安全保护中，请关闭开发者工具';
    }
  };

  setInterval(checkDevTools, 1000);

  // 监听 devtool 事件
  window.addEventListener('devtoolschange', (e) => {
    if (e.detail.isOpen) {
      document.body.innerHTML = '网站安全保护中，请关闭开发者工具';
    }
  });
};

/**
 * 添加水印
 */
export const addWatermark = () => {
  const watermark = document.createElement('div');
  watermark.style.position = 'fixed';
  watermark.style.top = '0';
  watermark.style.left = '0';
  watermark.style.width = '100%';
  watermark.style.height = '100%';
  watermark.style.pointerEvents = 'none';
  watermark.style.background = 'url("data:image/svg+xml;utf8,<svg xmlns=\'http://www.w3.org/2000/svg\' width=\'200\' height=\'200\'><text x=\'20\' y=\'100\' transform=\'rotate(-45 100 100)\' fill=\'rgba(0,0,0,0.05)\' font-size=\'20\'>stuhelper.com</text></svg>")';
  watermark.style.zIndex = '9999';

  // 确保水印不能被删除
  const observer = new MutationObserver(function() {
    if (!document.body.contains(watermark)) {
      document.body.appendChild(watermark);
    }
  });

  // 开始观察文档变化
  observer.observe(document.body, {
    childList: true,
    subtree: true
  });

  document.body.appendChild(watermark);
};

/**
 * 添加安全标识符
 */
export const addSecurityIdentifier = () => {
  // 将安全标识符添加到 localStorage
  localStorage.setItem('securityVersion', 'production-v1.0');
  
  // 定期检查安全标识符是否被移除
  setInterval(() => {
    if (localStorage.getItem('securityVersion') !== 'production-v1.0') {
      localStorage.setItem('securityVersion', 'production-v1.0');
    }
  }, 2000);
};

/**
 * 使用CSS风格禁用复制
 */
export const addDisableCopyCSS = () => {
  const style = document.createElement('style');
  style.innerHTML = `
    body {
      -webkit-touch-callout: none;
      -webkit-user-select: none;
      -khtml-user-select: none;
      -moz-user-select: none;
      -ms-user-select: none;
      user-select: none;
    }
    
    input, textarea {
      -webkit-touch-callout: default;
      -webkit-user-select: text;
      -khtml-user-select: text;
      -moz-user-select: text;
      -ms-user-select: text;
      user-select: text;
    }
  `;
  document.head.appendChild(style);
};

/**
 * 添加防护伪造输入框
 * 用户可能会尝试在控制台创建假的登录表单来存储密码
 */
export const protectFakeInputs = () => {
  // 定期检查是否存在可疑的输入元素
  setInterval(() => {
    const inputs = document.querySelectorAll('input[type="password"]');
    inputs.forEach(input => {
      // 检查这个输入框是否是我们应用创建的
      if (!input.hasAttribute('data-security-verified')) {
        // 可能是恶意脚本注入的，移除它
        input.remove();
      }
    });
  }, 2000);
};

/**
 * 防止iframe嵌入
 */
export const preventFraming = () => {
  if (window.self !== window.top) {
    // 如果当前网页被嵌入iframe中，重定向到真实网站
    window.top.location.href = window.self.location.href;
  }
};

/**
 * 应用所有增强安全措施
 */
export const applyEnhancedSecurity = () => {
  // 仅在生产环境中应用
  if (process.env.NODE_ENV === 'production' || process.env.REACT_APP_ENV === 'production') {
    disableRightClick();
    disableCopyPaste();
    disableDeveloperTools();
    addSecurityIdentifier();
    addDisableCopyCSS();
    protectFakeInputs();
    preventFraming();
    
    // 确保 DOM 加载完成后添加水印
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', addWatermark);
    } else {
      addWatermark();
    }
    
    // 添加额外的调试检测
    window.addEventListener('resize', () => {
      const widthThreshold = window.outerWidth - window.innerWidth > 160;
      const heightThreshold = window.outerHeight - window.innerHeight > 160;
      
      if (widthThreshold || heightThreshold) {
        document.body.innerHTML = '网站安全保护中，请关闭开发者工具';
      }
    });
  }
}; 