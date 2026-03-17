import axios from 'axios';
import api, { handleApiError } from './api';
import cacheService from './cacheService';
import { pinyin } from 'pinyin-pro';

const { CACHE_KEYS, CACHE_EXPIRATION, setCache, getCache, hasCache, clearCache, isPageRefresh } = cacheService;

/**
 * 为课程添加拼音属性
 * @param {Object} course - 课程对象
 * @returns {Object} - 添加了拼音属性的课程对象
 */
const addPinyinToCourse = (course) => {
  if (!course || !course.name) return course;
  
  try {
    const fullPinyin = pinyin(course.name, { toneType: 'none', type: 'array' }).join('').toLowerCase();
    const firstLetters = pinyin(course.name, { pattern: 'first', toneType: 'none', type: 'array' }).join('').toLowerCase();
    const pinyinParts = pinyin(course.name, { toneType: 'none', type: 'array' });
    
    return {
      ...course,
      pinyin: fullPinyin,
      firstLetters,
      pinyinParts
    };
  } catch (error) {
    console.error('添加拼音属性失败:', error);
    return course;
  }
};

/**
 * 获取所有课程
 * @param {Object} options - 选项对象
 * @param {boolean} options.forceRefresh - 是否强制刷新缓存
 * @returns {Promise<Array>} - 课程数组
 */
export const getAllCourses = async (options = {}) => {
  try {
    const { forceRefresh = false } = options;
    
    // 检查是否有缓存且不需要强制刷新
    // 注意：现在isPageRefresh只在真正的页面刷新或手动触发时返回true
    if (!forceRefresh && !isPageRefresh()) {
      const cachedCourses = getCache(CACHE_KEYS.ALL_COURSES);
      if (cachedCourses) {
        console.log('使用缓存的课程列表');
        return cachedCourses;
      }
    }
    
    // 没有缓存或需要强制刷新，从API获取
    console.log('从API获取课程列表');
    const response = await api.get('/courses');
    const courses = response.data || [];
    
    // 添加拼音属性
    const coursesWithPinyin = courses.map(addPinyinToCourse);
    
    // 缓存数据
    setCache(CACHE_KEYS.ALL_COURSES, coursesWithPinyin, CACHE_EXPIRATION.COURSES);
    
    return coursesWithPinyin;
  } catch (error) {
    console.error('获取课程列表失败:', error);
    // 如果有缓存，在请求失败时返回缓存数据
    const cachedCourses = getCache(CACHE_KEYS.ALL_COURSES);
    if (cachedCourses) {
      console.log('请求失败，使用缓存数据');
      return cachedCourses;
    }
    throw handleApiError(error, '获取课程列表失败，请稍后重试');
  }
};

/**
 * 获取评价总数，优先使用缓存
 * @param {boolean} forceRefresh - 是否强制刷新缓存
 * @returns {Promise<number>} - 评价总数
 */
export const getReviewCount = async (forceRefresh = false) => {
  try {
    // 检查是否有缓存且不需要强制刷新
    if (!forceRefresh) {
      const cachedCount = getCache(CACHE_KEYS.REVIEW_COUNT);
      if (cachedCount !== null) {
        console.log('使用缓存的评价总数');
        return cachedCount;
      }
    }
    
    // 没有缓存或需要强制刷新，从API获取
    console.log('从API获取评价总数');
    const response = await api.get('/reviews');
    const count = response.data.pagination?.total || 0;
    
    // 缓存数据
    setCache(CACHE_KEYS.REVIEW_COUNT, count);
    
    return count;
  } catch (error) {
    console.error('获取评价总数失败:', error);
    // 如果有缓存，在请求失败时返回缓存数据
    const cachedCount = getCache(CACHE_KEYS.REVIEW_COUNT);
    if (cachedCount !== null) {
      console.log('请求失败，使用缓存数据');
      return cachedCount;
    }
    return 0;
  }
};

/**
 * 搜索课程
 * @param {Object} searchParams - 搜索参数对象
 * @param {boolean} useCache - 是否使用缓存
 * @returns {Promise<Object>} - 包含搜索结果和错误信息的对象
 */
