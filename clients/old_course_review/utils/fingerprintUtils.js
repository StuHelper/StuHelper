/**
 * 浏览器指纹工具
 * 用于生成和管理浏览器指纹，以便识别用户设备
 */

// 存储指纹的本地存储键
const FINGERPRINT_STORAGE_KEY = 'browser_fingerprint';

/**
 * 生成简单的浏览器指纹
 * 基于用户代理、屏幕分辨率、时区等信息
 * @returns {string} 浏览器指纹哈希值
 */
const generateSimpleFingerprint = () => {
  const components = [
    navigator.userAgent,
    window.screen.width + 'x' + window.screen.height,
    window.screen.colorDepth,
    new Date().getTimezoneOffset(),
    navigator.language || navigator.userLanguage,
    navigator.platform,
    !!navigator.cookieEnabled,
    !!window.localStorage,
    !!window.sessionStorage,
    !!window.indexedDB
  ];

  // 将组件连接成字符串并生成哈希
  const fingerprintString = components.join('###');
  return hashString(fingerprintString);
};

/**
 * 简单的字符串哈希函数
 * @param {string} str - 要哈希的字符串
 * @returns {string} 哈希值
 */
const hashString = (str) => {
  let hash = 0;
  if (str.length === 0) return hash.toString(16);
  
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // 转换为32位整数
  }
  
  // 添加随机盐值使哈希更难预测
  const salt = Math.random().toString(36).substring(2, 15);
  const saltedHash = hash.toString(16) + salt;
  
  return saltedHash;
};

/**
 * 获取或生成浏览器指纹
 * 如果本地存储中已有指纹，则返回已有的；否则生成新的并存储
 * @returns {string} 浏览器指纹
 */
export const getBrowserFingerprint = () => {
  try {
    // 尝试从本地存储获取指纹
    const storedFingerprint = localStorage.getItem(FINGERPRINT_STORAGE_KEY);
    
    if (storedFingerprint) {
      return storedFingerprint;
    }
    
    // 如果没有存储的指纹，生成新的
    const newFingerprint = generateSimpleFingerprint();
    localStorage.setItem(FINGERPRINT_STORAGE_KEY, newFingerprint);
    
    return newFingerprint;
  } catch (error) {
    console.error('获取浏览器指纹失败:', error);
    // 如果出错（例如禁用了localStorage），则每次生成新的指纹
    return generateSimpleFingerprint();
  }
};

export default {
  getBrowserFingerprint
}; 