/**
 * 缓存服务 - 用于管理课程数据的本地存储和内存缓存
 */

// 缓存键定义
export const CACHE_KEYS = {
  ALL_COURSES: 'ALL_COURSES',
  COURSE_DETAILS: 'COURSE_DETAILS_',
  DEPARTMENTS: 'DEPARTMENTS',
  REVIEW_COUNT: 'REVIEW_COUNT',
  LATEST_REVIEWS: 'LATEST_REVIEWS',
  COURSE_REVIEWS: 'COURSE_REVIEWS_',
  REVIEW_DETAILS: 'REVIEW_DETAILS_',
  SEARCH_RESULTS: 'SEARCH_RESULTS_',
  SESSION_ID: 'SESSION_ID' // 新增：会话ID键
};

// 缓存过期时间（毫秒）
export const CACHE_EXPIRATION = {
  DEFAULT: 5 * 60 * 1000, // 默认5分钟
  COURSES: 30 * 60 * 1000, // 课程列表30分钟
  COURSE_DETAILS: 15 * 60 * 1000, // 课程详情15分钟
  DEPARTMENTS: 60 * 60 * 1000, // 院系列表1小时
  REVIEWS: 5 * 60 * 1000, // 评论5分钟
  SEARCH: 5 * 60 * 1000 // 搜索结果5分钟
};

// 内存缓存
const memoryCache = new Map();

// 生成会话ID
const generateSessionId = () => {
  return Date.now().toString(36) + Math.random().toString(36).substring(2);
};

// 获取或创建会话ID
const getOrCreateSessionId = () => {
  try {
    // 尝试从sessionStorage获取会话ID
    let sessionId = sessionStorage.getItem(CACHE_KEYS.SESSION_ID);
    
    // 如果不存在，创建新的会话ID
    if (!sessionId) {
      sessionId = generateSessionId();
      sessionStorage.setItem(CACHE_KEYS.SESSION_ID, sessionId);
      console.log('创建新会话ID:', sessionId);
    }
    
    return sessionId;
  } catch (error) {
    console.error('获取或创建会话ID失败:', error);
    return generateSessionId(); // 出错时返回新生成的ID
  }
};

// 检测页面刷新
let lastNavigationType = '';
let manualRefreshTriggered = false;
let currentSessionId = getOrCreateSessionId();

try {
  // 获取页面加载性能信息
  const perfEntries = performance.getEntriesByType('navigation');
  if (perfEntries && perfEntries.length > 0) {
    lastNavigationType = perfEntries[0].type;
    
    // 如果是页面刷新，清除内存缓存
    if (lastNavigationType === 'reload') {
      console.log('检测到页面刷新，清除内存缓存');
      memoryCache.clear();
      
      // 刷新时重新生成会话ID
      currentSessionId = generateSessionId();
      sessionStorage.setItem(CACHE_KEYS.SESSION_ID, currentSessionId);
      console.log('页面刷新，创建新会话ID:', currentSessionId);
    }
  }
} catch (error) {
  console.error('获取页面导航类型失败:', error);
}

/**
 * 检查是否是页面刷新
 * @returns {boolean} 是否是页面刷新
 */
export const isPageRefresh = () => {
  // 只有在真正的页面刷新或手动触发刷新时返回true
  return lastNavigationType === 'reload' || manualRefreshTriggered;
};

/**
 * 获取当前会话ID
 * @returns {string} 当前会话ID
 */
export const getSessionId = () => {
  return currentSessionId;
};

/**
 * 手动触发刷新缓存
 * @param {boolean} [reset=true] - 触发后是否重置标志
 */
export const forceRefreshCache = (reset = true) => {
  manualRefreshTriggered = true;
  
  // 清除内存缓存
  memoryCache.clear();
  
  // 重新生成会话ID
  currentSessionId = generateSessionId();
  sessionStorage.setItem(CACHE_KEYS.SESSION_ID, currentSessionId);
  console.log('手动刷新，创建新会话ID:', currentSessionId);
  
  // 如果需要，在操作后重置标志
  if (reset) {
    setTimeout(() => {
      manualRefreshTriggered = false;
    }, 100);
  }
  
  return true;
};

/**
 * 检查本地存储是否可用
 * @returns {boolean} 本地存储是否可用
 */
const isLocalStorageAvailable = () => {
  try {
    const testKey = '__storage_test__';
    localStorage.setItem(testKey, testKey);
    localStorage.removeItem(testKey);
    return true;
  } catch (e) {
    return false;
  }
};

/**
 * 设置缓存
 * @param {string} key - 缓存键
 * @param {*} data - 要缓存的数据
 * @param {number} [expirationTime] - 过期时间（毫秒），不设置则使用默认值
 */
export const setCache = (key, data, expirationTime) => {
  try {
    if (!key) return;
    
    // 确定过期时间
    const expiration = expirationTime || CACHE_EXPIRATION.DEFAULT;
    const expiresAt = Date.now() + expiration;
    
    // 创建缓存项，添加会话ID
    const cacheItem = {
      data,
      expiresAt,
      sessionId: currentSessionId
    };
    
    // 存储到内存缓存
    memoryCache.set(key, cacheItem);
    
    // 如果本地存储可用，也存储到本地存储
    if (isLocalStorageAvailable()) {
      localStorage.setItem(key, JSON.stringify(cacheItem));
    }
  } catch (error) {
    console.error('缓存设置失败:', error);
  }
};