export const searchCourses = async (searchParams, useCache = true) => {
  try {
    // 如果搜索参数为空或只有空字符串，返回所有课程
    const isEmptySearch = !searchParams || 
      (Object.keys(searchParams).length === 0) || 
      (Object.keys(searchParams).length === 1 && searchParams.courseName === '');
    
    if (isEmptySearch) {
      const courses = await getAllCourses();
      return { 
        courses: courses.slice(0, 20), // 只返回前20个结果
        total: courses.length,
        error: null 
      };
    }
    
    // 构建查询参数
    const queryParams = new URLSearchParams();
    
    if (searchParams.courseName) {
      queryParams.append('query', searchParams.courseName);
    }
    
    if (searchParams.department) {
      queryParams.append('department', searchParams.department);
    }
    
    if (searchParams.type) {
      queryParams.append('type', searchParams.type);
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
    const cacheKey = `SEARCH_${queryParams.toString()}`;
    
    // 检查缓存
    if (useCache) {
      const cachedResults = getCache(cacheKey);
      if (cachedResults) {
        console.log('使用缓存的搜索结果');
        return cachedResults;
      }
    }
    
    // 从API获取
    console.log('从API获取搜索结果');
    const response = await api.get(`/search/courses?${queryParams.toString()}`);
    
    const result = {
      courses: response.data.courses || [],
      total: response.data.total || 0,
      error: null
    };
    
    // 缓存结果，但使用较短的过期时间
    setCache(cacheKey, result, 5 * 60 * 1000); // 5分钟过期
    
    return result;
  } catch (error) {
    console.error('搜索课程失败:', error);
    return { 
      courses: [], 
      total: 0,
      error: handleApiError(error, '搜索课程失败，请稍后重试') 
    };
  }
};

/**
 * 获取课程详情
 * @param {number} courseId - 课程ID
 * @param {Object} options - 选项对象
 * @param {boolean} options.forceRefresh - 是否强制刷新缓存
 * @returns {Promise<Object>} - 课程详情对象
 */
export const getCourseDetails = async (courseId, options = {}) => {
  try {
    const { forceRefresh = false } = options;
    const cacheKey = `${CACHE_KEYS.COURSE_DETAILS}${courseId}`;
    
    // 检查是否有缓存且不需要强制刷新
    // 注意：现在isPageRefresh只在真正的页面刷新或手动触发时返回true
    if (!forceRefresh && !isPageRefresh()) {
      const cachedCourse = getCache(cacheKey);
      if (cachedCourse) {
        console.log(`使用缓存的课程详情 (ID: ${courseId})`);
        return cachedCourse;
      }
    }
    
    // 没有缓存或需要强制刷新，从API获取
    console.log(`从API获取课程详情 (ID: ${courseId})`);
    const response = await api.get(`/courses/${courseId}`);
    const course = response.data;
    
    // 缓存数据
    setCache(cacheKey, course, CACHE_EXPIRATION.COURSE_DETAILS);
    
    return course;
  } catch (error) {
    console.error(`获取课程详情失败 (ID: ${courseId}):`, error);
    // 如果有缓存，在请求失败时返回缓存数据
    const cacheKey = `${CACHE_KEYS.COURSE_DETAILS}${courseId}`;
    const cachedCourse = getCache(cacheKey);
    if (cachedCourse) {
      console.log(`请求失败，使用缓存的课程详情 (ID: ${courseId})`);
      return cachedCourse;
    }
    throw handleApiError(error, '获取课程详情失败，请稍后重试');
  }
};

/**
 * 批量获取课程详情
 * @param {number[]} courseIds - 课程ID数组
 * @param {Object} options - 选项对象
 * @param {boolean} options.forceRefresh - 是否强制刷新缓存
 * @returns {Promise<Object>} - 课程详情映射对象
 */
export const batchGetCourseDetails = async (courseIds, options = {}) => {
  if (!courseIds || courseIds.length === 0) {
    return { courses: {}, errors: {} };
  }
  
  const { forceRefresh = false } = options;
  
  // 使用Promise.all并行获取所有课程详情
  const results = await Promise.all(
    courseIds.map(id => getCourseDetails(id, { forceRefresh }).catch(error => ({ error })))
  );
  
  // 整理结果
  const courses = {};
  const errors = {};
  
  results.forEach((result, index) => {
    const courseId = courseIds[index];
    if (result.error) {
      errors[courseId] = result.error;
    } else {
      courses[courseId] = result;
    }
  });
  
  return { courses, errors };
};

/**
 * 获取所有院系
 * @param {Object} options - 选项对象
 * @param {boolean} options.forceRefresh - 是否强制刷新缓存
 * @returns {Promise<Array>} - 院系数组
 */
export const getAllDepartments = async (options = {}) => {
  try {
    const { forceRefresh = false } = options;
    
    // 检查是否有缓存且不需要强制刷新
    // 注意：现在isPageRefresh只在真正的页面刷新或手动触发时返回true
    if (!forceRefresh && !isPageRefresh()) {
      const cachedDepartments = getCache(CACHE_KEYS.DEPARTMENTS);
      if (cachedDepartments) {
        console.log('使用缓存的院系列表');
        return cachedDepartments;
      }
    }
    
    // 没有缓存或需要强制刷新，从API获取
    console.log('从API获取院系列表');
    const response = await api.get('/departments');
    const departments = response.data || [];
    
    // 缓存数据
    setCache(CACHE_KEYS.DEPARTMENTS, departments, CACHE_EXPIRATION.DEPARTMENTS);
    
    return departments;
  } catch (error) {
    console.error('获取院系列表失败:', error);
    // 如果有缓存，在请求失败时返回缓存数据
    const cachedDepartments = getCache(CACHE_KEYS.DEPARTMENTS);
    if (cachedDepartments) {
      console.log('请求失败，使用缓存数据');
      return cachedDepartments;
    }
    throw handleApiError(error, '获取院系列表失败，请稍后重试');
  }
};

/**
 * 获取所有学期
 * @returns {Array} - 学期数组
 */
export const getSemesters = () => {
  const currentYear = new Date().getFullYear();
  const semesters = [];
  
  // 生成最近5年的学期
  for (let year = currentYear; year >= currentYear - 4; year--) {
    semesters.push(`${year}-${year + 1} 第一学期`);
    semesters.push(`${year - 1}-${year} 第二学期`);
  }
  
  return semesters;
};

// 默认导出所有方法
export default {
  getAllCourses,
  getReviewCount,
  searchCourses,
  getCourseDetails,
  batchGetCourseDetails,
  getAllDepartments,
  getSemesters
}; 