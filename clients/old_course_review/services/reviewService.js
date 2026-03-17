import api, { handleApiError } from './api';
import cacheService from './cacheService';

const { CACHE_KEYS, setCache, getCache, hasCache, clearCache, isPageRefresh } = cacheService;

/**
 * 获取最新评论
 * @param {number} limit - 获取数量
 * @param {boolean} forceRefresh - 是否强制刷新缓存
 * @returns {Promise<Object>} - 包含评论数组和错误信息的对象
 */
export const getLatestReviews = async (limit = 5, forceRefresh = false) => {
  try {
    const cacheKey = `${CACHE_KEYS.LATEST_REVIEWS}_${limit}`;
    
    // 检查是否有缓存且不需要强制刷新
    // 注意：现在isPageRefresh只在真正的页面刷新或手动触发时返回true
    if (!forceRefresh && !isPageRefresh()) {
      const cachedReviews = getCache(cacheKey);
      if (cachedReviews) {
        console.log('使用缓存的最新评论');
        return { reviews: cachedReviews, error: null };
      }
    }
    
    // 没有缓存或需要强制刷新，从API获取
    console.log('从API获取最新评论');
    const response = await api.get(`/reviews/latest?limit=${limit}`);
    const reviews = response.data || [];
    
    // 缓存数据，使用较短的过期时间
    setCache(cacheKey, reviews, 5 * 60 * 1000); // 5分钟过期
    
    return { reviews, error: null };
  } catch (error) {
    console.error('获取最新评论失败:', error);
    // 如果有缓存，在请求失败时返回缓存数据
    const cacheKey = `${CACHE_KEYS.LATEST_REVIEWS}_${limit}`;
    const cachedReviews = getCache(cacheKey);
    if (cachedReviews) {
      console.log('请求失败，使用缓存数据');
      return { reviews: cachedReviews, error: null };
    }
    return { 
      reviews: [], 
      error: handleApiError(error, '获取最新评论失败，请稍后重试') 
    };
  }
};

/**
 * 获取课程评论
 * @param {number} courseId - 课程ID
 * @param {Object} options - 选项对象
 * @param {boolean} options.forceRefresh - 是否强制刷新缓存
 * @returns {Promise<Object>} - 包含评论数组、分页信息和错误信息的对象
 */
export const getCourseReviews = async (courseId, options = {}) => {
  try {
    const { 
      forceRefresh = false
    } = options;
    
    // 简化缓存键，不再使用查询参数
    const cacheKey = `${CACHE_KEYS.COURSE_REVIEWS}_${courseId}`;
    
    // 检查是否有缓存且不需要强制刷新
    // 注意：现在isPageRefresh只在真正的页面刷新或手动触发时返回true
    if (!forceRefresh && !isPageRefresh()) {
      const cachedData = getCache(cacheKey);
      if (cachedData) {
        console.log(`使用缓存的课程评论 (ID: ${courseId})`);
        return { ...cachedData, error: null };
      }
    }
    
    // 没有缓存或需要强制刷新，从API获取
    console.log(`从API获取课程评论 (ID: ${courseId})`);
    // 直接使用基本URL，不添加查询参数
    const response = await api.get(`/reviews/course/${courseId}`);
    
    const result = {
      reviews: response.data.reviews || [],
      pagination: response.data.pagination || {
        total: 0,
        page: 1,
        limit: 100,
        totalPages: 0
      }
    };
    
    // 缓存数据，使用较短的过期时间
    setCache(cacheKey, result, 5 * 60 * 1000); // 5分钟过期
    
    return { ...result, error: null };
  } catch (error) {
    console.error(`获取课程评论失败 (ID: ${courseId}):`, error);
    // 如果有缓存，在请求失败时返回缓存数据
    const cacheKey = `${CACHE_KEYS.COURSE_REVIEWS}_${courseId}`;
    const cachedData = getCache(cacheKey);
    if (cachedData) {
      console.log(`请求失败，使用缓存的课程评论 (ID: ${courseId})`);
      return { ...cachedData, error: null };
    }
    return { 
      reviews: [], 
      pagination: {
        total: 0,
        page: 1,
        limit: 100,
        totalPages: 0
      },
      error: handleApiError(error, '获取课程评论失败，请稍后重试') 
    };
  }
};

/**
 * 获取评论详情
 * @param {number} reviewId - 评论ID
 * @param {boolean} forceRefresh - 是否强制刷新缓存
 * @returns {Promise<Object>} - 包含评论详情和错误信息的对象
 */
