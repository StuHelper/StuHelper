/**
 * 错误类型枚举
 */
export const ERROR_TYPES = {
  NETWORK: 'NETWORK_ERROR',
  API: 'API_ERROR',
  VALIDATION: 'VALIDATION_ERROR',
  AUTH: 'AUTH_ERROR',
  NOT_FOUND: 'NOT_FOUND_ERROR',
  SERVER: 'SERVER_ERROR',
  CLIENT: 'CLIENT_ERROR',
  UNKNOWN: 'UNKNOWN_ERROR'
};

/**
 * 创建标准错误对象
 * @param {string} message - 错误消息
 * @param {string} type - 错误类型
 * @param {*} details - 错误详情
 * @returns {Object} - 标准错误对象
 */
export const createError = (message, type = ERROR_TYPES.UNKNOWN, details = null) => {
  return {
    message,
    type,
    details,
    timestamp: new Date().toISOString()
  };
};

/**
 * 处理API错误
 * @param {Error} error - 错误对象
 * @param {string} fallbackMessage - 默认错误消息
 * @returns {Object} - 标准错误对象
 */
export const handleApiError = (error, fallbackMessage = '发生错误，请稍后重试') => {
  // 如果已经是标准错误对象，直接返回
  if (error && error.type && Object.values(ERROR_TYPES).includes(error.type)) {
    return error;
  }
  
  // 处理Axios错误
  if (error && error.response) {
    const { status, data } = error.response;
    
    // 提取错误消息
    const errorMessage = data?.message || data?.error || fallbackMessage;
    
    // 根据状态码确定错误类型
    switch (status) {
      case 400:
        return createError(errorMessage, ERROR_TYPES.VALIDATION, data);
      case 401:
      case 403:
        return createError(errorMessage, ERROR_TYPES.AUTH, data);
      case 404:
        return createError(errorMessage, ERROR_TYPES.NOT_FOUND, data);
      case 500:
      case 502:
      case 503:
      case 504:
        return createError(errorMessage, ERROR_TYPES.SERVER, data);
      default:
        return createError(errorMessage, ERROR_TYPES.API, data);
    }
  }
  
  // 处理网络错误
  if (error && error.request) {
    return createError(
      '无法连接到服务器，请检查网络连接',
      ERROR_TYPES.NETWORK,
      error.request
    );
  }
  
  // 处理其他错误
  return createError(
    error?.message || fallbackMessage,
    ERROR_TYPES.UNKNOWN,
    error
  );
};

/**
 * 处理表单验证错误
 * @param {Object} errors - 表单错误对象
 * @returns {Object} - 标准错误对象
 */
export const handleValidationError = (errors) => {
  const errorMessages = Object.values(errors)
    .filter(Boolean)
    .map(error => error.message || error)
    .join('; ');
  
  return createError(
    errorMessages || '表单验证失败，请检查输入',
    ERROR_TYPES.VALIDATION,
    errors
  );
};

/**
 * 记录错误
 * @param {Object} error - 错误对象
 * @param {string} context - 错误上下文
 */
export const logError = (error, context = '') => {
  // 在生产环境中，可以将错误发送到错误跟踪服务
  if (process.env.NODE_ENV === 'production') {
    // 这里可以集成Sentry等错误跟踪服务
    console.error(`[${context}]`, error);
  } else {
    // 在开发环境中，打印详细错误信息
    console.error(`[${context}]`, error);
  }
};

/**
 * 获取用户友好的错误消息
 * @param {Object} error - 错误对象
 * @param {Object} options - 选项
 * @param {boolean} options.includeDetails - 是否包含详细信息
 * @param {Object} options.typeMessages - 错误类型对应的消息
 * @returns {string} - 用户友好的错误消息
 */
export const getFriendlyErrorMessage = (error, options = {}) => {
  const { includeDetails = false, typeMessages = {} } = options;
  
  // 如果不是标准错误对象，转换为标准错误对象
  const standardError = error.type 
    ? error 
    : handleApiError(error);
  
  // 根据错误类型获取默认消息
  const defaultTypeMessages = {
    [ERROR_TYPES.NETWORK]: '网络连接错误，请检查您的网络连接',
    [ERROR_TYPES.API]: '服务请求失败，请稍后重试',
    [ERROR_TYPES.VALIDATION]: '输入数据无效，请检查您的输入',
    [ERROR_TYPES.AUTH]: '身份验证失败，请重新登录',
    [ERROR_TYPES.NOT_FOUND]: '请求的资源不存在',
    [ERROR_TYPES.SERVER]: '服务器错误，请稍后重试',
    [ERROR_TYPES.CLIENT]: '客户端错误，请刷新页面重试',
    [ERROR_TYPES.UNKNOWN]: '发生未知错误，请稍后重试'
  };
  
  // 合并默认消息和自定义消息
  const mergedTypeMessages = { ...defaultTypeMessages, ...typeMessages };
  
  // 获取错误消息
  let message = standardError.message || mergedTypeMessages[standardError.type] || defaultTypeMessages[ERROR_TYPES.UNKNOWN];
  
  // 如果需要包含详细信息且有详细信息
  if (includeDetails && standardError.details) {
    const details = typeof standardError.details === 'string' 
      ? standardError.details 
      : JSON.stringify(standardError.details);
    
    message = `${message} (${details})`;
  }
  
  return message;
};

export default {
  ERROR_TYPES,
  createError,
  handleApiError,
  handleValidationError,
  logError,
  getFriendlyErrorMessage
}; 