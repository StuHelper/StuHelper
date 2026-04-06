/**
 * 教学中心文本 - 中文
 */
export default {
  title: '教学中心',
  subtitle: '北航教学资源与评价平台',
  modules: {
    review: {
      name: '评课',
      description: '查看课程评价，分享你的学习体验'
    },
    teacher: {
      name: '教师',
      description: '了解教师教学风格与评分'
    },
    spoc: {
      name: '智学北航',
      description: 'SPOC 课程资源与学习辅助'
    },
    resource: {
      name: '资源',
      description: '课件、笔记、历年试题共享'
    }
  },
  comingSoon: '即将上线',
  stats: {
    courses: '{count} 门课程',
    reviews: '{count} 条评价'
  },
  profile: {
    overallRating: '综合评分',
    avgRating: '平均评分',
    courseCount: '授课数',
    reviewCount: '评价数',
    viewDetail: '查看详情',
    ratingTrend: '评分趋势',
    ratingTrendChartAria: '教师评分趋势折线图',
    courseList: '授课列表',
    reviewsCount: '{count} 条评价',
    notFound: '未找到教师信息',
    backToHome: '返回首页',
    ratingLabel: '评分',
    trend: {
      stable: '持平',
      up: '上升',
      down: '下降'
    }
  },
  hub: {
    title: '教师主页',
    description: '搜索教师、查看评分和评价',
    searchPlaceholder: '输入教师姓名搜索...',
    popular: '热门教师',
    searchResults: '搜索结果',
    reviewUnit: ' 条评价',
    noResults: '未找到匹配的教师，换个名字试试。',
    noTeachers: '暂无教师数据。',
    actions: {
      browseCourses: '先去看课程列表',
      backToReview: '返回评课中心'
    },
    highlights: {
      entry: {
        title: '教师资料入口',
        description: '教师详情页已经接入路由，后续会补完整的教师列表、搜索与筛选入口。'
      },
      capabilities: {
        title: '教学相关能力',
        description: '该模块会逐步承接教师画像、课程关联、评分维度和统计展示。'
      },
      status: {
        title: '当前状态',
        description: '教师模块主页仍在实现中，因此这里先提供占位说明，而不再错误跳转到课程列表。'
      }
    }
  }
}
