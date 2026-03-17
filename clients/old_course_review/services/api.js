import axios from 'axios';
import errorUtils from '../utils/errorUtils';

const { handleApiError } = errorUtils;

// 创建axios实例
const api = axios.create({
  baseURL: process.env.REACT_APP_API_URL || 'http://localhost:5000/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
});

// 请求拦截器
api.interceptors.request.use(
  config => {
    // 在发送请求之前做些什么
    console.log(`请求: ${config.method.toUpperCase()} ${config.url}`);
    
    // 添加请求重试计数
    config.retryCount = config.retryCount || 0;
    
    return config;
  },
  error => {
    // 对请求错误做些什么
    console.error('请求错误:', error);
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  response => {
    // 对响应数据做点什么
    console.log(`响应: ${response.status} ${response.config.url}`);
    return response;
  },
  async error => {
    // 获取错误配置和响应
    const { config, response } = error;
    
    // 如果没有配置，说明请求没有发出去，直接拒绝
    if (!config) {
      return Promise.reject(error);
    }
    
    // 设置最大重试次数
    const maxRetries = 2;
    
    // 如果是网络错误或5xx错误，并且没有超过最大重试次数，则重试
    if (
      (!response || (response && response.status >= 500)) && 
      config.retryCount < maxRetries
    ) {
      // 增加重试计数
      config.retryCount += 1;
      
      // 设置重试延迟
      const delay = 1000; // 1秒
      
      console.log(`重试请求 (${config.retryCount}/${maxRetries}): ${config.url}`);
      
      // 延迟后重试
      return new Promise(resolve => {
        setTimeout(() => resolve(api(config)), delay);
      });
    }
    
    // 对响应错误做点什么
    console.error('响应错误:', error);
    
    // 使用错误处理工具处理错误
    return Promise.reject(error);
  }
);

// 导出错误处理函数
export { handleApiError };

// 导出api实例
export default api; 