export const getReviewDetails = async (reviewId, forceRefresh = false) => {
  try {
    const cacheKey = `${CACHE_KEYS.REVIEW_DETAILS}${reviewId}`;
    
    // 检查是否有缓存且不需要强制刷新，且不是页面刷新
    if (!forceRefresh && !isPageRefresh()) {
      const cachedReview = getCache(cacheKey);
      if (cachedReview) {
        console.log(`使用缓存的评论详情 (ID: ${reviewId})`);
        return { review: cachedReview, error: null };
      }
    }
    
    // 没有缓存或需要强制刷新，从API获取
    console.log(`从API获取评论详情 (ID: ${reviewId})`);
    const response = await api.get(`/reviews/${reviewId}`);
    const review = response.data;
    
    // 缓存数据
    setCache(cacheKey, review);
    
    return { review, error: null };
  } catch (error) {
    console.error(`获取评论详情失败 (ID: ${reviewId}):`, error);
    // 如果有缓存，在请求失败时返回缓存数据
    const cacheKey = `${CACHE_KEYS.REVIEW_DETAILS}${reviewId}`;
    const cachedReview = getCache(cacheKey);
    if (cachedReview) {
      console.log(`请求失败，使用缓存的评论详情 (ID: ${reviewId})`);
      return { review: cachedReview, error: null };
    }
    return { 
      review: null, 
      error: handleApiError(error, '获取评论详情失败，请稍后重试') 
    };
  }
};

/**
 * 批量获取评论详情
 * @param {number[]} reviewIds - 评论ID数组
 * @param {boolean} forceRefresh - 是否强制刷新缓存
 * @returns {Promise<Object>} - 评论详情映射对象
 */
export const batchGetReviewDetails = async (reviewIds, forceRefresh = false) => {
  if (!reviewIds || reviewIds.length === 0) {
    return { reviews: {}, errors: {} };
  }
  
  // 使用Promise.all并行获取所有评论详情
  const results = await Promise.all(
    reviewIds.map(id => getReviewDetails(id, forceRefresh))
  );
  
  // 整理结果
  const reviews = {};
  const errors = {};
  
  results.forEach((result, index) => {
    const reviewId = reviewIds[index];
    if (result.review) {
      reviews[reviewId] = result.review;
    }
    if (result.error) {
      errors[reviewId] = result.error;
    }
  });
  
  return { reviews, errors };
};

/**
 * 提交评论
 * @param {Object} reviewData - 评论数据
 * @returns {Promise<Object>} - 包含提交结果和错误信息的对象
 */
export const submitReview = async (reviewData) => {
  try {
    const response = await api.post('/reviews', reviewData);
    
    // 清除相关缓存
    clearCache(CACHE_KEYS.LATEST_REVIEWS);
    if (reviewData.courseId) {
      clearCache(`${CACHE_KEYS.COURSE_REVIEWS}_${reviewData.courseId}`);
    }
    
    return { success: true, data: response.data, error: null };
  } catch (error) {
    console.error('提交评论失败:', error);
    return { 
      success: false, 
      data: null, 
      error: handleApiError(error, '提交评论失败，请稍后重试') 
    };
  }
};

/**
 * 对评论投票
 * @param {number} reviewId - 评论ID
 * @param {Object} voteData - 投票数据
 * @param {string} voteData.voteType - 投票类型 ('like' 或 'dislike')
 * @param {string} voteData.browserFingerprint - 浏览器指纹
 * @returns {Promise<Object>} - 包含投票结果和错误信息的对象
 */
export const voteReview = async (reviewId, voteData) => {
  try {
    const response = await api.post(`/reviews/${reviewId}/vote`, voteData);
    
    // 清除相关缓存
    clearCache(`${CACHE_KEYS.REVIEW_DETAILS}${reviewId}`);
    
    return { success: true, data: response.data, error: null };
  } catch (error) {
    console.error(`对评论投票失败 (ID: ${reviewId}):`, error);
    return { 
      success: false, 
      data: null, 
      error: handleApiError(error, '投票失败，请稍后重试') 
    };
  }
};

/**
 * 删除评论
 * @param {number} reviewId - 评论ID
 * @param {number} courseId - 课程ID
 * @returns {Promise<Object>} - 包含删除结果和错误信息的对象
 */
export const deleteReview = async (reviewId, courseId) => {
  try {
    const response = await api.delete(`/reviews/${reviewId}`);
    
    // 清除相关缓存
    clearCache(CACHE_KEYS.LATEST_REVIEWS);
    clearCache(`${CACHE_KEYS.COURSE_REVIEWS}_${courseId}`);
    clearCache(`${CACHE_KEYS.REVIEW_DETAILS}${reviewId}`);
    
    return { success: true, data: response.data, error: null };
  } catch (error) {
    console.error(`删除评论失败 (ID: ${reviewId}):`, error);
    return { 
      success: false, 
      data: null, 
      error: handleApiError(error, '删除评论失败，请稍后重试') 
    };
  }
};

// 默认导出所有方法
export default {
  getLatestReviews,
  getCourseReviews,
  getReviewDetails,
  batchGetReviewDetails,
  submitReview,
  voteReview,
  deleteReview
}; 