/**
 * 获取缓存
 * @param {string} key - 缓存键
 * @param {boolean} [ignoreSession=false] - 是否忽略会话检查
 * @returns {*} 缓存的数据，如果不存在或已过期则返回null
 */
export const getCache = (key, ignoreSession = false) => {
  try {
    if (!key) return null;
    
    // 首先检查内存缓存
    let cacheItem = memoryCache.get(key);
    
    // 如果内存中没有，但本地存储可用，则尝试从本地存储获取
    if (!cacheItem && isLocalStorageAvailable()) {
      const storedItem = localStorage.getItem(key);
      if (storedItem) {
        try {
          cacheItem = JSON.parse(storedItem);
          // 将从本地存储恢复的项目也放入内存缓存
          memoryCache.set(key, cacheItem);
        } catch (e) {
          // 解析错误，删除无效的本地存储项
          localStorage.removeItem(key);
          return null;
        }
      }
    }
    
    // 如果没有找到缓存项
    if (!cacheItem) return null;
    
    // 检查是否过期
    if (cacheItem.expiresAt && cacheItem.expiresAt < Date.now()) {
      // 已过期，清除缓存
      memoryCache.delete(key);
      if (isLocalStorageAvailable()) {
        localStorage.removeItem(key);
      }
      return null;
    }
    
    // 检查会话ID是否匹配（除非忽略会话检查）
    if (!ignoreSession && cacheItem.sessionId && cacheItem.sessionId !== currentSessionId) {
      console.log(`缓存项会话ID不匹配，当前: ${currentSessionId}, 缓存: ${cacheItem.sessionId}`);
      return null;
    }
    
    // 返回缓存数据
    return cacheItem.data;
  } catch (error) {
    console.error('获取缓存失败:', error);
    return null;
  }
};

/**
 * 检查是否存在有效的缓存
 * @param {string} key - 缓存键
 * @param {boolean} [ignoreSession=false] - 是否忽略会话检查
 * @returns {boolean} 是否存在有效的缓存
 */
export const hasCache = (key, ignoreSession = false) => {
  try {
    if (!key) return false;
    
    // 首先检查内存缓存
    let cacheItem = memoryCache.get(key);
    
    // 如果内存中没有，但本地存储可用，则尝试从本地存储获取
    if (!cacheItem && isLocalStorageAvailable()) {
      const storedItem = localStorage.getItem(key);
      if (storedItem) {
        try {
          cacheItem = JSON.parse(storedItem);
          // 将从本地存储恢复的项目也放入内存缓存
          memoryCache.set(key, cacheItem);
        } catch (e) {
          // 解析错误，删除无效的本地存储项
          localStorage.removeItem(key);
          return false;
        }
      }
    }
    
    // 如果没有找到缓存项
    if (!cacheItem) return false;
    
    // 检查是否过期
    if (cacheItem.expiresAt && cacheItem.expiresAt < Date.now()) {
      // 已过期，清除缓存
      memoryCache.delete(key);
      if (isLocalStorageAvailable()) {
        localStorage.removeItem(key);
      }
      return false;
    }
    
    // 检查会话ID是否匹配（除非忽略会话检查）
    if (!ignoreSession && cacheItem.sessionId && cacheItem.sessionId !== currentSessionId) {
      console.log(`缓存项会话ID不匹配，当前: ${currentSessionId}, 缓存: ${cacheItem.sessionId}`);
      return false;
    }
    
    // 存在有效缓存
    return true;
  } catch (error) {
    console.error('检查缓存失败:', error);
    return false;
  }
};

/**
 * 清除缓存
 * @param {string} [key] - 要清除的缓存键，不提供则清除所有缓存
 */
export const clearCache = (key) => {
  try {
    if (key) {
      // 清除特定键的缓存
      memoryCache.delete(key);
      
      // 如果是前缀匹配，清除所有匹配的键
      if (key.endsWith('_')) {
        // 清除内存缓存中匹配的键
        for (const cacheKey of memoryCache.keys()) {
          if (cacheKey.startsWith(key)) {
            memoryCache.delete(cacheKey);
          }
        }
        
        // 清除本地存储中匹配的键
        if (isLocalStorageAvailable()) {
          for (let i = 0; i < localStorage.length; i++) {
            const storageKey = localStorage.key(i);
            if (storageKey && storageKey.startsWith(key)) {
              localStorage.removeItem(storageKey);
            }
          }
        }
      } else if (isLocalStorageAvailable()) {
        // 清除本地存储中的特定键
        localStorage.removeItem(key);
      }
    } else {
      // 清除所有缓存
      memoryCache.clear();
      if (isLocalStorageAvailable()) {
        localStorage.clear();
      }
    }
  } catch (error) {
    console.error('清除缓存失败:', error);
  }
};

// 默认导出
export default {
  CACHE_KEYS,
  CACHE_EXPIRATION,
  setCache,
  getCache,
  hasCache,
  clearCache,
  isPageRefresh,
  forceRefreshCache,
  getSessionId
}; 