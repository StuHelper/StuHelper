/**
 * 页面标题管理工具
 * 用于动态设置页面标题，支持基础标题后缀和页面特定标题
 */

const BASE_TITLE = '评课社区@BUAA';

/**
 * 设置页面标题
 * @param {string} pageTitle - 页面特定标题，为空则只显示基础标题
 */
export const setDocumentTitle = (pageTitle) => {
  if (!pageTitle) {
    document.title = BASE_TITLE;
  } else {
    document.title = `${pageTitle} - ${BASE_TITLE}`;
  }
};

/**
 * 预定义页面标题
 * 常用页面的标题映射
 */
export const PAGE_TITLES = {
  HOME: '',  // 首页只显示基础标题
  COURSE_LIST: '课程列表',
  LATEST_REVIEWS: '最新测评',
  SEARCH: '搜索',
  ABOUT: '关于我们',
  REVIEW_POST: '发布评价'
};

export default {
  setDocumentTitle,
  PAGE_TITLES,
}; 