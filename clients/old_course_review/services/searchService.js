import api, { handleApiError } from './api';
import cacheService from './cacheService';
import { pinyin } from 'pinyin-pro';

const { CACHE_KEYS, CACHE_EXPIRATION, setCache, getCache, hasCache, clearCache, isPageRefresh } = cacheService;

/**
 * 搜索课程
 * @param {Object} searchParams - 搜索参数
 * @param {boolean} useCache - 是否使用缓存
 * @returns {Promise<Object>} - 搜索结果
 */
export const searchCourses = async (searchParams = {}, useCache = true) => {
  try {
    // 构建查询参数
    const queryParams = new URLSearchParams();
    
    if (searchParams.query) {
      queryParams.append('query', searchParams.query);
    }
    
    if (searchParams.department) {
      queryParams.append('department', searchParams.department);
    }
    
    if (searchParams.level) {
      queryParams.append('level', searchParams.level);
    }
    
    if (searchParams.page) {
      queryParams.append('page', searchParams.page);
    }
    
    if (searchParams.limit) {
      queryParams.append('limit', searchParams.limit);
    }
    
    if (searchParams.sort) {
      queryParams.append('sort', searchParams.sort);
    }
    
    if (searchParams.order) {
      queryParams.append('order', searchParams.order);
    }
    
    // 缓存键
    const cacheKey = `${CACHE_KEYS.SEARCH_RESULTS}COURSES_${queryParams.toString()}`;
    
    // 检查缓存
    // 注意：现在isPageRefresh只在真正的页面刷新或手动触发时返回true
    if (useCache && !isPageRefresh()) {
      const cachedResults = getCache(cacheKey);
      if (cachedResults) {
        console.log('使用缓存的课程搜索结果');
        return cachedResults;
      }
    }
    
    // 从API获取
    console.log('从API获取课程搜索结果');
    const response = await api.get(`/search/courses?${queryParams.toString()}`);
    
    const result = {
      courses: response.data.courses || [],
      total: response.data.total || 0,
      pagination: response.data.pagination || {},
      error: null
    };
    
    // 缓存结果
    setCache(cacheKey, result, CACHE_EXPIRATION.SEARCH);
    
    return result;
  } catch (error) {
    console.error('搜索课程失败:', error);
    return {
      courses: [],
      total: 0,
      pagination: {},
      error: handleApiError(error, '搜索课程失败，请稍后重试')
    };
  }
};

/**
 * 搜索评论
 * @param {Object} searchParams - 搜索参数
 * @param {boolean} useCache - 是否使用缓存
 * @returns {Promise<Object>} - 搜索结果
 */
export const searchReviews = async (searchParams = {}, useCache = true) => {
  try {
    // 构建查询参数
    const queryParams = new URLSearchParams();
    
    if (searchParams.query) {
      queryParams.append('query', searchParams.query);
    }
    
    if (searchParams.courseId) {
      queryParams.append('courseId', searchParams.courseId);
    }
    
    if (searchParams.semester) {
      queryParams.append('semester', searchParams.semester);
    }
    
    if (searchParams.teacher) {
      queryParams.append('teacher', searchParams.teacher);
    }
    
    if (searchParams.minRating) {
      queryParams.append('minRating', searchParams.minRating);
    }
    
    if (searchParams.page) {
      queryParams.append('page', searchParams.page);
    }
    
    if (searchParams.limit) {
      queryParams.append('limit', searchParams.limit);
    }
    
    if (searchParams.sort) {
      queryParams.append('sort', searchParams.sort);
    }
    
    if (searchParams.order) {
      queryParams.append('order', searchParams.order);
    }
    
    // 缓存键
    const cacheKey = `${CACHE_KEYS.SEARCH_RESULTS}REVIEWS_${queryParams.toString()}`;
    
    // 检查缓存
    // 注意：现在isPageRefresh只在真正的页面刷新或手动触发时返回true
    if (useCache && !isPageRefresh()) {
      const cachedResults = getCache(cacheKey);
      if (cachedResults) {
        console.log('使用缓存的评论搜索结果');
        return cachedResults;
      }
    }
    
    // 从API获取
    console.log('从API获取评论搜索结果');
    const response = await api.get(`/search/reviews?${queryParams.toString()}`);
    
    const result = {
      reviews: response.data.reviews || [],
      total: response.data.total || 0,
      pagination: response.data.pagination || {},
      error: null
    };
    
    // 缓存结果
    setCache(cacheKey, result, cacheService.CACHE_EXPIRATION.SEARCH);
    
    return result;
  } catch (error) {
    console.error('搜索评论失败:', error);
    return {
      reviews: [],
      total: 0,
      pagination: {},
      error: handleApiError(error, '搜索评论失败，请稍后重试')
    };
  }
};

/**
 * 本地搜索课程（客户端搜索，用于快速筛选）
 * @param {Array} courses - 课程数组
 * @param {string} query - 搜索关键词
 * @returns {Array} - 过滤后的课程数组
 */
export const localSearchCourses = (courses, query) => {
  if (!courses || !Array.isArray(courses) || courses.length === 0) {
    return [];
  }
  
  if (!query || query.trim() === '') {
    return courses;
  }
  
  const searchTermLower = query.toLowerCase().trim();
  
  // 计算每个课程的匹配分数
  const coursesWithScore = courses.map(course => {
    let score = 0;
    
    // 1. 直接名称匹配
    if (course.name && course.name.toLowerCase().includes(searchTermLower)) {
      score += 100;
    }
    
    // 2. 课程代码匹配
    if (course.code && course.code.toLowerCase().includes(searchTermLower)) {
      score += 80;
    }
    
    // 3. 完整拼音匹配
    if (course.pinyin && course.pinyin.includes(searchTermLower)) {
      score += 70;
    }
    
    // 4. 首字母完全匹配
    if (course.firstLetters && course.firstLetters === searchTermLower) {
      score += 65;
    }
    
    // 5. 首字母包含匹配
    if (course.firstLetters && course.firstLetters.includes(searchTermLower)) {
      score += 60;
    }
    
    // 6. 教师名称匹配
    if (course.teacher && course.teacher.toLowerCase().includes(searchTermLower)) {
      score += 50;
    }
    
    // 7. 院系名称匹配
    if (course.department && course.department.toLowerCase().includes(searchTermLower)) {
      score += 40;
    }
    
    return { ...course, score };
  });
  
  // 过滤掉得分为0的课程，并按得分降序排序
  return coursesWithScore
    .filter(course => course.score > 0)
    .sort((a, b) => b.score - a.score);
};

/**
 * 清除搜索缓存
 */
export const clearSearchCache = () => {
  clearCache(CACHE_KEYS.SEARCH_RESULTS);
};

// 默认导出
export default {
  searchCourses,
  searchReviews,
  localSearchCourses,
  clearSearchCache
}; 