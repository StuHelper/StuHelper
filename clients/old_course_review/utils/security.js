/**
 * 安全工具函数 - 防止复制、禁用右键以及开发者工具
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
    // 检查是否为移动设备
    const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
    
    // 在移动设备上使用不同的检测标准
    if (isMobile) {
      // 移动设备上的检测更宽松，避免误判
      const widthThreshold = window.outerWidth - window.innerWidth > 200;
      const heightThreshold = window.outerHeight - window.innerHeight > 300;
      
      if (widthThreshold && heightThreshold) {
        // 同时满足宽度和高度条件才认为是开发者工具
        document.body.innerHTML = '网站安全保护中，请关闭开发者工具';
      }
    } else {
      // 桌面设备上的标准检测
      const widthThreshold = window.outerWidth - window.innerWidth > 160;
      const heightThreshold = window.outerHeight - window.innerHeight > 160;
      
      if (widthThreshold || heightThreshold) {
        // 桌面设备上满足任一条件即可
        document.body.innerHTML = '网站安全保护中，请关闭开发者工具';
      }
    }
  };

  setInterval(checkDevTools, 1000);

  // 监听 devtool 事件
  window.addEventListener('devtoolschange', (e) => {
    // 检查是否为移动设备，iOS 设备上更容易误触发
    const isIOS = /iPhone|iPad|iPod/i.test(navigator.userAgent);
    
    if (e.detail.isOpen && !isIOS) {
      document.body.innerHTML = '网站安全保护中，请关闭开发者工具';
    }
  });
};

/**
 * 应用所有安全措施
 */
export const applyAllSecurityMeasures = () => {
  // 仅在生产环境中应用
  if (process.env.NODE_ENV === 'production' || process.env.REACT_APP_ENV === 'production') {
    disableRightClick();
    disableCopyPaste();
    disableDeveloperTools();
    
    // 添加额外的调试检测
    window.addEventListener('resize', () => {
      checkDevTools();
    });
  }
};

// 用于检查开发者工具是否打开
const checkDevTools = () => {
  // 检查是否为移动设备
  const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
  
  if (isMobile) {
    // 移动设备上的检测需要更严格的条件组合，避免误判
    const widthThreshold = window.outerWidth - window.innerWidth > 200;
    const heightThreshold = window.outerHeight - window.innerHeight > 300;
    
    if (widthThreshold && heightThreshold) {
      document.body.innerHTML = '网站安全保护中，请关闭开发者工具';
    }
  } else {
    // 桌面设备上的检测
    const widthThreshold = window.outerWidth - window.innerWidth > 160;
    const heightThreshold = window.outerHeight - window.innerHeight > 160;
    
    if (widthThreshold || heightThreshold) {
      document.body.innerHTML = '网站安全保护中，请关闭开发者工具';
    }
  }
}; 